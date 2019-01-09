package mock

import (
	"context"
	"fmt"
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/rmrobinson/nerves/services/domotics"
)

var (
	basePersistentBridge = &domotics.Bridge{
		Type:             domotics.BridgeType_LOOPBACK,
		ModelId:          "PBTest1",
		ModelName:        "Test Persistent Bridge",
		ModelDescription: "Bridge for testing persistent operations",
		Manufacturer:     "Faltung Systems",
	}
	basePersistentDevice = &domotics.Device{
		ModelId:          "PBDevice1",
		ModelName:        "Test Persistent Device",
		ModelDescription: "Device for testing persistent operations",
		Manufacturer:     "Faltung Systems",
	}
)

// PersistentBridge is an implementation of the SyncBridge backed by a database to demonstrate how to achieve persistence when the underlying bridge does not support it.
type PersistentBridge struct {
	bridgeID  string
	persister domotics.BridgePersister
}

// NewPersistentBridge creates a new persistent bridge backed by the supplied persister.
func NewPersistentBridge(persister domotics.BridgePersister) *PersistentBridge {
	ret := &PersistentBridge{
		persister: persister,
	}
	return ret
}

func (b *PersistentBridge) setup() {
	bridgeConfig := &domotics.BridgeConfig{
		Name: "Test Persistent Bridge",
	}
	id, err := b.persister.CreateBridge(context.Background(), bridgeConfig)
	if err != nil {
		log.Printf("Error creating bridge: %s\n", err.Error())
		return
	}
	b.bridgeID = id

	for i := 0; i < 5; i++ {
		d := newDevice()
		d.d.IsActive = false
		d.d.Address = fmt.Sprintf("/test/%d", i)

		b.persister.CreateDevice(context.Background(), d.d)
	}
	for i := 5; i < 7; i++ {
		d := newDevice()
		d.d.IsActive = true
		d.d.Address = fmt.Sprintf("/test/%d", i)

		b.persister.CreateDevice(context.Background(), d.d)
	}

	log.Printf("Created bridge")
}

// Run creates the bridge and sets it up if not already configured. No random state changes are made.
func (b *PersistentBridge) Run() {
	_, err := b.persister.Bridge(context.Background())
	if err == domotics.ErrDatabaseNotSetup {
		b.setup()
	}
}

// Bridge returns the details of this bridge.
func (b *PersistentBridge) Bridge(ctx context.Context) (*domotics.Bridge, error) {
	bridge, err := b.persister.Bridge(ctx)
	if err != nil {
		return nil, err
	}
	proto.Merge(bridge, basePersistentBridge)
	return bridge, nil
}

// SetBridgeConfig allows the configuration of this bridge to be updated.
func (b *PersistentBridge) SetBridgeConfig(ctx context.Context, config *domotics.BridgeConfig) error {
	return b.persister.SetBridgeConfig(ctx, config)
}

// SetBridgeState allows the state of this bridge to be updated.
func (b *PersistentBridge) SetBridgeState(ctx context.Context, state *domotics.BridgeState) error {
	return b.persister.SetBridgeState(ctx, state)
}

// SearchForAvailableDevices begins the process of finding new devices that have been added to this bridge.
func (b *PersistentBridge) SearchForAvailableDevices(ctx context.Context) error {
	return b.persister.SearchForAvailableDevices(ctx)
}

// AvailableDevices returns the set of devices that this bridge knows about but hasn't configured yet.
func (b *PersistentBridge) AvailableDevices(ctx context.Context) ([]*domotics.Device, error) {
	devices, err := b.persister.AvailableDevices(ctx)
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		proto.Merge(device, basePersistentDevice)
	}
	return devices, nil
}

// Devices returns the list of configured devices on this bridge.
func (b *PersistentBridge) Devices(ctx context.Context) ([]*domotics.Device, error) {
	devices, err := b.persister.Devices(ctx)
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		proto.Merge(device, basePersistentDevice)
	}
	return devices, nil
}

// Device retrieves the requested device.
func (b *PersistentBridge) Device(ctx context.Context, id string) (*domotics.Device, error) {
	device, err := b.persister.Device(ctx, id)
	if err != nil {
		return nil, err
	}
	proto.Merge(device, basePersistentDevice)
	return device, nil
}

// SetDeviceConfig allows the configuration of a device to be updated.
func (b *PersistentBridge) SetDeviceConfig(ctx context.Context, dev *domotics.Device, config *domotics.DeviceConfig) error {
	return b.persister.SetDeviceConfig(ctx, dev, config)
}

// SetDeviceState allows the state of a device to be updated.
func (b *PersistentBridge) SetDeviceState(ctx context.Context, dev *domotics.Device, state *domotics.DeviceState) error {
	return b.persister.SetDeviceState(ctx, dev, state)
}

// AddDevice takes a device found in the set of AvailableDevices and configures it with the bridge for use.
func (b *PersistentBridge) AddDevice(ctx context.Context, id string) error {
	return b.persister.AddDevice(ctx, id)
}

// DeleteDevice requests that the specified device be removed from the bridge.
func (b *PersistentBridge) DeleteDevice(ctx context.Context, id string) error {
	return b.persister.DeleteDevice(ctx, id)
}
