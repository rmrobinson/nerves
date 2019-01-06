package mock

import (
	"context"
	"math/rand"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/rmrobinson/nerves/services/domotics"
)

// AsyncBridge is a mock bridge that supports notifications on state changes.
type AsyncBridge struct {
	b *domotics.Bridge

	availDevices []*domotics.Device
	devices      map[string]*device

	notifier domotics.Notifier
}

// NewAsyncBridge creates a new instance of the AsyncBridge.
func NewAsyncBridge() *AsyncBridge {
	ret := &AsyncBridge{
		b: &domotics.Bridge{
			Id:               uuid.New().String(),
			Type:             domotics.BridgeType_LOOPBACK,
			Mode:             domotics.BridgeMode_ACTIVE,
			ModeReason:       "",
			ModelId:          "ABTest1",
			ModelName:        "Test Async Bridge",
			ModelDescription: "Bridge for testing async operations",
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

// Run begins the process of sending fake device updates to the registered Hub.
func (ab *AsyncBridge) Run() {
	t := time.NewTicker(7 * time.Second)
	for {
		select {
		case <-t.C:
			for id, d := range ab.devices {
				d.update()
				ab.devices[id] = d

				if ab.notifier != nil {
					ab.notifier.DeviceUpdated(ab.b.Id, d.d)
				}
			}
		}
	}
}

// SetNotifier takes the supplied notifier and registers it.
func (ab *AsyncBridge) SetNotifier(n domotics.Notifier) {
	ab.notifier = n
}

// Bridge returns the details of this bridge.
func (ab *AsyncBridge) Bridge(context.Context) (*domotics.Bridge, error) {
	return ab.b, nil
}
// SetBridgeConfig allows the configuration of this bridge to be updated.
func (ab *AsyncBridge) SetBridgeConfig(ctx context.Context, config *domotics.BridgeConfig) error {
	ab.b.Config = config
	return nil
}
// SetBridgeState allows the state of this bridge to be updated.
func (ab *AsyncBridge) SetBridgeState(ctx context.Context, state *domotics.BridgeState) error {
	return ErrReadOnly
}

// SearchForAvailableDevices begins the process of finding new devices that have been added to this bridge.
func (ab *AsyncBridge) SearchForAvailableDevices(context.Context) error {
	if len(ab.availDevices) < 1 {
		count := rand.Intn(5)

		for i := 0; i < count; i++ {
			ab.availDevices = append(ab.availDevices, newDevice().d)
		}
	}

	return nil
}
// AvailableDevices returns the set of devices that this bridge knows about but hasn't configured yet.
func (ab *AsyncBridge) AvailableDevices(context.Context) ([]*domotics.Device, error) {
	return ab.availDevices, nil
}

// Devices returns the list of configured devices on this bridge.
func (ab *AsyncBridge) Devices(context.Context) ([]*domotics.Device, error) {
	var ret []*domotics.Device
	for _, d := range ab.devices {
		ret = append(ret, d.d)
	}
	return ret, nil
}
// Device retrieves the requested device.
func (ab *AsyncBridge) Device(ctx context.Context, id string) (*domotics.Device, error) {
	if d, ok := ab.devices[id]; ok {
		return d.d, nil
	}
	return nil, ErrDeviceNotPresent
}

// SetDeviceConfig allows the configuration of a device to be updated.
func (ab *AsyncBridge) SetDeviceConfig(ctx context.Context, dev *domotics.Device, config *domotics.DeviceConfig) error {
	var d *device
	var ok bool
	if d, ok = ab.devices[dev.Id]; !ok {
		return ErrDeviceNotPresent
	}

	d.d.Config = proto.Clone(config).(*domotics.DeviceConfig)
	return nil
}
// SetDeviceState allows the state of a device to be updated.
func (ab *AsyncBridge) SetDeviceState(ctx context.Context, dev *domotics.Device, state *domotics.DeviceState) error {
	var d *device
	var ok bool
	if d, ok = ab.devices[dev.Id]; !ok {
		return ErrDeviceNotPresent
	}

	d.d.State = proto.Clone(state).(*domotics.DeviceState)
	return nil

}
// AddDevice takes a device found in the set of AvailableDevices and configures it with the bridge for use.
func (ab *AsyncBridge) AddDevice(ctx context.Context, id string) error {
	var d *domotics.Device
	found := false
	for idx, availDevice := range ab.availDevices {
		if availDevice.Id == id {
			d = availDevice
			ab.availDevices = append(ab.availDevices[:idx], ab.availDevices[idx+1:]...)
			found = true
			break
		}
	}

	if !found {
		return ErrDeviceNotPresent
	}

	ab.devices[d.Id] = &device{
		d: d,
	}
	return nil
}
// DeleteDevice requests that the specified device be removed from the bridge.
func (ab *AsyncBridge) DeleteDevice(ctx context.Context, id string) error {
	if _, ok := ab.devices[id]; !ok {
		return ErrDeviceNotPresent
	}
	delete(ab.devices, id)
	return nil
}
