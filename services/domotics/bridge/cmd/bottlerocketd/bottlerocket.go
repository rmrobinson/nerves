package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	br "github.com/rmrobinson/bottlerocket-go"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
)

var (
	// ErrUnableToSetupBottlerocket is returned if the supplied bridge configuration fails to properly initialize br.
	ErrUnableToSetupBottlerocket = errors.New("unable to set up bottlerocket")
)

var (
	houses        = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P"}
	maxDeviceID   = 16
	x10AddrPrefix = "/x10/"
	baseX10Bridge = &bridge.Bridge{
		ModelId:          "CM17A",
		ModelName:        "Firecracker",
		ModelDescription: "Serial-X10 bridge",
		Manufacturer:     "x10.com",
		State: &bridge.BridgeState{
			IsPaired: true,
			Version: &bridge.BridgeState_Version{
				Api: "1.0.0",
				Sw:  "0.05b3",
			},
		},
	}
	baseX10Device = &bridge.Device{
		ModelId:          "1",
		ModelName:        "X10 Wall Unit",
		ModelDescription: "Plug-in X10 control unit",
		Manufacturer:     "x10.com",
	}
)

// Bottlerocket offers the standard bridge capabilities over the Bottlerocket X10 USB/serial interface.
type Bottlerocket struct {
	logger *zap.Logger

	id string
	br *br.Bottlerocket
}

// NewBottlerocket takes a previously set up bottlerocket handle and exposes it as a bottlerocket bridge.
func NewBottlerocket(logger *zap.Logger, id string, bottlerocketImpl *br.Bottlerocket) *Bottlerocket {
	return &Bottlerocket{
		logger: logger,
		id:     id,
		br:     bottlerocketImpl,
	}
}

func (b *Bottlerocket) getDevices(ctx context.Context) (map[string]*bridge.Device, error) {
	devices := map[string]*bridge.Device{}
	// Populate the devices
	for _, houseID := range houses {
		for deviceID := 1; deviceID <= maxDeviceID; deviceID++ {
			d := &bridge.Device{
				Id:       fmt.Sprintf("%s-%s%d", b.id, houseID, deviceID),
				IsActive: false,
				Address:  fmt.Sprintf("%s%s%d", x10AddrPrefix, houseID, deviceID),
				Config: &bridge.DeviceConfig{
					Name:        "X10 device",
					Description: "Basic X10 device",
				},
				State: &bridge.DeviceState{
					Binary: &bridge.DeviceState_Binary{
						IsOn: false,
					},
				},
			}
			proto.Merge(d, baseX10Device)
			devices[d.Id] = d
		}
	}

	// TODO: probably load a profile file here to properly populate this.

	return devices, nil
}

func (b *Bottlerocket) getBridge(ctx context.Context) (*bridge.Bridge, error) {
	ret := &bridge.Bridge{
		Config: &bridge.BridgeConfig{
			Address: &bridge.Address{
				Usb: &bridge.Address_Usb{
					Path: b.br.Path(),
				},
			},
			Timezone: "UTC",
		},
	}
	proto.Merge(ret, baseX10Bridge)

	return ret, nil
}

// SetDeviceState triggers the requested change to the supplied serial port.
func (b *Bottlerocket) SetDeviceState(ctx context.Context, dev *bridge.Device, state *bridge.DeviceState) error {
	var err error

	addr := strings.TrimPrefix(dev.Address, x10AddrPrefix)
	if state.Binary.IsOn {
		err = b.br.SendCommand(addr, "ON")
	} else {
		err = b.br.SendCommand(addr, "OFF")
	}

	if err != nil {
		return err
	}

	return nil
}
