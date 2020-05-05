package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	br "github.com/rmrobinson/bottlerocket-go"
	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	brUSBPathEnvVar   = "BOTTLEROCKET_USB_PATH"
	brCachePathEnvVar = "BOTTLEROCKET_CACHE_PATH"
	portEnvVar        = "PORT"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(portEnvVar)
	viper.BindEnv(brUSBPathEnvVar)
	viper.BindEnv(brCachePathEnvVar)

	hub := domotics.NewHub(logger)

	brUSBPath := viper.GetString(brUSBPathEnvVar)
	if len(brUSBPath) < 1 {
		logger.Fatal("usb path missing")
	}

	brConfig := &domotics.BridgeConfig{
		Name:      domotics.BridgeType_BOTTLEROCKET.String(),
		CachePath: viper.GetString(brCachePathEnvVar),
		Address: &domotics.Address{
			Usb: &domotics.Address_Usb{
				Path: brUSBPath,
			},
		},
	}

	setupNeeded := false
	if _, err := os.Stat(brConfig.CachePath); os.IsNotExist(err) {
		setupNeeded = true
	}

	db := &bridge.DB{}
	err = db.Open(brConfig.CachePath)
	if err != nil {
		logger.Fatal("error opening db cache",
			zap.Error(err),
		)
	}

	brh := &br.Bottlerocket{}
	err = brh.Open(brConfig.Address.Usb.Path)
	if err != nil {
		logger.Fatal("error initializing bottlerocket port",
			zap.String("port_path", brConfig.Address.Usb.Path),
			zap.Error(err),
		)
	}

	bridge := NewBottlerocket(logger, brh, db)

	if setupNeeded {
		err = bridge.Setup(context.Background())
		if err != nil {
			logger.Fatal("unable to setup",
				zap.Error(err),
			)
		}
	}

	if err = hub.AddBridge(bridge, time.Second*30); err != nil {
		logger.Warn("error adding module to bridge",
			zap.String("module_name", brConfig.Name),
			zap.Error(err),
		)
	}

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

	api := domotics.NewAPI(logger, hub, nil)

	grpcServer := grpc.NewServer()
	domotics.RegisterBridgeServiceServer(grpcServer, api)
	domotics.RegisterDeviceServiceServer(grpcServer, api)
	grpcServer.Serve(lis)
}
