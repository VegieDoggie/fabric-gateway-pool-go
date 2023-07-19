// MIT License
//
// Copyright (c) 2023 VegetableDoggies

package fabric_gateway_pool_go

import (
	"context"
	"crypto/x509"
	"fmt"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"os"
	"path"
)

func newGrpcConnection(option *Options) (*grpc.ClientConn, error) {
	certificate, err := loadCertificate(option.TlsCertPath)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, option.GatewayPeer)

	connection, err := grpc.DialContext(context.Background(), option.PeerEndpoint,
		grpc.WithTransportCredentials(transportCredentials),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                option.GrpcInterval, // 客户端发送心跳的时间间隔
			Timeout:             option.GrpcTimeout,  // 如果在此时间内没有收到响应，将断开连接
			PermitWithoutStream: true,                // 允许在没有活动流的情况下发送心跳
		}))

	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	return connection, nil
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	certificatePEM, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}
	return identity.CertificateFromPEM(certificatePEM)
}

func newSign(option *Options) (identity.Sign, error) {

	files, err := os.ReadDir(option.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key directory: %w", err)
	}
	privateKeyPEM, err := os.ReadFile(path.Join(option.KeyPath, files[0].Name()))

	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, err
	}

	return sign, nil
}

func newIdentity(option *Options) (*identity.X509Identity, error) {
	certificate, err := loadCertificate(option.CertPath)
	if err != nil {
		return nil, err
	}

	id, err := identity.NewX509Identity(option.MspID, certificate)
	if err != nil {
		return nil, err
	}

	return id, nil
}

func NewGateway(option *Options) (*client.Gateway, error) {
	grpcConn, err := newGrpcConnection(option)
	if err != nil {
		return nil, err
	}

	signer, err := newSign(option)
	if err != nil {
		return nil, err
	}

	id, err := newIdentity(option)
	if err != nil {
		return nil, err
	}

	cli, err := client.Connect(
		id,
		client.WithClientConnection(grpcConn),
		client.WithSign(signer),
		client.WithEndorseTimeout(option.EndorseTimeout),
		client.WithEvaluateTimeout(option.EvaluateTimeout),
		client.WithCommitStatusTimeout(option.CommitTimeout),
		client.WithSubmitTimeout(option.SubmitTimeout),
	)

	if err != nil {
		return nil, err
	}
	return cli, nil
}
