//MIT License
//
//Copyright (c) 2023 VegetableDoggies

package fabric_gateway_pool_go

import "time"

type Options struct {
	PoolSize          int           `json:"poolSize"`                               // 连接池大小
	HeartbeatCheck    bool          `json:"heartbeatCheck"`                         // 是否开启心跳检测
	HeartbeatInterval time.Duration `json:"heartbeatInterval"`                      // 心跳检测间隔
	MspID             string        `json:"mspID" yaml:"mspID"`                     // mspID
	CertPath          string        `json:"certPath" yaml:"certPath"`               // certPath
	KeyPath           string        `json:"keyPath" yaml:"keyPath"`                 // keyPath
	TlsCertPath       string        `json:"tlsCertPath" yaml:"tlsCertPath"`         // tls证书路径
	PeerEndpoint      string        `json:"peerEndpoint" yaml:"peerEndpoint"`       // peer访问地址
	GatewayPeer       string        `json:"gatewayPeer" yaml:"gatewayPeer"`         // peer id
	GrpcTimeout       time.Duration `json:"grpcTimeout" yaml:"grpcTimeout"`         // grpc心跳超时时间
	GrpcInterval      time.Duration `json:"interval" yaml:"interval"`               // gprc心跳间隔
	EndorseTimeout    time.Duration `json:"endorseTimeout" yaml:"endorseTimeout"`   // 背书超时时间
	SubmitTimeout     time.Duration `json:"submitTimeout" yaml:"submitTimeout"`     // 交易提交时间
	CommitTimeout     time.Duration `json:"commitTimeout" yaml:"commitTimeout"`     // 节点提交超时时间
	EvaluateTimeout   time.Duration `json:"evaluateTimeout" yaml:"evaluateTimeout"` // 查询超时
	ChannelName       string        `json:"channelName" yaml:"channelName"`         // 通道名
}
