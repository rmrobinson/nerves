package main

import (
	"context"
	"fmt"
	"net"

	mpa "github.com/rmrobinson/monoprice-amp-go"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/spf13/viper"
	"github.com/tarm/serial"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	monopAmpUSBPathEnvVar = "MONOPAMP_USB_PATH"
	monopAmpPortBaudRate  = 9600
	idEnvVar              = "ID"
	portEnvVar            = "PORT"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(idEnvVar)
	viper.BindEnv(portEnvVar)
	viper.BindEnv(monopAmpUSBPathEnvVar)

	monopAmpUSBPath := viper.GetString(monopAmpUSBPathEnvVar)
	if len(monopAmpUSBPath) < 1 {
		logger.Fatal("usb path missing")
	}

	c := &serial.Config{
		Name: monopAmpUSBPath,
		Baud: monopAmpPortBaudRate,
	}
	port, err := serial.OpenPort(c)
	if err != nil {
		logger.Fatal("error initializing serial port",
			zap.String("port_path", monopAmpUSBPath),
			zap.Error(err),
		)
	}
	defer port.Close()

	amp, err := mpa.NewSerialAmplifier(port)
	if err != nil {
		logger.Fatal("error initializing monoprice amp library",
			zap.String("port_path", monopAmpUSBPath),
			zap.Error(err),
		)
	}

	br := NewMonopAmp(amp, viper.GetString(idEnvVar), monopAmpUSBPath)

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

	ad := bridge.NewAdvertiser(logger, viper.GetString(idEnvVar), connStr)
	go ad.Run()
	defer ad.Shutdown()

	grpcServer := grpc.NewServer()
	bridge.RegisterBridgeServiceServer(grpcServer, sbs)
	grpcServer.Serve(lis)
}
