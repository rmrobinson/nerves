package domotics

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var (
	// ErrDeviceNotFound is returned if the requested device does not exist.
	ErrDeviceNotFound = status.New(codes.NotFound, "device not found")
	// ErrNotImplemented is returned if the requested method is not yet implemented.
	ErrNotImplemented = status.New(codes.Unimplemented, "not implemented")
)

// API is a handle to the building implementation of the device and bridge gRPC server interfaces.
type API struct {
	logger *zap.Logger
	hub    *Hub
}

// NewAPI creates a new API backed by the supplied hub implementation.
func NewAPI(logger *zap.Logger, hub *Hub) *API {
	return &API{
		logger: logger,
		hub:    hub,
	}
}

// ListBridges retrieves the bridges configured on the hub.
func (a *API) ListBridges(ctx context.Context, req *ListBridgesRequest) (*ListBridgesResponse, error) {
	resp := &ListBridgesResponse{}
	for _, bridge := range a.hub.Bridges() {
		resp.Bridges = append(resp.Bridges, bridge)
	}

	return resp, nil
}

// SetBridgeConfig saves a new bridge configuration on the specified bridge.
func (a *API) SetBridgeConfig(ctx context.Context, req *SetBridgeConfigRequest) (*SetBridgeConfigResponse, error) {
	resp, err := a.hub.SetBridgeConfig(ctx, req.Id, req.Config)

	return &SetBridgeConfigResponse{
		Bridge: resp,
	}, err
}

// StreamBridgeUpdates monitors the hub and propagates any bridge change updates to registered listeners.
func (a *API) StreamBridgeUpdates(req *StreamBridgeUpdatesRequest, stream BridgeService_StreamBridgeUpdatesServer) error {
	peer, isOk := peer.FromContext(stream.Context())

	addr := "unknown"
	if isOk {
		addr = peer.Addr.String()
	}

	a.logger.Debug("watchBridges request",
		zap.String("peer_addr", addr),
	)

	sink := a.hub.bridgeUpdatesSource.NewSink()

	// Send all of the currently active bridges to start.
	for _, impl := range a.hub.bridges {
		update := &BridgeUpdate{
			Action: BridgeUpdate_ADDED,
			Bridge: impl.bridge,
		}

		a.logger.Debug("sending seed info",
			zap.String("peer_addr", addr),
			zap.String("bridge_info", update.String()),
		)

		if err := stream.Send(update); err != nil {
			return err
		}
	}
	// TODO: the above is subject to a race condition where the add is processed after we've added the watcher
	// but before we get the range of bridges so we duplicate data.
	// This shouldn't cause issues on the client (they should be tolerant to this) but let's fix this anyways.

	// Now we wait for updates
	for {
		update, ok := <-sink.Messages()
		if !ok {
			// Channel has been closed; so we'll close the connection as well
			return nil
		}

		bridgeUpdate, ok := update.(*BridgeUpdate)
		if !ok {
			panic("bridge update cast failed)")
		}

		a.logger.Debug("sending update",
			zap.String("peer_addr", addr),
			zap.String("bridge_info", update.String()),
		)
		if err := stream.Send(bridgeUpdate); err != nil {
			return err
		}
	}

	return nil
}

// ListDevices retrieves all registered devices.
func (a *API) ListDevices(ctx context.Context, req *ListDevicesRequest) (*ListDevicesResponse, error) {
	resp := &ListDevicesResponse{}

	var devices []*Device
	if len(req.BridgeId) > 0 {
		var err error
		devices, err = a.hub.DevicesOnBridge(req.BridgeId)
		if err != nil {
			return nil, err
		}
	} else {
		devices = a.hub.Devices()
	}

	for _, device := range devices {
		resp.Devices = append(resp.Devices, device)
	}

	return resp, nil
}

// ListAvailableDevices returns the list of devices available for use but haven't been added yet.
func (a *API) ListAvailableDevices(context.Context, *ListDevicesRequest) (*ListDevicesResponse, error) {
	return nil, ErrNotImplemented.Err()
}

// GetDevice retrieves the specified device.
func (a *API) GetDevice(ctx context.Context, req *GetDeviceRequest) (*GetDeviceResponse, error) {
	for _, device := range a.hub.Devices() {
		if device.Id == req.Id {
			return &GetDeviceResponse{
				Device: device,
			}, nil
		}
	}

	return nil, ErrDeviceNotFound.Err()
}

// SetDeviceConfig updates the specified device with the provided config.
func (a *API) SetDeviceConfig(ctx context.Context, req *SetDeviceConfigRequest) (*SetDeviceConfigResponse, error) {
	resp, err := a.hub.SetDeviceConfig(ctx, req.Id, req.Config, req.User)

	return &SetDeviceConfigResponse{
		Device: resp,
	}, err
}

// SetDeviceState updates the specified device with the provided state.
func (a *API) SetDeviceState(ctx context.Context, req *SetDeviceStateRequest) (*SetDeviceStateResponse, error) {
	resp, err := a.hub.SetDeviceState(ctx, req.Id, req.State, req.User)

	return &SetDeviceStateResponse{
		Device: resp,
	}, err
}

// StreamDeviceUpdates monitors changes for all devices tied to the hub.
func (a *API) StreamDeviceUpdates(req *StreamDeviceUpdatesRequest, stream DeviceService_StreamDeviceUpdatesServer) error {
	peer, isOk := peer.FromContext(stream.Context())

	addr := "unknown"
	if isOk {
		addr = peer.Addr.String()
	}

	a.logger.Debug("watchDevices request",
		zap.String("peer_addr", addr),
	)

	sink := a.hub.deviceUpdatesSource.NewSink()

	// Send all of the currently active bridges to start.
	for _, impl := range a.hub.Devices() {
		update := &DeviceUpdate{
			Action: DeviceUpdate_ADDED,
			Device: impl,
		}

		a.logger.Debug("sending seed info",
			zap.String("peer_addr", addr),
			zap.String("device_info", update.String()),
		)

		if err := stream.Send(update); err != nil {
			return err
		}
	}
	// TODO: the above is subject to a race condition where the add is processed after we've added the watcher
	// but before we get the range of devices so we duplicate data.
	// This shouldn't cause issues on the client (they should be tolerant to this) but let's fix this anyways.

	// Now we wait for updates
	for {
		update, ok := <-sink.Messages()
		if !ok {
			// Channel has been closed; so we'll close the connection as well
			return nil
		}

		deviceUpdate, ok := update.(*DeviceUpdate)

		if !ok {
			panic("device update cast incorrect")
		}

		a.logger.Debug("sending update",
			zap.String("peer_addr", addr),
			zap.String("device_info", update.String()),
		)

		if err := stream.Send(deviceUpdate); err != nil {
			return err
		}
	}

	return nil
}
