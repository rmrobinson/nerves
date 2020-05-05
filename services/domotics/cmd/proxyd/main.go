package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/rmrobinson/nerves/services/domotics"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	var (
		port      = flag.Int("port", 1338, "Port to listen on")
		proxyAddr = flag.String("proxy", "", "Address to proxy requests to")
	)
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	// Setup the proxy connection first
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(*proxyAddr, opts...)

	if err != nil {
		logger.Fatal("error initializing proxy connection",
			zap.String("proxy_addr", *proxyAddr),
			zap.Error(err),
		)
	}

	logger.Info("proxying",
		zap.String("proxy_addr", *proxyAddr),
	)

	// Setup the hub and proxy once we have a connected remote.
	hub := domotics.NewHub(logger)

	p := domotics.NewProxyBridge(logger, hub, conn)
	go p.Run()

	connStr := fmt.Sprintf("%s:%d", "", *port)
	lis, err := net.Listen("tcp", connStr)
	if err != nil {
		logger.Fatal("error initializing listener",
			zap.Error(err),
		)
	}
	defer lis.Close()
	logger.Info("listening",
		zap.String("local_addr", connStr),
	)

	api := domotics.NewAPI(logger, hub, nil)

	grpcServer := grpc.NewServer()
	domotics.RegisterBridgeServiceServer(grpcServer, api)
	domotics.RegisterDeviceServiceServer(grpcServer, api)
	grpcServer.Serve(lis)
}
