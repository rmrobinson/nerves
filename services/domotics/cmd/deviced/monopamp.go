package main

import (
	"context"
	"log"

	mpa "github.com/rmrobinson/monoprice-amp-go"
	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/tarm/serial"
)

const (
	monopricePortBaudRate = 9600
)

type monopampImpl struct {
	port *serial.Port
	db *domotics.BridgeDB

	bridge domotics.SyncBridge
}

func (b *monopampImpl) setup(config *domotics.BridgeConfig) error {
	if config.Address.Usb == nil {
		return ErrBridgeConfigInvalid
	}

	b.db = &domotics.BridgeDB{}
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
		log.Printf("Error initializing serial port: %s\n", err.Error())
		b.db.Close()
		return ErrUnableToSetupMonopAmp
	}

	amp, err := mpa.NewSerialAmplifier(b.port)
	if err != nil {
		log.Printf("Error initializing monoprice amp: %s\n", err.Error())
		b.db.Close()
		b.port.Close()
		return err
	}

	monopBridge := bridge.NewMonopAmpBridge(amp, b.db)
	b.bridge = monopBridge
	return monopBridge.Setup(context.Background())
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
