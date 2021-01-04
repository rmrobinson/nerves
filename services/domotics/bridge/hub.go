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
//
// The hub may have many consumers interested in its updates - these are managed by the updateSource property.
// Updates are published to this source by a single goroutine which 'owns' changing the bridge and device maps.
type Hub struct {
	logger          *zap.Logger
	updateSource    *stream.Source
	internalUpdates chan *Update

	devices      map[string]*hubDevice
	devicesMutex sync.RWMutex

	bridges      map[string]*hubBridge
	bridgesMutex sync.RWMutex
}

// NewHub creates a new hub with the supplied logger.
func NewHub(logger *zap.Logger) *Hub {
	ret := &Hub{
		logger:          logger,
		updateSource:    stream.NewSource(logger),
		internalUpdates: make(chan *Update, 100),
		devices:         map[string]*hubDevice{},
		bridges:         map[string]*hubBridge{},
	}

	go ret.processUpdates()

	return ret
}

// ListDevices returns the set of currently managed devices and their currently known states
func (h *Hub) ListDevices() ([]*Device, error) {
	resp := []*Device{}

	h.devicesMutex.RLock()
	defer h.devicesMutex.RUnlock()
	for _, device := range h.devices {
		resp = append(resp, proto.Clone(device.d).(*Device))
	}

	return resp, nil
}

// GetDevice retrieves the specified device.
func (h *Hub) GetDevice(id string) (*Device, error) {
	h.devicesMutex.RLock()
	defer h.devicesMutex.RUnlock()

	if device, found := h.devices[id]; found {
		// We clone the item before returning to avoid issues with sets being ignored.
		return proto.Clone(device.d).(*Device), nil
	}

	return nil, ErrDeviceNotFound.Err()
}

// UpdateDeviceConfig updates the specified device with the provided config.
func (h *Hub) UpdateDeviceConfig(ctx context.Context, id string, config *DeviceConfig) (*Device, error) {
	// This is only a read lock since we aren't mutating the device here - we just require it not change during our call.
	h.devicesMutex.RLock()
	defer h.devicesMutex.RUnlock()

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
	// This is only a read lock since we aren't mutating the device here - we just require it not change during our call.
	h.devicesMutex.RLock()
	defer h.devicesMutex.RUnlock()

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
	h.bridgesMutex.RLock()
	defer h.bridgesMutex.RUnlock()

	if hb, found := h.bridges[id]; found {
		return hb.b, nil
	}

	return nil, ErrBridgeNotFound.Err()
}

// AddBridge takes the supplied bridge services client, checks to see if it is already present,
// and if not adds the bridge client to the set of bridges active.
func (h *Hub) AddBridge(c BridgeServiceClient) error {
	// This method is the one exception to the case of writes occurring in the processing goroutine.
	// Because we are not talking about 'updates' to a bridge but actual editing the bridge set, we
	// lock the bridge map for writing here.
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
	h.bridges[hb.b.Id] = hb

	h.internalUpdates <- &Update{
		Action: Update_ADDED,
		Update: &Update_BridgeUpdate{
			&BridgeUpdate{
				Bridge:   hb.b,
				BridgeId: hb.b.Id,
			},
		},
	}

	for _, device := range brInfo.Devices {
		h.internalUpdates <- &Update{
			Action: Update_ADDED,
			Update: &Update_DeviceUpdate{
				&DeviceUpdate{
					Device:   device,
					DeviceId: device.Id,
					BridgeId: hb.b.Id,
				},
			},
		}
	}

	go h.processBridgeStream(hb)

	return nil
}

func (h *Hub) processBridgeStream(hb *hubBridge) {
	ctx, cancel := context.WithCancel(context.Background())
	hb.streamCancel = cancel
	defer func() {
		if hb.streamCancel != nil {
			hb.streamCancel()
			hb.streamCancel = nil
		}
	}()

	logger := h.logger.With(zap.String("bridge_id", hb.b.Id))

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
			h.RemoveBridge(hb.b.Id)
			return
		} else if err != nil {
			logger.Error("error watching bridge, removing",
				zap.Error(err),
			)

			// We assume that a bridge whose stream has an error is unavailable
			h.RemoveBridge(hb.b.Id)
			return
		}

		// Guard against remote services which don't properly annotate the update.
		if msg.GetBridgeUpdate() != nil {
			msg.GetBridgeUpdate().BridgeId = hb.b.Id
		} else if msg.GetDeviceUpdate() != nil {
			msg.GetDeviceUpdate().BridgeId = hb.b.Id
		}

		h.internalUpdates <- msg
	}
}

// RemoveBridge takes the specified bridge ID out of the system.
// It will not close the socket, if it is still open, but it will cancel any streaming requests.
// This will trigger a state change for any devices owned by this bridge, marking them as offline.
// Removing a bridge does not remove the device, as it is expected that a bridge going away is temporary.
func (h *Hub) RemoveBridge(id string) error {
	h.bridgesMutex.RLock()
	defer h.bridgesMutex.RUnlock()

	hb, found := h.bridges[id]
	if !found {
		h.logger.Info("removing a bridge that is already gone",
			zap.String("bridge_id", id),
		)
		return ErrBridgeNotFound.Err()
	}

	// We mark all of the devices owned by this bridge as unavailable
	h.devicesMutex.RLock()
	defer h.devicesMutex.RUnlock()
	for deviceID, hd := range h.devices {
		if hd.hb != hb {
			continue
		}

		h.devices[deviceID].d.State.IsReachable = false

		h.internalUpdates <- &Update{
			Action: Update_CHANGED,
			Update: &Update_DeviceUpdate{
				&DeviceUpdate{
					Device:   h.devices[deviceID].d,
					DeviceId: deviceID,
					BridgeId: hb.b.Id,
				},
			},
		}
	}

	h.internalUpdates <- &Update{
		Action: Update_REMOVED,
		Update: &Update_BridgeUpdate{
			&BridgeUpdate{
				BridgeId: hb.b.Id,
			},
		},
	}

	if hb.streamCancel != nil {
		hb.streamCancel()
		hb.streamCancel = nil
	}

	return nil
}

// Updates exposes the stream of received changes to the underlying bridges and devices.
func (h *Hub) Updates() <-chan *Update {
	ret := make(chan *Update)

	go func() {
		sink := h.updateSource.NewSink()
		defer sink.Close()
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

func (h *Hub) processUpdates() {
	// Since any action in this method is the only one manipulating the maps (except for bridge add)
	// we can be guaranteed that map entries will exist or will not; we don't need to guard them at check time.
	// We do, however, need to guard writes as other callers may be reading the map values.
	for {
		update := <-h.internalUpdates

		// Process bridge updates
		if update.GetBridgeUpdate() != nil {
			h.logger.Info("bridge update",
				zap.String("action", update.Action.String()),
				zap.String("bridge_id", update.GetBridgeUpdate().BridgeId),
			)

			// Added updates are already handled by the caller.
			if update.Action == Update_CHANGED {
				h.bridgesMutex.Lock()
				if hb, found := h.bridges[update.GetBridgeUpdate().BridgeId]; found {
					hb.b = proto.Clone(update.GetBridgeUpdate().Bridge).(*Bridge)
				} else {
					h.logger.Info("received bridge changed call for non-existent bridge",
						zap.String("bridge_id", update.GetBridgeUpdate().BridgeId),
					)
				}
				h.bridgesMutex.Unlock()
			} else if update.Action == Update_REMOVED {
				h.bridgesMutex.Lock()
				delete(h.bridges, update.GetBridgeUpdate().BridgeId)
				h.bridgesMutex.Unlock()
			}

			// Pass this through to the external update channel for refreshing.
			h.updateSource.SendMessage(update)
			continue
		}

		// Process device updates
		if deviceUpdate := update.GetDeviceUpdate(); deviceUpdate != nil {
			if deviceUpdate.Device == nil && len(deviceUpdate.DeviceId) < 1 {
				h.logger.Info("received malformed device update, ignoring",
					zap.String("msg", update.String()),
					zap.String("action", update.Action.String()),
					zap.String("bridge_id", deviceUpdate.BridgeId),
				)
				return
			}

			if deviceUpdate.Device != nil {
				deviceUpdate.DeviceId = deviceUpdate.Device.Id
			}
			deviceID := deviceUpdate.DeviceId

			h.logger.Info("device update",
				zap.String("action", update.Action.String()),
				zap.String("bridge_id", deviceUpdate.BridgeId),
				zap.String("device_id", deviceID),
			)

			hb, found := h.bridges[deviceUpdate.BridgeId]
			if !found {
				h.logger.Info("received device update for non-existent bridge",
					zap.String("device_id", deviceID),
					zap.String("bridge_id", deviceUpdate.BridgeId),
				)
				continue
			}

			h.devicesMutex.Lock()
			switch update.Action {
			case Update_ADDED:
				if hubd, found := h.devices[deviceID]; found {
					h.logger.Info("received an 'added' event for a device ID already present, replacing",
						zap.String("device_id", deviceID),
						zap.String("bridge_id", hubd.hb.b.Id),
					)

					delete(h.devices, deviceID)
				}
				h.devices[deviceID] = &hubDevice{
					d:  proto.Clone(deviceUpdate.Device).(*Device),
					hb: hb,
				}
			case Update_REMOVED:
				delete(h.devices, deviceID)
			case Update_CHANGED:
				h.devices[deviceID].d = proto.Clone(deviceUpdate.Device).(*Device)
			default:
				h.logger.Info("received update with unsupported action",
					zap.Int("action", int(update.Action)),
				)
			}
			h.devicesMutex.Unlock()

			// Pass this through to the external update channel for refreshing.
			h.updateSource.SendMessage(update)
			continue
		}
	}
}
