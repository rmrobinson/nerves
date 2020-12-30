package main

import (
	"context"
	"fmt"

	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
)

// Console offers the standard bridge capabilities echoing to the console.
type Console struct {
	logger *zap.Logger

	id string
}

// NewConsole creates a console-based bridge for echoing purposes.
func NewConsole(logger *zap.Logger, id string) *Console {
	return &Console{
		logger: logger,
		id:     id,
	}
}

func (c *Console) getDevices(ctx context.Context) (map[string]*bridge.Device, error) {
	devices := map[string]*bridge.Device{}
	// Populate the devices

	d := &bridge.Device{
		Id:       fmt.Sprintf("%s-1", c.id),
		IsActive: true,
		Type:     bridge.DeviceType_SWITCH,
		Address:  "/console/stdout",
		Config: &bridge.DeviceConfig{
			Name:        "Console device",
			Description: "Basic echo device",
		},
		State: &bridge.DeviceState{
			Binary: &bridge.DeviceState_Binary{
				IsOn: false,
			},
			Version: &bridge.Version{
				Sw: "0.1",
				Hw: "0.1",
			},
		},
		ModelId:          "N/A",
		ModelName:        "Console echoer",
		ModelDescription: "Simply echos the requested fields to the console",
		Manufacturer:     "faltung.ca",
	}

	devices[d.Id] = d

	return devices, nil
}

func (c *Console) getBridge(ctx context.Context) (*bridge.Bridge, error) {
	ret := &bridge.Bridge{
		Id:               c.id,
		ModelId:          "C1",
		ModelName:        "Console",
		ModelDescription: "Console echo bridge",
		Manufacturer:     "faltung.ca",
		State: &bridge.BridgeState{
			IsPaired: true,
			Version: &bridge.Version{
				Api: "1.0.0",
				Sw:  "1.0.0",
			},
		},
		Config: &bridge.BridgeConfig{
			Timezone: "UTC",
		},
	}

	return ret, nil
}

// SetDeviceState echos the command to the console.
func (c *Console) SetDeviceState(ctx context.Context, dev *bridge.Device, state *bridge.DeviceState) error {
	fmt.Printf("Setting %s to %t\n", dev.Address, state.Binary.IsOn)
	return nil
}
