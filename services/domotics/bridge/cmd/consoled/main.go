package main

import (
	"context"
	"fmt"
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

	br := NewConsole(logger, viper.GetString(idEnvVar))

	connStr := fmt.Sprintf("%s:%d", "127.0.0.1", viper.GetInt(portEnvVar))
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

	ad := bridge.NewAdvertiser(logger, viper.GetString(idEnvVar), connStr)
	go ad.Run()
	defer ad.Shutdown()

	grpcServer := grpc.NewServer()
	bridge.RegisterBridgeServiceServer(grpcServer, sbs)
	bridge.RegisterPingServiceServer(grpcServer, ad)
	grpcServer.Serve(lis)
}
