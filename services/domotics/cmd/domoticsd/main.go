package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	// ErrUnableToSetupMonopAmp is returned if the supplied bridge configuration fails to properly initialize monop.
	ErrUnableToSetupMonopAmp = errors.New("unable to set up monoprice amp")
	// ErrBridgeConfigInvalid is returned if the supplied bridge configuration is invalid.
	ErrBridgeConfigInvalid = errors.New("bridge config invalid")
	portEnvVar             = "PORT"
	monopUSBPathEnvVar     = "MONOP_USB_PATH"
	monopCachePathEnvVar   = "MONOP_CACHE_PATH"
	hueCachePathEnvVar     = "HUE_CACHE_PATH"
	proxyToAddrEnvVar      = "PROXY_TO_ADDR"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(portEnvVar)
	viper.BindEnv(monopUSBPathEnvVar)
	viper.BindEnv(monopCachePathEnvVar)
	viper.BindEnv(hueCachePathEnvVar)
	viper.BindEnv(proxyToAddrEnvVar)

	hub := domotics.NewHub(logger)

	var toClose []io.Closer

	monopUSBPath := viper.GetString(monopUSBPathEnvVar)
	if len(monopUSBPath) > 0 {
		monopConfig := &domotics.BridgeConfig{
			Name:      domotics.BridgeType_MONOPRICEAMP.String(),
			CachePath: viper.GetString(monopCachePathEnvVar),
			Address: &domotics.Address{
				Usb: &domotics.Address_Usb{
					Path: monopUSBPath,
				},
			},
		}

		monop := &monopampImpl{
			logger: logger,
		}
		if err := monop.setup(monopConfig); err != nil {
			logger.Fatal("error initializing module",
				zap.String("module_name", monopConfig.Name),
				zap.Error(err),
			)
		}
		toClose = append(toClose, monop)
		if err = hub.AddBridge(monop.bridge, time.Second*30); err != nil {
			logger.Warn("error adding module to bridge",
				zap.String("module_name", monopConfig.Name),
				zap.Error(err),
			)
		}
	}

	hueCachePath := viper.GetString(hueCachePathEnvVar)
	if len(hueCachePath) > 0 {
		hueConfig := &domotics.BridgeConfig{
			Name:      domotics.BridgeType_HUE.String(),
			CachePath: hueCachePath,
			// The addresses are auto-discovered by the Hue locator
		}

		hue := &hueImpl{
			logger: logger,
			hub:    hub,
		}
		if err := hue.setup(hueConfig); err != nil {
			logger.Fatal("error initializing module",
				zap.String("module_name", hueConfig.Name),
				zap.Error(err),
			)
		}
		toClose = append(toClose, hue)
		go hue.Run() // the bridges are added via Run(), not here.
	}

	proxyAddr := viper.GetString(proxyToAddrEnvVar)
	if len(proxyAddr) > 0 {
		proxyAddrParts := strings.Split(proxyAddr, ":")

		proxyConfig := &domotics.BridgeConfig{
			Name: domotics.BridgeType_PROXY.String(),
			Address: &domotics.Address{
				Ip: &domotics.Address_Ip{
					Host: proxyAddrParts[0],
				},
			},
		}

		if len(proxyAddrParts) < 2 {
			logger.Info("no port supplied for proxy, defaulting to 10102")
			proxyConfig.Address.Ip.Port = 10102
		}
		if proxyAddrPort, err := strconv.ParseInt(proxyAddrParts[1], 10, 32); err == nil {
			proxyConfig.Address.Ip.Port = int32(proxyAddrPort)
		} else {
			logger.Info("error parsing port for proxy, defaulting to 10102")
			proxyConfig.Address.Ip.Port = 10102
		}

		proxy := &proxyImpl{
			logger: logger,
		}
		if err := proxy.setup(proxyConfig, hub); err != nil {
			logger.Fatal("error initializing module",
				zap.String("module_name", proxyConfig.Name),
				zap.Error(err),
			)
		}
		toClose = append(toClose, proxy)
		// the proxyImpl setup adds itself to the hub
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

	api := domotics.NewAPI(logger, hub)

	grpcServer := grpc.NewServer()
	domotics.RegisterBridgeServiceServer(grpcServer, api)
	domotics.RegisterDeviceServiceServer(grpcServer, api)
	grpcServer.Serve(lis)

	for _, impl := range toClose {
		impl.Close()
	}
}
