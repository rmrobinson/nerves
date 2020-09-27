package remote

import (
	"context"

	"github.com/rmrobinson/nerves/lib/stream"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
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
	// ErrInternal is returned if the requested method had an error
	ErrInternal = status.New(codes.Internal, "internal error")
)

// SyncBridge is a simplified interface for implementing the domotics Bridge gRPC contract.
// This allows for rapid implementation of bridge capabilities by simplistic, synchronous
// libraries which don't require advanced features.
type SyncBridge interface {
	SetDeviceState(context.Context, *bridge.UpdateDeviceStateRequest) error
}

// Service provides simplistic, synchronous bridges an easy way to integrate functionality
// without requiring a complete gRPC server implementation of the domotics Bridge contract.
// The bridge configured here is typically statically defined, with a pre-set number
// of devices. This service guards against mis-addressed devices, out-of-range options, etc.
// and only allows for basic state management.
// This interface is built under the assumption that this service will be the only thing
// allowing writes to the underlying bridge, providing guards for serialized calls. It caches
// device profiles and does not actually query the underlying system.
type Service struct {
	logger  *zap.Logger
	brInfo  *bridge.Bridge
	devices map[string]*bridge.Device
	br      SyncBridge

	updates *stream.Source
}

// NewService takes the supplied bridge and device profiles and takes on management of them.
// The supplied synchronous bridge interface will be used when the service detects an incoming
// write which requires a state change in the underlying device.
func NewService(logger *zap.Logger, brInfo *bridge.Bridge, devices map[string]*bridge.Device, br SyncBridge) *Service {
	return &Service{
		logger:  logger,
		brInfo:  brInfo,
		devices: devices,
		br:      br,

		updates: stream.NewSource(logger),
	}
}

// GetBridge retrieves the bridge info of this service.
func (s *Service) GetBridge(ctx context.Context, req *bridge.GetBridgeRequest) (*bridge.Bridge, error) {
	return s.brInfo, nil
}

// ListDevices retrieves all registered devices.
func (s *Service) ListDevices(ctx context.Context, req *bridge.ListDevicesRequest) (*bridge.ListDevicesResponse, error) {
	resp := &bridge.ListDevicesResponse{}

	for _, device := range s.devices {
		resp.Devices = append(resp.Devices, device)
	}

	return resp, nil
}

// GetDevice retrieves the specified device.
func (s *Service) GetDevice(ctx context.Context, req *bridge.GetDeviceRequest) (*bridge.Device, error) {
	if device, found := s.devices[req.Id]; found {
		return device, nil
	}

	return nil, ErrDeviceNotFound.Err()
}

// SetDeviceConfig exists to satisfy the domotics Bridge contract, but is not actually supported.
func (s *Service) SetDeviceConfig(ctx context.Context, req *bridge.UpdateDeviceConfigRequest) (*bridge.Device, error) {
	return nil, ErrNotImplemented.Err()
}

// SetDeviceState updates the specified device with the provided state.
func (s *Service) SetDeviceState(ctx context.Context, req *bridge.UpdateDeviceStateRequest) (*bridge.Device, error) {
	// TODO: guard against noop writes

	if _, found := s.devices[req.Id]; !found {
		return nil, ErrDeviceNotFound.Err()
	}

	err := s.br.SetDeviceState(ctx, req)
	if err != nil {
		s.logger.Info("error setting device state",
			zap.String("id", req.Id),
			zap.Error(err),
		)
		return nil, ErrInternal.Err()
	}

	s.devices[req.Id].State = req.State

	s.updates.SendMessage(&bridge.Update{
		Action: bridge.Update_ADDED,
		Update: &bridge.Update_DeviceUpdate{
			&bridge.DeviceUpdate{
				Device: s.devices[req.Id],
			},
		},
	})

	return s.devices[req.Id], nil
}

// StreamBridgeUpdates monitors changes for all changes which occur on the bridge.
// This will only pick up successful device writes.
func (s *Service) StreamBridgeUpdates(req *bridge.StreamBridgeUpdatesRequest, stream bridge.BridgeService_StreamBridgeUpdatesServer) error {
	peer, isOk := peer.FromContext(stream.Context())

	addr := "unknown"
	if isOk {
		addr = peer.Addr.String()
	}

	logger := s.logger.With(zap.String("peer_addr", addr))

	logger.Debug("bridge update stream initiated")

	sink := s.updates.NewSink()

	// Send all of the devices to start.
	for _, device := range s.devices {
		update := &bridge.Update{
			Action: bridge.Update_ADDED,
			Update: &bridge.Update_DeviceUpdate{
				&bridge.DeviceUpdate{
					Device: device,
				},
			},
		}

		logger.Debug("sending seed info",
			zap.String("device_info", update.String()),
		)

		if err := stream.Send(update); err != nil {
			logger.Error("unable to send update",
				zap.Error(err),
			)
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

		bridgeUpdate, ok := update.(*bridge.Update)

		if !ok {
			panic("update cast incorrect")
		}

		logger.Debug("sending update",
			zap.String("info", bridgeUpdate.String()),
		)

		if err := stream.Send(bridgeUpdate); err != nil {
			return err
		}
	}

	return nil
}
