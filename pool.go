// MIT License
//
// Copyright (c) 2023 VegetableDoggies

package fabric_gateway_pool_go

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
)

type HlfGWPool struct {
	conf              *Options
	poolSize          int
	idleConnections   chan *client.Gateway
	heartbeatCheck    bool
	heartbeatInterval time.Duration
	factory           func(options *Options) (*client.Gateway, error)
	curSize           int32
	close             int32
	creating          int32
}

func NewHlfGWPool(option *Options) (*HlfGWPool, error) {
	if option.PoolSize <= 0 {
		return nil, fmt.Errorf("pool size must be greater than 0")
	}

	pool := &HlfGWPool{
		poolSize:          option.PoolSize,
		idleConnections:   make(chan *client.Gateway, option.PoolSize),
		heartbeatCheck:    option.HeartbeatCheck,
		heartbeatInterval: option.HeartbeatInterval,
		factory:           NewGateway,
		conf:              option,
		curSize:           0,
		close:             0,
	}

	for i := 0; i < option.PoolSize; i++ {
		conn, err := pool.factory(option)
		if err != nil {
			return nil, fmt.Errorf("failed to create connection: %v", err)
		}
		pool.idleConnections <- conn
		atomic.AddInt32(&pool.curSize, 1)
	}

	if pool.heartbeatCheck {
		pool.startHeartbeatCheck()
	}

	return pool, nil
}

func (pool *HlfGWPool) startHeartbeatCheck() {

	go func() {
		ticker := time.NewTicker(pool.heartbeatInterval)
		defer ticker.Stop()

		for {
			if pool.close > 0 {
				runtime.Goexit()
				return
			}
			select {
			case <-ticker.C:
				pool.checkHeartbeat()
			}
		}
	}()
}

// checkHeartbeat checks the heartbeat of connections and closes invalid connections
func (pool *HlfGWPool) checkHeartbeat() {
	if pool.heartbeatCheck {
		for conn := range pool.idleConnections {
			if conn.GetNetwork(pool.conf.ChannelName) == nil {
				atomic.AddInt32(&pool.curSize, -1)
				err := conn.Close()
				if err != nil {
					continue
				}
			}
			pool.idleConnections <- conn
		}
	}
}

// GetConnection retrieves a Fabric Gateway connection from the pool
func (pool *HlfGWPool) GetConnection() (*client.Gateway, error) {
	select {
	case conn := <-pool.idleConnections:
		return conn, nil
	default:
		if pool.curSize < int32(pool.poolSize) {
			if atomic.CompareAndSwapInt32(&pool.creating, 0, 1) {
				conn, err := pool.factory(pool.conf)
				if err != nil {
					return nil, fmt.Errorf("failed to create connection: %v", err)
				}
				atomic.AddInt32(&pool.curSize, 1)
				atomic.StoreInt32(&pool.creating, 0)
				return conn, nil
			} else {
				return pool.GetConnection()
			}
		} else {
			return pool.GetConnection()
		}
	}
}

// ReleaseConnection releases a Fabric Gateway connection back to the pool
func (pool *HlfGWPool) ReleaseConnection(conn *client.Gateway) {
	pool.idleConnections <- conn
}

// Close releases all resources used by the connection pool
func (pool *HlfGWPool) Close() (err error) {
	atomic.AddInt32(&pool.close, 1)
	for conn := range pool.idleConnections {
		if conn != nil {
			atomic.AddInt32(&pool.curSize, -1)
			err = conn.Close()
			continue
		}
	}

	atomic.StoreInt32(&pool.curSize, 0)
	close(pool.idleConnections)
	return err
}
