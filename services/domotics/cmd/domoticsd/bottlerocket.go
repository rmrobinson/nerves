package main

import (
	"context"
	"os"

	br "github.com/rmrobinson/bottlerocket-go"
	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
)

type bottlerocketImpl struct {
	logger *zap.Logger

	bridge domotics.SyncBridge
	db     *bridge.DB

	br *br.Bottlerocket
}

func (b *bottlerocketImpl) setup(config *domotics.BridgeConfig) error {
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

	b.br = &br.Bottlerocket{}
	err = b.br.Open(config.Address.Usb.Path)
	if err != nil {
		b.logger.Warn("error initializing bottlerocket port",
			zap.String("port_path", config.Address.Usb.Path),
			zap.Error(err),
		)

		b.db.Close()
		return ErrUnableToSetupBottlerocket
	}

	brBridge := bridge.NewBottlerocket(b.br, b.db)
	b.bridge = brBridge

	if setupNeeded {
		return brBridge.Setup(context.Background())
	}
	return nil
}

// Close cleans up any open resources
func (b *bottlerocketImpl) Close() error {
	if b.br != nil {
		b.br.Close()
	}
	if b.db != nil {
		b.db.Close()
	}
	return nil
}
