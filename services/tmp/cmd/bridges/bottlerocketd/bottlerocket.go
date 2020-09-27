package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	br "github.com/rmrobinson/bottlerocket-go"
	"github.com/rmrobinson/nerves/services/domotics"
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
	baseX10Bridge = &domotics.Bridge{
		ModelId:          "CM17A",
		ModelName:        "Firecracker",
		ModelDescription: "Serial-X10 bridge",
		Manufacturer:     "x10.com",
		State: &domotics.BridgeState{
			IsPaired: true,
			Version: &domotics.BridgeState_Version{
				Api: "1.0.0",
				Sw:  "0.05b3",
			},
		},
	}
	baseX10Device = &domotics.Device{
		ModelId:          "1",
		ModelName:        "X10 Wall Unit",
		ModelDescription: "Plug-in X10 control unit",
		Manufacturer:     "x10.com",
	}
)

// Bottlerocket offers the standard bridge capabilities over the Bottlerocket X10 USB/serial interface.
type Bottlerocket struct {
	logger *zap.Logger

	br *br.Bottlerocket

	persister bridge.Persister
}

// NewBottlerocket takes a previously set up bottlerocket handle and exposes it as a bottlerocket bridge.
func NewBottlerocket(logger *zap.Logger, bridge *br.Bottlerocket, persister bridge.Persister) *Bottlerocket {
	return &Bottlerocket{
		logger:    logger,
		br:        bridge,
		persister: persister,
	}
}

// Setup seeds the persistent store with the correct data.
func (b *Bottlerocket) Setup(ctx context.Context) error {
	_, err := b.persister.Bridge(ctx)
	if err == bridge.ErrDatabaseNotSetup {
		return b.setup(ctx)
	}
	return nil
}

func (b *Bottlerocket) setup(ctx context.Context) error {
	// Populate the devices
	for _, houseID := range houses {
		for deviceID := 1; deviceID <= maxDeviceID; deviceID++ {
			d := &domotics.Device{
				// Id is populated by CreateDevice
				IsActive: false,
				Address:  fmt.Sprintf("%s%s%d", x10AddrPrefix, houseID, deviceID),
				Config: &domotics.DeviceConfig{
					Name:        "X10 device",
					Description: "Basic X10 device",
				},
				State: &domotics.DeviceState{
					Binary: &domotics.DeviceState_BinaryState{
						IsOn: false,
					},
				},
			}
			proto.Merge(d, baseX10Device)
			if err := b.persister.CreateDevice(ctx, d); err != nil {
				return err
			}
		}
	}

	return nil
}

// Bridge retrieves the state of the bridge from the persistent store.
func (b *Bottlerocket) Bridge(ctx context.Context) (*domotics.Bridge, error) {
	bridge, err := b.persister.Bridge(ctx)
	if err != nil {
		return nil, err
	}

	ret := &domotics.Bridge{
		Config: &domotics.BridgeConfig{
			Address: &domotics.Address{
				Usb: &domotics.Address_Usb{
					Path: b.br.Path(),
				},
			},
			Timezone: "UTC",
		},
	}
	proto.Merge(ret, baseX10Bridge)
	proto.Merge(ret, bridge)
	return ret, nil
}

// SetBridgeConfig saves the supplied config into the persistent store.
func (b *Bottlerocket) SetBridgeConfig(ctx context.Context, config *domotics.BridgeConfig) error {
	return b.persister.SetBridgeConfig(ctx, config)
}

// SetBridgeState saves the supplied state into the persistent store.
func (b *Bottlerocket) SetBridgeState(ctx context.Context, state *domotics.BridgeState) error {
	return b.persister.SetBridgeState(ctx, state)
}

// SearchForAvailableDevices is a noop as the devices are fixed and seeded at setup time.
func (b *Bottlerocket) SearchForAvailableDevices(context.Context) error {
	return nil
}

// AvailableDevices returns all X10 devices that have a valid address but is not yet in use.
func (b *Bottlerocket) AvailableDevices(ctx context.Context) ([]*domotics.Device, error) {
	return b.persister.AvailableDevices(ctx)
}

// Devices returns all X10 devices that are in use.
func (b *Bottlerocket) Devices(ctx context.Context) ([]*domotics.Device, error) {
	devices, err := b.persister.Devices(ctx)
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		proto.Merge(device, baseX10Device)
	}
	return devices, nil
}

// Device retrieves the specified device.
func (b *Bottlerocket) Device(ctx context.Context, id string) (*domotics.Device, error) {
	device, err := b.persister.Device(ctx, id)
	if err != nil {
		return nil, err
	}
	proto.Merge(device, baseX10Device)
	return device, nil
}

// SetDeviceConfig saves the specified config into the persistent store.
func (b *Bottlerocket) SetDeviceConfig(ctx context.Context, dev *domotics.Device, config *domotics.DeviceConfig) error {
	return b.persister.SetDeviceConfig(ctx, dev, config)
}

// SetDeviceState saves the supplied state into the persistent store as well as triggering the requested change
// to the supplied serial port.
func (b *Bottlerocket) SetDeviceState(ctx context.Context, dev *domotics.Device, state *domotics.DeviceState) error {
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

	return b.persister.SetDeviceState(ctx, dev, state)
}

// AddDevice takes an available X10 device and adds it to the set of in use devices.
func (b *Bottlerocket) AddDevice(ctx context.Context, id string) error {
	// Move the device from available to in use
	return b.persister.AddDevice(ctx, id)
}

// DeleteDevice takes an in-use X10 device and moves it to the set of available X10 devices.
func (b *Bottlerocket) DeleteDevice(ctx context.Context, id string) error {
	// Move the device from in use to available, and remove the saved values
	return b.persister.DeleteDevice(ctx, id)
}
