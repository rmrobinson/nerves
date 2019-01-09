package mock

import (
	"context"
	"math/rand"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/rmrobinson/nerves/services/domotics"
)

// SyncBridge is a mock bridge that does not send notifications on state changes.
type SyncBridge struct {
	b *domotics.Bridge

	availDevices []*domotics.Device
	devices      map[string]*device
}

// NewSyncBridge creates a new instance of the SyncBridge.
func NewSyncBridge() *SyncBridge {
	ret := &SyncBridge{
		b: &domotics.Bridge{
			Id:               uuid.New().String(),
			Type:             domotics.BridgeType_LOOPBACK,
			Mode:             domotics.BridgeMode_ACTIVE,
			ModeReason:       "",
			ModelId:          "SBTest1",
			ModelName:        "Test Sync Bridge",
			ModelDescription: "Bridge for testing sync operations",
			Manufacturer:     "Faltung Systems",
			State: &domotics.BridgeState{
				IsPaired: true,
			},
		},
		devices: map[string]*device{},
	}

	count := rand.Intn(5)
	for i := 0; i < count; i++ {
		d := newDevice()
		ret.devices[d.d.Id] = d
	}

	return ret
}

// Run begins the process of updating the configured devices with fake state information.
func (sb *SyncBridge) Run() {
	t := time.NewTicker(time.Second * 6)
	for {
		select {
		case <-t.C:
			for id, d := range sb.devices {
				d.update()
				sb.devices[id] = d
			}
		}
	}
}

// Bridge returns the details of this bridge.
func (sb *SyncBridge) Bridge(ctx context.Context) (*domotics.Bridge, error) {
	return sb.b, nil
}

// SetBridgeConfig allows the configuration of this bridge to be updated.
func (sb *SyncBridge) SetBridgeConfig(ctx context.Context, config *domotics.BridgeConfig) error {
	sb.b.Config = config
	return nil
}

// SetBridgeState allows the state of this bridge to be updated.
func (sb *SyncBridge) SetBridgeState(ctx context.Context, state *domotics.BridgeState) error {
	return ErrReadOnly
}

// SearchForAvailableDevices begins the process of finding new devices that have been added to this bridge.
func (sb *SyncBridge) SearchForAvailableDevices(context.Context) error {
	if len(sb.availDevices) < 1 {
		count := rand.Intn(5)

		for i := 0; i < count; i++ {
			sb.availDevices = append(sb.availDevices, newDevice().d)
		}
	}

	return nil
}

// AvailableDevices returns the set of devices that this bridge knows about but hasn't configured yet.
func (sb *SyncBridge) AvailableDevices(context.Context) ([]*domotics.Device, error) {
	return sb.availDevices, nil
}

// Devices returns the list of configured devices on this bridge.
func (sb *SyncBridge) Devices(context.Context) ([]*domotics.Device, error) {
	var ret []*domotics.Device
	for _, d := range sb.devices {
		ret = append(ret, d.d)
	}
	return ret, nil
}

// Device retrieves the requested device.
func (sb *SyncBridge) Device(ctx context.Context, id string) (*domotics.Device, error) {
	if d, ok := sb.devices[id]; ok {
		return d.d, nil
	}
	return nil, ErrDeviceNotPresent
}

// SetDeviceConfig allows the configuration of a device to be updated.
func (sb *SyncBridge) SetDeviceConfig(ctx context.Context, dev *domotics.Device, config *domotics.DeviceConfig) error {
	var d *device
	var ok bool
	if d, ok = sb.devices[dev.Id]; !ok {
		return ErrDeviceNotPresent
	}

	d.d.Config = proto.Clone(config).(*domotics.DeviceConfig)
	return nil
}

// SetDeviceState allows the state of a device to be updated.
func (sb *SyncBridge) SetDeviceState(ctx context.Context, dev *domotics.Device, state *domotics.DeviceState) error {
	var d *device
	var ok bool
	if d, ok = sb.devices[dev.Id]; !ok {
		return ErrDeviceNotPresent
	}

	d.d.State = proto.Clone(state).(*domotics.DeviceState)
	return nil

}

// AddDevice takes a device found in the set of AvailableDevices and configures it with the bridge for use.
func (sb *SyncBridge) AddDevice(ctx context.Context, id string) error {
	var d *domotics.Device
	found := false
	for idx, availDevice := range sb.availDevices {
		if availDevice.Id == id {
			d = availDevice
			sb.availDevices = append(sb.availDevices[:idx], sb.availDevices[idx+1:]...)
			found = true
			break
		}
	}

	if !found {
		return ErrDeviceNotPresent
	}

	sb.devices[d.Id] = &device{
		d: d,
	}
	return nil
}

// DeleteDevice requests that the specified device be removed from the bridge.
func (sb *SyncBridge) DeleteDevice(ctx context.Context, id string) error {
	if _, ok := sb.devices[id]; !ok {
		return ErrDeviceNotPresent
	}
	delete(sb.devices, id)
	return nil
}
