package main

import (
	"fmt"
	"log"

	"github.com/rmrobinson/nerves/services/domotics"
	"google.golang.org/grpc"
)

type proxyImpl struct {
	conn *grpc.ClientConn
	p *domotics.ProxyHub
}

func (b *proxyImpl) setup(config *domotics.BridgeConfig, hub *domotics.Hub) error {
	if config.Address.Ip == nil {
		return ErrBridgeConfigInvalid
	}

	var err error
	addr := fmt.Sprintf("%s:%d", config.Address.Ip.Host, config.Address.Ip.Port)
	log.Printf("Proxying requests to %s\n", addr)

	// Setup the proxyImpl connection first
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	b.conn, err = grpc.Dial(addr, opts...)
	if err != nil {
		log.Printf("Error initializing proxyImpl connection to %s: %s\n", addr, err.Error())
		return err
	}

	b.p = domotics.NewProxyBridge(hub, b.conn)
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
