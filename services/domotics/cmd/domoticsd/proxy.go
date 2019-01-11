package main

import (
	"fmt"

	"github.com/rmrobinson/nerves/services/domotics"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type proxyImpl struct {
	conn *grpc.ClientConn

	logger *zap.Logger

	p *domotics.ProxyHub
}

func (b *proxyImpl) setup(config *domotics.BridgeConfig, hub *domotics.Hub) error {
	if config.Address.Ip == nil {
		return ErrBridgeConfigInvalid
	}

	var err error
	addr := fmt.Sprintf("%s:%d", config.Address.Ip.Host, config.Address.Ip.Port)
	b.logger.Info("proxying requests",
		zap.String("proxy_addr", addr),
	)

	// Setup the proxyImpl connection first
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	b.conn, err = grpc.Dial(addr, opts...)
	if err != nil {
		b.logger.Warn("error initializing proxy connection",
			zap.String("proxy_addr", addr),
			zap.Error(err),
		)
		return err
	}

	b.p = domotics.NewProxyBridge(b.logger, hub, b.conn)
	go b.p.Run()

	return nil
}

// Close cleans up any open resources.
func (b *proxyImpl) Close() error {
	var connErr error
	if b.conn != nil {
		connErr = b.conn.Close()
	}
	return connErr
}
