package main

import (
	"context"
	"fmt"
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
	portEnvVar      = "PORT"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(idEnvVar)
	viper.BindEnv(portEnvVar)
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

	connStr := fmt.Sprintf("%s:%d", "", viper.GetInt(portEnvVar))
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

	grpcServer := grpc.NewServer()
	bridge.RegisterBridgeServiceServer(grpcServer, sbs)
	grpcServer.Serve(lis)
}
