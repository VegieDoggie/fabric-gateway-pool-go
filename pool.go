// MIT License
//
// Copyright (c) 2023 VegetableDoggies

package fabric_gateway_pool_go

import (
	"fmt"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"runtime"
	"sync"
	"time"
)

type HlfGWPool struct {
	mux               sync.Mutex
	conf              *Options
	poolSize          int
	minActive         int // 最小存活连接数量
	maxIdle           int // 最大空闲连接数量
	connections       []*client.Gateway
	idleConnections   chan *client.Gateway
	heartbeatCheck    bool
	heartbeatInterval time.Duration
	closeCh           chan struct{}
	factory           func(options *Options) (*client.Gateway, error)
}

func NewHlfGWPool(option *Options) (*HlfGWPool, error) {
	if option.PoolSize <= 0 {
		return nil, fmt.Errorf("pool size must be greater than 0")
	}

	if option.MinActive < 0 || option.MinActive > option.PoolSize {
		return nil, fmt.Errorf("minimum active connections must be between 0 and pool size")
	}

	if option.MaxIdle < option.MinActive {
		return nil, fmt.Errorf("maximum idle connections must be greater than or equal to minimum active connections")
	}

	pool := &HlfGWPool{
		poolSize:          option.PoolSize,
		minActive:         option.MinActive,
		maxIdle:           option.MaxIdle,
		connections:       make([]*client.Gateway, option.PoolSize),
		idleConnections:   make(chan *client.Gateway, option.MaxIdle),
		heartbeatCheck:    option.HeartbeatCheck,
		heartbeatInterval: option.HeartbeatInterval,
		factory:           NewGateway,
		conf:              option,
	}

	for i := 0; i < option.PoolSize; i++ {
		conn, err := pool.factory(option)
		if err != nil {
			return nil, fmt.Errorf("failed to create connection: %v", err)
		}

		pool.connections[i] = conn
		pool.idleConnections <- conn
	}

	if pool.heartbeatCheck {
		pool.startHeartbeatCheck()
	}

	return pool, nil
}

func (pool *HlfGWPool) startHeartbeatCheck() {
	pool.closeCh = make(chan struct{})

	go func() {
		ticker := time.NewTicker(pool.heartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-pool.closeCh:
				runtime.Goexit()
				return
			case <-ticker.C:
				pool.checkHeartbeat()
			}
		}
	}()
}

// checkHeartbeat checks the heartbeat of connections and closes invalid connections
func (pool *HlfGWPool) checkHeartbeat() {
	pool.mux.Lock()
	defer pool.mux.Unlock()

	for i, conn := range pool.connections {
		// TODO: 修改检测方式
		if conn.GetNetwork(pool.conf.ChannelName) == nil {
			pool.connections[i] = nil
			err := conn.Close()
			if err != nil {
				panic(err)
			}
		}

		if pool.idleConnections != nil && len(pool.idleConnections) > pool.maxIdle {
			pool.connections[i] = nil
			idleConn := <-pool.idleConnections
			err := idleConn.Close()
			if err != nil {
				panic(err)
			}
		}
	}

	compactConnections := make([]*client.Gateway, 0, pool.poolSize)
	for _, conn := range pool.connections {
		if conn != nil {
			compactConnections = append(compactConnections, conn)
		}
	}

	pool.connections = compactConnections
}

// GetConnection retrieves a Fabric Gateway connection from the pool
func (pool *HlfGWPool) GetConnection() (*client.Gateway, error) {
	conn := <-pool.idleConnections
	if conn != nil {
		return conn, nil
	}

	pool.mux.Lock()
	defer pool.mux.Unlock()

	if len(pool.idleConnections) > 0 {
		conn = <-pool.idleConnections
		return conn, nil
	}

	conn, err := pool.factory(pool.conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	pool.connections = append(pool.connections, conn)
	return conn, nil
}

// ReleaseConnection releases a Fabric Gateway connection back to the pool
func (pool *HlfGWPool) ReleaseConnection(conn *client.Gateway) {
	pool.idleConnections <- conn
}

// Close releases all resources used by the connection pool
func (pool *HlfGWPool) Close() (err error) {
	pool.mux.Lock()
	defer pool.mux.Unlock()

	for _, conn := range pool.connections {
		if conn != nil {
			err = conn.Close()
			continue
		}
	}

	close(pool.closeCh)
	return err
}
