package building

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
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

// API is a handle to the building implementation of the device and bridge gRPC server interfaces.
type API struct {
	logger *zap.Logger
	svc    *Service
}

// NewAPI creates a new API backed by the supplied service implementation.
func NewAPI(logger *zap.Logger, svc *Service) *API {
	return &API{
		logger: logger,
		svc:    svc,
	}
}

// GetBridge retrieves the bridge profile of this domotics service.
func (a *API) GetBridge(ctx context.Context, req *bridge.GetBridgeRequest) (*bridge.Bridge, error) {
	return a.svc.bridge, nil
}

// StreamBridgeUpdates monitors the hub and propagates any bridge change updates to registered listeners.
func (a *API) StreamBridgeUpdates(req *bridge.StreamBridgeUpdatesRequest, stream bridge.BridgeService_StreamBridgeUpdatesServer) error {
	peer, isOk := peer.FromContext(stream.Context())

	addr := "unknown"
	if isOk {
		addr = peer.Addr.String()
	}

	logger := a.logger.With(zap.String("perr_addr", addr))

	logger.Debug("watchBridges request")

	sink := a.svc.bridgeUpdatesSource.NewSink()

	// Send all of the currently active devices to start.
	devices, err := a.svc.GetDevices(context.Background())
	if err != nil {
		a.logger.Error("unable to retrieve devices",
			zap.Error(err),
		)
		return err
	}

	for _, device := range devices {
		update := &bridge.Update{
			Action: bridge.Update_ADDED,
			Update: &bridge.Update_DeviceUpdate{
				DeviceUpdate: &bridge.DeviceUpdate{
					Device:   device,
					BridgeId: "todo-building-bridge-id",
				},
			},
		}

		logger.Debug("sending seed info",
			zap.String("update", update.String()),
		)

		if err := stream.Send(update); err != nil {
			logger.Error("error sending update",
				zap.Error(err),
			)
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

		bridgeUpdate, ok := update.(*bridge.Update)
		if !ok {
			panic("bridge update cast failed)")
		}

		logger.Debug("sending update",
			zap.String("bridge_info", update.String()),
		)
		if err := stream.Send(bridgeUpdate); err != nil {
			logger.Error("err sending update",
				zap.Error(err),
			)
			return err
		}
	}

	return nil
}

// ListDevices retrieves all registered devices.
func (a *API) ListDevices(ctx context.Context, req *bridge.ListDevicesRequest) (*bridge.ListDevicesResponse, error) {
	resp := &bridge.ListDevicesResponse{}

	devices, err := a.svc.GetDevices(ctx)
	if err != nil {
		a.logger.Error("error getting devices",
			zap.Error(err),
		)
		return nil, ErrInternal.Err()
	}

	for _, device := range devices {
		resp.Devices = append(resp.Devices, device)
	}

	return resp, nil
}

// GetDevice retrieves the specified device.
func (a *API) GetDevice(ctx context.Context, req *bridge.GetDeviceRequest) (*bridge.Device, error) {
	devices, err := a.svc.GetDevices(ctx)
	if err != nil {
		a.logger.Error("error getting devices",
			zap.Error(err),
		)
		return nil, ErrInternal.Err()
	}

	for _, device := range devices {
		if device.Id == req.Id {
			return device, nil
		}
	}

	return nil, ErrDeviceNotFound.Err()
}

// UpdateDeviceConfig updates the specified device with the provided config.
func (a *API) UpdateDeviceConfig(ctx context.Context, req *bridge.UpdateDeviceConfigRequest) (*bridge.Device, error) {
	return nil, ErrNotImplemented.Err()
}

// UpdateDeviceState updates the specified device with the provided state.
func (a *API) UpdateDeviceState(ctx context.Context, req *bridge.UpdateDeviceStateRequest) (*bridge.Device, error) {
	return nil, ErrNotImplemented.Err()
}

var (
	// ErrBuildingCreateFailed is returned when creating the building failed
	ErrBuildingCreateFailed = status.New(codes.Internal, "unable to create bridge")
	// ErrFloorCreateFailed is returned when creating the floor failed
	ErrFloorCreateFailed = status.New(codes.Internal, "unable to create floor")
	// ErrRoomCreateFailed is returned when creating the room failed
	ErrRoomCreateFailed = status.New(codes.Internal, "unable to create room")
)

// CreateBuilding satisfies the BuildingAdminService gRPC server API.
func (a *API) CreateBuilding(ctx context.Context, req *CreateBuildingRequest) (*Building, error) {
	err := a.svc.AddBuilding(ctx, req.Building)
	if err != nil {
		a.logger.Info("error creating building",
			zap.Error(err),
		)
		return nil, ErrBuildingCreateFailed.Err()
	}

	return req.Building, nil
}

// UpdateBuilding satisfies the BuildingAdminService gRPC server API.
func (a *API) UpdateBuilding(ctx context.Context, req *UpdateBuildingRequest) (*Building, error) {
	return nil, ErrNotImplemented.Err()
}

// DeleteBuilding satisfies the BuildingAdminService gRPC server API.
func (a *API) DeleteBuilding(ctx context.Context, req *DeleteBuildingRequest) (*empty.Empty, error) {
	return &empty.Empty{}, ErrNotImplemented.Err()
}

// AddBuildingBridge satisfies the BuildingAdminService gRPC server API.
func (a *API) AddBuildingBridge(ctx context.Context, req *AddBridgeRequest) (*Building, error) {
	return nil, ErrNotImplemented.Err()
}

// RemoveBuildingBridge satisfies the BuildingAdminService gRPC server API.
func (a *API) RemoveBuildingBridge(ctx context.Context, req *RemoveBridgeRequest) (*Building, error) {
	return nil, ErrNotImplemented.Err()
}

// CreateFloor satisfies the BuildingAdminService gRPC server API.
func (a *API) CreateFloor(ctx context.Context, req *CreateFloorRequest) (*Floor, error) {
	err := a.svc.AddFloor(ctx, req.Floor, req.BuildingId)
	if err != nil {
		a.logger.Info("error creating floor",
			zap.String("building_id", req.BuildingId),
			zap.Error(err),
		)
		return nil, ErrFloorCreateFailed.Err()
	}

	return req.Floor, nil
}

// UpdateFloor satisfies the BuildingAdminService gRPC server API.
func (a *API) UpdateFloor(ctx context.Context, req *UpdateFloorRequest) (*Floor, error) {
	return nil, ErrNotImplemented.Err()
}

// DeleteFloor satisfies the BuildingAdminService gRPC server API.
func (a *API) DeleteFloor(ctx context.Context, req *DeleteFloorRequest) (*empty.Empty, error) {
	return &empty.Empty{}, ErrNotImplemented.Err()
}

// CreateRoom satisfies the BuildingAdminService gRPC server API.
func (a *API) CreateRoom(ctx context.Context, req *CreateRoomRequest) (*Room, error) {
	err := a.svc.AddRoom(ctx, req.Room, req.FloorId)
	if err != nil {
		a.logger.Info("error creating room",
			zap.String("floor_id", req.FloorId),
			zap.Error(err),
		)
		return nil, ErrRoomCreateFailed.Err()
	}

	return req.Room, nil
}

// UpdateRoom satisfies the BuildingAdminService gRPC server API.
func (a *API) UpdateRoom(ctx context.Context, req *UpdateRoomRequest) (*Room, error) {
	return nil, ErrNotImplemented.Err()
}

// DeleteRoom satisfies the BuildingAdminService gRPC server API.
func (a *API) DeleteRoom(ctx context.Context, req *DeleteRoomRequest) (*empty.Empty, error) {
	return &empty.Empty{}, ErrNotImplemented.Err()
}

// ListBuildings satisfies the BuildingService gRPC server API.
func (a *API) ListBuildings(ctx context.Context, req *ListBuildingsRequest) (*ListBuildingsResponse, error) {
	buildings, err := a.svc.GetBuildings(ctx)
	if err != nil {
		return nil, err
	}

	return &ListBuildingsResponse{
		Buildings: buildings,
	}, nil
}

// GetBuilding satisfies the BuildingService gRPC server API.
func (a *API) GetBuilding(ctx context.Context, req *GetBuildingRequest) (*Building, error) {
	building, err := a.svc.GetBuilding(ctx, req.BuildingId)
	if err != nil {
		// err is going to be a wrapped status already.
		return nil, err
	}

	return building, nil
}

// ListFloors satisfies the BuildingService gRPC server API.
func (a *API) ListFloors(ctx context.Context, req *ListFloorsRequest) (*ListFloorsResponse, error) {
	floors, err := a.svc.GetFloors(ctx, req.BuildingId)
	if err != nil {
		return nil, err
	}

	return &ListFloorsResponse{
		Floors: floors,
	}, nil

}

// GetFloor satisfies the BuildingService gRPC server API.
func (a *API) GetFloor(ctx context.Context, req *GetFloorRequest) (*Floor, error) {
	floor, err := a.svc.GetFloor(ctx, req.Id)
	if err != nil {
		// err is going to be a wrapped status already.
		return nil, err
	}

	return floor, nil
}

// StreamBuildingUpdates satisfies the BuildingService gRPC server API.
func (a *API) StreamBuildingUpdates(req *StreamBuildingUpdatesRequest, stream BuildingService_StreamBuildingUpdatesServer) error {
	return ErrNotImplemented.Err()
}
