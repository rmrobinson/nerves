package bridge

import (
	"context"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/rmrobinson/nerves/lib/stream"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var (
	// ErrMissingParam is returned if a request is missing a required field
	ErrMissingParam = status.New(codes.InvalidArgument, "required param missing")
	// ErrNotSupported is returned if the requested method is not supported.
	ErrNotSupported = status.New(codes.InvalidArgument, "operation not supported")
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
	SetDeviceState(context.Context, *Device, *DeviceState) error
}

// SyncBridgeService provides simplistic, synchronous bridges an easy way to integrate functionality
// without requiring a complete gRPC server implementation of the domotics Bridge contract.
// The bridge configured here is typically statically defined, with a pre-set number
// of devices. This service guards against mis-addressed devices, out-of-range options, etc.
// and only allows for basic state management.
// This interface is built under the assumption that this service will be the only thing
// allowing writes to the underlying bridge, providing guards for serialized calls. It caches
// device profiles and does not actually query the underlying system.
type SyncBridgeService struct {
	logger  *zap.Logger
	brInfo  *Bridge
	devices map[string]*Device
	br      SyncBridge
	brLock  sync.Mutex

	updates *stream.Source
}

// NewSyncBridgeService takes the supplied bridge and device profiles and takes on management of them.
// The supplied synchronous bridge interface will be used when the service detects an incoming
// write which requires a state change in the underlying device.
func NewSyncBridgeService(logger *zap.Logger, brInfo *Bridge, devices map[string]*Device, br SyncBridge) *SyncBridgeService {
	// Ensure we mark all these devices as reachable
	for id := range devices {
		devices[id].State.IsReachable = true
	}
	return &SyncBridgeService{
		logger:  logger,
		brInfo:  brInfo,
		devices: devices,
		br:      br,

		updates: stream.NewSource(logger),
	}
}

// GetBridge retrieves the bridge info of this service.
func (s *SyncBridgeService) GetBridge(ctx context.Context, req *GetBridgeRequest) (*Bridge, error) {
	ret := proto.Clone(s.brInfo).(*Bridge)

	s.brLock.Lock()
	defer s.brLock.Unlock()

	for _, device := range s.devices {
		ret.Devices = append(ret.Devices, device)
	}

	return ret, nil
}

// ListDevices retrieves all registered devices.
func (s *SyncBridgeService) ListDevices(ctx context.Context, req *ListDevicesRequest) (*ListDevicesResponse, error) {
	resp := &ListDevicesResponse{}

	for _, device := range s.devices {
		resp.Devices = append(resp.Devices, device)
	}

	return resp, nil
}

// GetDevice retrieves the specified device.
func (s *SyncBridgeService) GetDevice(ctx context.Context, req *GetDeviceRequest) (*Device, error) {
	if device, found := s.devices[req.Id]; found {
		return device, nil
	}

	return nil, ErrDeviceNotFound.Err()
}

// UpdateDeviceConfig exists to satisfy the domotics Bridge contract, but is not actually supported.
func (s *SyncBridgeService) UpdateDeviceConfig(ctx context.Context, req *UpdateDeviceConfigRequest) (*Device, error) {
	return nil, ErrNotSupported.Err()
}

// UpdateDeviceState updates the specified device with the provided state.
func (s *SyncBridgeService) UpdateDeviceState(ctx context.Context, req *UpdateDeviceStateRequest) (*Device, error) {
	var device *Device
	var found bool

	if len(req.Id) < 1 || req.State == nil {
		return nil, ErrMissingParam.Err()
	} else if req.State.IsReachable == false {
		return nil, ErrNotSupported.Err()
	} else if device, found = s.devices[req.Id]; !found {
		return nil, ErrDeviceNotFound.Err()
	}

	if req.State.String() == device.State.String() {
		s.logger.Debug("noop write, ignoring",
			zap.String("device_id", req.Id),
		)
		return device, nil
	}

	// TODO: check against request version field

	s.brLock.Lock()
	err := s.br.SetDeviceState(ctx, device, req.State)
	s.brLock.Unlock()

	if err != nil {
		s.logger.Info("error setting device state",
			zap.String("id", req.Id),
			zap.Error(err),
		)
		return nil, ErrInternal.Err()
	}

	s.devices[req.Id].State = req.State

	s.updates.SendMessage(&Update{
		Action: Update_CHANGED,
		Update: &Update_DeviceUpdate{
			&DeviceUpdate{
				Device:   s.devices[req.Id],
				BridgeId: s.brInfo.Id,
			},
		},
	})

	return s.devices[req.Id], nil
}

// StreamBridgeUpdates monitors changes for all changes which occur on the
// This will only pick up successful device writes.
func (s *SyncBridgeService) StreamBridgeUpdates(req *StreamBridgeUpdatesRequest, stream BridgeService_StreamBridgeUpdatesServer) error {
	peer, isOk := peer.FromContext(stream.Context())

	addr := "unknown"
	if isOk {
		addr = peer.Addr.String()
	}

	logger := s.logger.With(zap.String("peer_addr", addr))

	logger.Debug("bridge update stream initiated")

	sink := s.updates.NewSink()
	defer sink.Close()

	// Send all of the devices to start.
	for _, device := range s.devices {
		update := &Update{
			Action: Update_ADDED,
			Update: &Update_DeviceUpdate{
				&DeviceUpdate{
					Device:   device,
					BridgeId: s.brInfo.Id,
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
			logger.Debug("stream closed")
			// Channel has been closed; so we'll close the connection as well
			return nil
		}

		bridgeUpdate, ok := update.(*Update)

		if !ok {
			panic("update cast incorrect")
		}

		logger.Debug("sending update")

		if err := stream.Send(bridgeUpdate); err != nil {
			return err
		}
	}
}
