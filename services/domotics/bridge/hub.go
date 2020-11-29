package bridge

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/rmrobinson/nerves/lib/stream"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrBridgeAlreadyAdded is returned if the requested bridge already exists
	ErrBridgeAlreadyAdded = errors.New("bridge already added")
	// ErrBridgeNotFound is returned if the requested bridge ID could not be found
	ErrBridgeNotFound = status.New(codes.NotFound, "bridge not found")
)

type hubBridge struct {
	// b contains info about the bridge being referenced
	b *Bridge

	// c is the client connecting to this bridge
	c BridgeServiceClient

	// used to cancel the update stream on c
	streamCancel context.CancelFunc
}

type hubDevice struct {
	d  *Device
	hb *hubBridge
}

// Hub abstracts the management of multiple bridges and their dependent devices.
// It is designed to simplify the implementation of clients which operate in a multi-bridge environment.
// The actual bridge a given device is controlled by is abstracted through the Hub API.
type Hub struct {
	logger       *zap.Logger
	hubInfo      *Bridge
	updateSource *stream.Source

	devices      map[string]*hubDevice
	devicesMutex sync.Mutex

	bridges      map[string]*hubBridge
	bridgesMutex sync.Mutex
}

// NewHub creates a new hub with the supplied logger.
func NewHub(logger *zap.Logger, hubInfo *Bridge) *Hub {
	return &Hub{
		logger:       logger,
		hubInfo:      hubInfo,
		updateSource: stream.NewSource(logger),
		devices:      map[string]*hubDevice{},
		bridges:      map[string]*hubBridge{},
	}
}

// ListDevices returns the set of currently managed devices and their currently known states
func (h *Hub) ListDevices() ([]*Device, error) {
	resp := []*Device{}

	h.devicesMutex.Lock()
	defer h.devicesMutex.Unlock()
	for _, device := range h.devices {
		resp = append(resp, proto.Clone(device.d).(*Device))
	}

	return resp, nil
}

// GetDevice retrieves the specified device.
func (h *Hub) GetDevice(id string) (*Device, error) {
	h.devicesMutex.Lock()
	defer h.devicesMutex.Unlock()

	if device, found := h.devices[id]; found {
		// We clone the item before returning to avoid issues with sets being ignored.
		return proto.Clone(device.d).(*Device), nil
	}

	return nil, ErrDeviceNotFound.Err()
}

// UpdateDeviceConfig updates the specified device with the provided config.
func (h *Hub) UpdateDeviceConfig(ctx context.Context, id string, config *DeviceConfig) (*Device, error) {
	h.devicesMutex.Lock()
	defer h.devicesMutex.Unlock()

	hd, found := h.devices[id]

	if !found {
		return nil, ErrDeviceNotFound.Err()
	}

	if config.String() == hd.d.Config.String() {
		h.logger.Debug("noop write, ignoring",
			zap.String("device_id", id),
		)
		return proto.Clone(hd.d).(*Device), nil
	}

	// TODO: check against request version field

	resp, err := hd.hb.c.UpdateDeviceConfig(ctx, &UpdateDeviceConfigRequest{
		Id:     id,
		Config: config,
	})
	if err != nil {
		h.logger.Info("error setting device config",
			zap.String("id", id),
			zap.Error(err),
		)

		// Since we're invoking a gRPC call here we will have a grpc.Status type to pass along.
		// There is no need to manipulate the error as we're just proxying it through.
		return nil, err
	}

	// We do not update the 'devices' array here since the contract guarantees we'll be getting
	// an update from the owning bridge with this state change.
	return resp, nil
}

// UpdateDeviceState updates the specified device with the provided state.
func (h *Hub) UpdateDeviceState(ctx context.Context, id string, state *DeviceState) (*Device, error) {
	h.devicesMutex.Lock()
	defer h.devicesMutex.Unlock()

	hd, found := h.devices[id]

	if !found {
		return nil, ErrDeviceNotFound.Err()
	}

	if state.String() == hd.d.State.String() {
		h.logger.Debug("noop write, ignoring",
			zap.String("device_id", id),
		)
		return proto.Clone(hd.d).(*Device), nil
	}

	// TODO: check against request version field

	resp, err := hd.hb.c.UpdateDeviceState(ctx, &UpdateDeviceStateRequest{
		Id:    id,
		State: state,
	})
	if err != nil {
		h.logger.Info("error setting device state",
			zap.String("id", id),
			zap.Error(err),
		)

		// Since we're invoking a gRPC call here we will have a grpc.Status type to pass along.
		// There is no need to manipulate the error as we're just proxying it through.
		return nil, err
	}

	// We do not update the 'devices' array here since the contract guarantees we'll be getting
	// an update from the owning bridge with this state change.
	return resp, nil
}

// Bridge allows the caller to retrieve the bridge information, if present.
func (h *Hub) Bridge(id string) (*Bridge, error) {
	h.bridgesMutex.Lock()
	defer h.bridgesMutex.Unlock()

	if hb, found := h.bridges[id]; found {
		return hb.b, nil
	}

	return nil, ErrBridgeNotFound.Err()
}

// AddBridge takes the supplied bridge services client, checks to see if it is already present,
// and if not adds the bridge client to the set of bridges active.
func (h *Hub) AddBridge(c BridgeServiceClient) error {
	h.bridgesMutex.Lock()
	defer h.bridgesMutex.Unlock()

	brInfo, err := c.GetBridge(context.Background(), &GetBridgeRequest{})
	if err != nil {
		h.logger.Info("error retrieve bridge info during addition",
			zap.Error(err),
		)

		return err
	}

	if _, found := h.bridges[brInfo.Id]; found {
		return ErrBridgeAlreadyAdded
	}

	hb := &hubBridge{
		b: brInfo,
		c: c,
	}

	bridgeID := brInfo.Id
	logger := h.logger.With(zap.String("bridge_id", bridgeID))
	h.bridges[bridgeID] = hb

	h.updateSource.SendMessage(&Update{
		Action: Update_ADDED,
		Update: &Update_BridgeUpdate{
			&BridgeUpdate{
				Bridge:   hb.b,
				BridgeId: hb.b.Id,
			},
		},
	})

	for _, device := range brInfo.Devices {
		h.processUpdate(hb, &Update{
			Action: Update_ADDED,
			Update: &Update_DeviceUpdate{
				&DeviceUpdate{
					Device: device,
				},
			},
		})
	}

	go func(h *Hub, hb *hubBridge) {
		ctx, cancel := context.WithCancel(context.Background())
		hb.streamCancel = cancel
		defer cancel()

		stream, err := hb.c.StreamBridgeUpdates(ctx, &StreamBridgeUpdatesRequest{})
		if err != nil {
			logger.Error("error attempting to stream bridge updates",
				zap.Error(err),
			)
			return
		}

		for {
			msg, err := stream.Recv()

			if ctx.Err() == context.Canceled {
				// If we are seeing a cancelled error here we have already removed the bridge.
				// Simply exit the steam.
				logger.Info("bridge stream cancelled")
				return
			}

			if err == io.EOF {
				logger.Info("bridge connection went away")

				// We assume that a bridge whose stream is gone is unavailable
				h.RemoveBridge(bridgeID)
				return
			} else if err != nil {
				logger.Error("error watching bridge, removing",
					zap.Error(err),
				)

				// We assume that a bridge whose stream has an error is unavailable
				h.RemoveBridge(bridgeID)
				return
			}

			h.processUpdate(hb, msg)
		}
	}(h, hb)
	return nil
}

// RemoveBridge takes the specified bridge ID out of the system.
// It will not close the socket, if it is still open, but it will cancel any streaming requests.
// This will trigger a state change for any devices owned by this bridge, marking them as offline.
// Removing a bridge does not remove the device, as it is expected that a bridge going away is temporary.
func (h *Hub) RemoveBridge(id string) error {
	h.bridgesMutex.Lock()
	defer h.bridgesMutex.Unlock()

	hb, found := h.bridges[id]
	if !found {
		return ErrBridgeNotFound.Err()
	}

	deviceIDs := []string{}
	// We mark all of the devices owned by this bridge as unavailable
	h.devicesMutex.Lock()
	for deviceID, hd := range h.devices {
		if hd.hb != hb {
			continue
		}

		deviceIDs = append(deviceIDs, deviceID)
		h.devices[deviceID].d.State.IsReachable = false
	}
	h.devicesMutex.Unlock()

	// processUpdate() locks the devices map - we retrieved the known set of devices above
	// and the iterate through them here to ensure no deadlocks.
	for _, deviceID := range deviceIDs {
		h.processUpdate(hb, &Update{
			Action: Update_CHANGED,
			Update: &Update_DeviceUpdate{
				&DeviceUpdate{
					Device: h.devices[deviceID].d,
				},
			},
		})
	}

	// Next we remove this bridge from the set
	delete(h.bridges, id)

	h.updateSource.SendMessage(&Update{
		Action: Update_REMOVED,
		Update: &Update_BridgeUpdate{
			&BridgeUpdate{
				BridgeId: hb.b.Id,
			},
		},
	})

	if hb.streamCancel != nil {
		hb.streamCancel()
	}

	return nil
}

// Updates exposes the stream of received changes to the underlying bridges and devices.
func (h *Hub) Updates() <-chan *Update {
	ret := make(chan *Update)

	go func() {
		sink := h.updateSource.NewSink()
		for {
			u, ok := <-sink.Messages()
			if !ok {
				// Channel has been closed; so we'll close the connection as well
				close(ret)
				return
			}

			update, ok := u.(*Update)
			if !ok {
				panic("bridge update cast failed")
			}

			ret <- update
		}
	}()
	return ret
}

func (h *Hub) processUpdate(hb *hubBridge, u *Update) {
	if u.GetBridgeUpdate() != nil {
		h.logger.Info("bridge updated",
			zap.String("action", u.Action.String()),
			zap.String("bridge_info", u.GetBridgeUpdate().Bridge.String()),
		)

		h.bridgesMutex.Lock()
		hb.b = u.GetBridgeUpdate().Bridge
		h.bridgesMutex.Unlock()
	} else if u.GetDeviceUpdate() != nil {
		if u.GetDeviceUpdate().Device == nil && len(u.GetDeviceUpdate().DeviceId) < 1 {
			h.logger.Info("received malformed device update, ignoring",
				zap.String("msg", u.String()),
				zap.String("action", u.Action.String()),
				zap.String("bridge_id", u.GetDeviceUpdate().BridgeId),
			)
			return
		}

		deviceID := u.GetDeviceUpdate().DeviceId
		if u.GetDeviceUpdate().Device != nil {
			deviceID = u.GetDeviceUpdate().Device.Id
		}

		h.logger.Info("device updated",
			zap.String("action", u.Action.String()),
			zap.String("bridge_id", u.GetDeviceUpdate().BridgeId),
			zap.String("device_id", deviceID),
			zap.String("device_info", u.GetDeviceUpdate().Device.String()),
		)

		h.devicesMutex.Lock()
		switch u.Action {
		case Update_ADDED:
			if hubd, found := h.devices[deviceID]; found {
				h.logger.Info("received an 'added' event for a device ID already present, replacing",
					zap.String("device_id", deviceID),
					zap.String("existing_bridge_id", hubd.hb.b.Id),
					zap.String("new_bridge_id", u.GetDeviceUpdate().BridgeId),
				)

				delete(h.devices, deviceID)
			}
			h.devices[deviceID] = &hubDevice{
				d:  proto.Clone(u.GetDeviceUpdate().Device).(*Device),
				hb: hb,
			}
		case Update_REMOVED:
			delete(h.devices, deviceID)
		case Update_CHANGED:
			h.devices[deviceID].d = proto.Clone(u.GetDeviceUpdate().Device).(*Device)
		default:
			h.logger.Info("received update with unsupported action",
				zap.Int("action", int(u.Action)),
			)
		}
		h.devicesMutex.Unlock()

		// Pass this through to the external update channel for refreshing.
		h.updateSource.SendMessage(u)
	}
}
