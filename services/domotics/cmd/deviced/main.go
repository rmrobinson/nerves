package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/rmrobinson/nerves/services/domotics"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

var (
	// ErrUnableToSetupMonopAmp is returned if the supplied bridge configuration fails to properly initialize monop.
	ErrUnableToSetupMonopAmp = errors.New("unable to set up monoprice amp")
	// ErrBridgeConfigInvalid is returned if the supplied bridge configuration is invalid.
	ErrBridgeConfigInvalid = errors.New("bridge config invalid")
)

type rootConfig struct {
	Bridges []bridgeConfig `yaml:"bridges"`
}

type bridgeConfig struct {
	Address   addrConfig `yaml:"address"`
	CachePath string     `yaml:"cachePath"`
	Type      string     `yaml:"type"`
}

type addrConfig struct {
	USBPath   string `yaml:"usbPath"`
	IPAddress string `yaml:"ipAddress"`
	Port      int32  `yaml:"port"`
	Proto     string `yaml:"proto"`
}

func (rc *rootConfig) toProto() ([]*domotics.BridgeConfig, error) {
	var ret []*domotics.BridgeConfig

	for _, bridge := range rc.Bridges {
		config := &domotics.BridgeConfig{
			Name:      bridge.Type,
			CachePath: bridge.CachePath,
			Address:   &domotics.Address{},
		}

		if len(bridge.Address.USBPath) > 0 {
			config.Address.Usb = &domotics.Address_Usb{
				Path: bridge.Address.USBPath,
			}
		} else if len(bridge.Address.IPAddress) > 0 {
			config.Address.Ip = &domotics.Address_Ip{
				Host: bridge.Address.IPAddress,
				Port: bridge.Address.Port,
			}
		}
		ret = append(ret, config)
	}

	return ret, nil
}

func main() {
	var (
		port       = flag.Int("port", 1337, "The port for the deviced process to listen on")
		configPath = flag.String("config", "", "The path to the config file")
	)

	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	yamlFile, err := ioutil.ReadFile(*configPath)
	if err != nil {
		logger.Fatal("error opening config file",
			zap.String("config_path", *configPath),
			zap.Error(err),
		)
	}

	config := rootConfig{}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		logger.Fatal("error parsing config file",
			zap.String("config_path", *configPath),
			zap.Error(err),
		)
	}

	hub := domotics.NewHub(logger)

	bridgeConfigs, err := config.toProto()
	if err != nil {
		logger.Fatal("error with config file format",
			zap.String("config_path", *configPath),
			zap.Error(err),
		)
	}

	var toClose []io.Closer

	for _, bridgeConfig := range bridgeConfigs {
		logger.Info("initializing module",
			zap.String("module_name", bridgeConfig.Name),
		)

		switch bridgeConfig.Name {
		case "monopamp":
			monop := &monopampImpl{
				logger: logger,
			}
			if err := monop.setup(bridgeConfig); err != nil {
				logger.Fatal("error initializing module",
					zap.String("module_name", bridgeConfig.Name),
					zap.Error(err),
				)
			}
			toClose = append(toClose, monop)
			hub.AddBridge(monop.bridge, time.Second)
		case "hue":
			hue := &hueImpl{
				logger: logger,
			}
			if err := hue.setup(bridgeConfig, hub); err != nil {
				logger.Fatal("error initializing module",
					zap.String("module_name", bridgeConfig.Name),
					zap.Error(err),
				)
			}
			toClose = append(toClose, hue)
			go hue.Run() // the bridges are added via Run(), not here.
		case "proxy":
			proxy := &proxyImpl{
				logger: logger,
			}
			if err := proxy.setup(bridgeConfig, hub); err != nil {
				logger.Fatal("error initializing module",
					zap.String("module_name", bridgeConfig.Name),
					zap.Error(err),
				)
			}
			toClose = append(toClose, proxy)
			// the proxyImpl setup adds itself to the hub
		default:
			logger.Info("unsupported module config detected, ignoring",
				zap.String("module_name", bridgeConfig.Name),
			)
		}
	}

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

	api := domotics.NewAPI(logger, hub)

	grpcServer := grpc.NewServer()
	domotics.RegisterBridgeServiceServer(grpcServer, api)
	domotics.RegisterDeviceServiceServer(grpcServer, api)
	grpcServer.Serve(lis)

	for _, impl := range toClose {
		impl.Close()
	}
}
