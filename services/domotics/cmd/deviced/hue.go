package main

import (
	"context"
	"errors"
	"time"

	"github.com/rmrobinson/hue-go"
	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
)

var (
	// ErrUnableToSetupHue is returned if the hue locator cannot be started.
	ErrUnableToSetupHue = errors.New("unable to setup hue locator")
)

type hueImpl struct {
	logger *zap.Logger
	db     bridge.HuePersister

	hub  *domotics.Hub
	quit chan bool
}

func (b *hueImpl) setup(config *domotics.BridgeConfig, hub *domotics.Hub) error {
	if len(config.CachePath) < 1 {
		return ErrBridgeConfigInvalid
	}

	db := &bridge.HueDB{}
	err := db.Open(config.CachePath)
	if err != nil {
		b.logger.Warn("error initializing hue db",
			zap.Error(err),
		)
		return ErrUnableToSetupHue
	}
	b.db = db
	b.hub = hub
	b.quit = make(chan bool)

	return nil
}

// Close cleans up the Hue listener.
func (b *hueImpl) Close() error {
	if b.db != nil {
		b.db.Close()
	}

	b.quit <- true
	return nil
}

// Run starts listening for Hue bridge broadcasts.
func (b *hueImpl) Run() {
	bridges := make(chan hue.Bridge)

	locator := hue.NewLocator()
	go locator.Run(bridges)

	for {
		select {
		case br := <-bridges:
			b.logger.Info("hue bridge located",
				zap.String("bridge_id", br.ID()),
			)

			username, err := b.db.Profile(context.Background(), br.ID())
			if err != nil {
				b.logger.Warn("unable to get pairing id",
					zap.String("bridge_id", br.ID()),
					zap.Error(err),
				)
			} else {
				br.Username = username
			}

			b.hub.AddBridge(bridge.NewHueBridge(&br), time.Second)
		case <-b.quit:
			return
		}
	}
}
