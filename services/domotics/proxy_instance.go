package domotics

import (
	"context"

	"google.golang.org/grpc"
)

type proxyInstance struct {
	bridgeID string

	remoteBridge BridgeServiceClient
	remoteDevice DeviceServiceClient
	notifier     Notifier
}

func newProxyInstance(conn *grpc.ClientConn, id string) *proxyInstance {
	return &proxyInstance{
		bridgeID:     id,
		remoteBridge: NewBridgeServiceClient(conn),
		remoteDevice: NewDeviceServiceClient(conn),
	}
}

// SetNotifier saves the notifier to use for the proxy instance.
func (pi *proxyInstance) SetNotifier(notifier Notifier) {
	pi.notifier = notifier
}

// Bridge queries the linked peer to retrieve information on the requested bridge.
func (pi *proxyInstance) Bridge(ctx context.Context) (*Bridge, error) {
	resp, err := pi.remoteBridge.ListBridges(ctx, &ListBridgesRequest{})
	if err != nil {
		return nil, err
	}

	for _, bridge := range resp.Bridges {
		if bridge.Id == pi.bridgeID {
			return bridge, nil
		}
	}

	return nil, ErrBridgeNotRegistered
}

// SetBridgeConfig updates the linked peer with the config of the specified bridge.
func (pi *proxyInstance) SetBridgeConfig(ctx context.Context, config *BridgeConfig) error {
	req := &SetBridgeConfigRequest{
		Id:     pi.bridgeID,
		Config: config,
	}

	_, err := pi.remoteBridge.SetBridgeConfig(ctx, req)
	return err
}

// SetBridgeState updates the linked peer with the state of the specified bridge.
func (pi *proxyInstance) SetBridgeState(context.Context, *BridgeState) error {
	return ErrNotImplemented.Err()
}

// SearchForAvailableDevices requests that the linked peer begin searching for new devices.
func (pi *proxyInstance) SearchForAvailableDevices(context.Context) error {
	return ErrNotImplemented.Err()
}

// AvailableDevices queries the linked peer for the list of available but not yet added devices.
func (pi *proxyInstance) AvailableDevices(ctx context.Context) ([]*Device, error) {
	resp, err := pi.remoteDevice.ListAvailableDevices(ctx, &ListDevicesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Devices, nil
}

// Devices queries the linked peer for its collection of devices.
func (pi *proxyInstance) Devices(ctx context.Context) ([]*Device, error) {
	resp, err := pi.remoteDevice.ListDevices(ctx, &ListDevicesRequest{BridgeId: pi.bridgeID})
	if err != nil {
		return nil, err
	}

	return resp.Devices, nil
}

// Device queries the linked peer for the specified device.
func (pi *proxyInstance) Device(ctx context.Context, id string) (*Device, error) {
	req := &GetDeviceRequest{
		Id: id,
	}
	resp, err := pi.remoteDevice.GetDevice(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Device, nil
}

// SetDeviceConfig updates the linked peer with the config for the specified device.
func (pi *proxyInstance) SetDeviceConfig(ctx context.Context, device *Device, config *DeviceConfig) error {
	req := &SetDeviceConfigRequest{
		Id:     device.Id,
		Config: config,
	}
	_, err := pi.remoteDevice.SetDeviceConfig(ctx, req)
	return err
}

// SetDeviceState updates the linked peer with the state for the specified device.
func (pi *proxyInstance) SetDeviceState(ctx context.Context, device *Device, state *DeviceState) error {
	req := &SetDeviceStateRequest{
		Id:    device.Id,
		State: state,
	}
	_, err := pi.remoteDevice.SetDeviceState(ctx, req)
	return err
}

// AddDevice requests that the linked peer add the specified device.
func (pi *proxyInstance) AddDevice(context.Context, string) error {
	return ErrNotImplemented.Err()
}

// DeleteDevice requests that the linked peer remove the specified device.
func (pi *proxyInstance) DeleteDevice(context.Context, string) error {
	return ErrNotImplemented.Err()
}
