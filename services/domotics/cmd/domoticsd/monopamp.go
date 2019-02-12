package main

import (
	"context"
	"os"

	mpa "github.com/rmrobinson/monoprice-amp-go"
	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/tarm/serial"
	"go.uber.org/zap"
)

const (
	monopricePortBaudRate = 9600
)

type monopampImpl struct {
	logger *zap.Logger

	bridge domotics.SyncBridge
	db     *bridge.DB

	port *serial.Port
}

func (b *monopampImpl) setup(config *domotics.BridgeConfig) error {
	if config.Address.Usb == nil {
		return ErrBridgeConfigInvalid
	}

	setupNeeded := false
	if _, err := os.Stat(config.CachePath); os.IsNotExist(err) {
		setupNeeded = true
	}

	b.db = &bridge.DB{}
	err := b.db.Open(config.CachePath)
	if err != nil {
		return err
	}

	c := &serial.Config{
		Name: config.Address.Usb.Path,
		Baud: monopricePortBaudRate,
	}
	b.port, err = serial.OpenPort(c)
	if err != nil {
		b.logger.Warn("error initializing serial port",
			zap.String("port_path", config.Address.Usb.Path),
			zap.Error(err),
		)

		b.db.Close()
		return ErrUnableToSetupMonopAmp
	}

	amp, err := mpa.NewSerialAmplifier(b.port)
	if err != nil {
		b.logger.Warn("error initializing monoprice amp library",
			zap.String("port_path", config.Address.Usb.Path),
			zap.Error(err),
		)
		b.db.Close()
		b.port.Close()
		return err
	}

	monopBridge := bridge.NewMonopAmpBridge(amp, b.db)
	b.bridge = monopBridge

	if setupNeeded {
		return monopBridge.Setup(context.Background())
	}
	return nil
}

// Close cleans up any open resources
func (b *monopampImpl) Close() error {
	var portErr error
	if b.port != nil {
		portErr = b.port.Close()
	}
	if b.db != nil {
		b.db.Close()
	}
	return portErr
}
