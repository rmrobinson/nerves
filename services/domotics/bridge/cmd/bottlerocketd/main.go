package main

import (
	"context"
	"net"

	br "github.com/rmrobinson/bottlerocket-go"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	brUSBPathEnvVar = "BOTTLEROCKET_USB_PATH"
	idEnvVar        = "ID"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(idEnvVar)
	viper.BindEnv(brUSBPathEnvVar)

	brUSBPath := viper.GetString(brUSBPathEnvVar)
	if len(brUSBPath) < 1 {
		logger.Fatal("usb path missing")
	}

	brh := &br.Bottlerocket{}
	err = brh.Open(brUSBPath)
	if err != nil {
		logger.Fatal("error initializing bottlerocket port",
			zap.String("port_path", brUSBPath),
			zap.Error(err),
		)
	}

	br := NewBottlerocket(logger, viper.GetString(idEnvVar), brh)

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
