package main

import (
	"context"
	"net"

	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	idEnvVar   = "ID"
	portEnvVar = "PORT"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(idEnvVar)
	viper.BindEnv(portEnvVar)

	brInfo := &bridge.Bridge{
		Id:           viper.GetString(idEnvVar),
		ModelId:      "hprox1",
		ModelName:    "Proxy",
		Manufacturer: "Faltung Systems",
	}

	hub := bridge.NewHub(logger)

	hm := &HubMonitor{
		logger: logger,
		conns:  map[string]*grpc.ClientConn{},
		hub:    hub,
		brInfo: brInfo,
	}
	m := bridge.NewMonitor(logger, hm, []string{"falnet_nerves:bridge"})

	logger.Info("listening for bridges")
	go m.Run(context.Background())

	lis, err := net.Listen("tcp", "0.0.0.0:"+viper.GetString(portEnvVar))
	if err != nil {
		logger.Fatal("error initializing listener",
			zap.Error(err),
		)
	}
	defer lis.Close()
	logger.Info("listening",
		zap.String("local_addr", lis.Addr().String()),
	)

	ad := bridge.NewAdvertiser(logger, hm.brInfo.Id, lis.Addr().String())
	go ad.Run()
	defer ad.Shutdown()

	grpcServer := grpc.NewServer()
	bridge.RegisterBridgeServiceServer(grpcServer, hm)
	bridge.RegisterPingServiceServer(grpcServer, ad)
	grpcServer.Serve(lis)
}
