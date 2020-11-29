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
	idEnvVar = "ID"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(idEnvVar)

	br := NewConsole(logger, viper.GetString(idEnvVar))

	lis, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		logger.Fatal("error initializing listener",
			zap.Error(err),
		)
	}
	defer lis.Close()
	logger.Info("listening",
		zap.String("local_addr", lis.Addr().String()),
	)

	brInfo, err := br.getBridge(context.Background())
	if err != nil {
		logger.Fatal("error getting bridge info",
			zap.Error(err),
		)
	}

	devices, err := br.getDevices(context.Background())
	if err != nil {
		logger.Fatal("error getting devices info",
			zap.Error(err),
		)
	}

	sbs := bridge.NewSyncBridgeService(logger, brInfo, devices, br)

	ad := bridge.NewAdvertiser(logger, viper.GetString(idEnvVar), lis.Addr().String())
	go ad.Run()
	defer ad.Shutdown()

	grpcServer := grpc.NewServer()
	bridge.RegisterBridgeServiceServer(grpcServer, sbs)
	bridge.RegisterPingServiceServer(grpcServer, ad)
	grpcServer.Serve(lis)
}
