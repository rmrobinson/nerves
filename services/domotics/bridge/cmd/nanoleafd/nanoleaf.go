package main

import (
	"context"
	"fmt"

	"github.com/rmrobinson/nanoleaf-go"
	"github.com/rmrobinson/nerves/lib/stream"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
	"google.golang.org/grpc/peer"
)

// Nanoleaf is an implementation of a bridge for a Nanoleaf light panel or similar light fixture.
type Nanoleaf struct {
	logger *zap.Logger

	id string
	c  *nanoleaf.Client

	updates *stream.Source
}

// NewNanoleaf creates a new instance of a Nanoleaf bridge
func NewNanoleaf(logger *zap.Logger, id string, c *nanoleaf.Client) *Nanoleaf {
	return &Nanoleaf{
		logger:  logger,
		id:      id,
		c:       c,
		updates: stream.NewSource(logger),
	}
}

func panelToDevice(p *nanoleaf.LightPanel) *bridge.Device {
	return &bridge.Device{
		Id:       fmt.Sprintf("%s", p.SerialNumber),
		IsActive: true,
		Type:     bridge.DeviceType_LIGHT,
		Config: &bridge.DeviceConfig{
			Name: p.Name,
		},
		State: &bridge.DeviceState{
			IsReachable: true,
			Binary: &bridge.DeviceState_Binary{
				IsOn: p.State.On.Value,
			},
			ColorHsb: &bridge.DeviceState_ColorHSB{
				Hue:        int32(p.State.Hue.Value),
				Brightness: int32(p.State.Brightness.Value),
				Saturation: int32(p.State.Saturation.Value),
			},
			Version: &bridge.Version{
				Sw: p.FirmwareVersion,
			},
		},
	}
}

// GetBridge retrieves the bridge info of this service.
func (n *Nanoleaf) GetBridge(ctx context.Context, req *bridge.GetBridgeRequest) (*bridge.Bridge, error) {
	panel, err := n.c.GetPanel(ctx)
	if err != nil {
		n.logger.Error("unable to retrieve panel",
			zap.Error(err),
		)
		return nil, bridge.ErrInternal.Err()
	}

	resp := &bridge.Bridge{
		Id:           n.id,
		ModelId:      panel.ModelNumber,
		Manufacturer: panel.Manufacturer,
		Config:       &bridge.BridgeConfig{},
		State: &bridge.BridgeState{
			IsPaired: true,
			Version: &bridge.Version{
				Sw: panel.FirmwareVersion,
			},
		},
		Devices: []*bridge.Device{
			panelToDevice(panel),
		},
	}

	return resp, nil
}

// ListDevices retrieves all registered devices.
func (n *Nanoleaf) ListDevices(ctx context.Context, req *bridge.ListDevicesRequest) (*bridge.ListDevicesResponse, error) {
	panel, err := n.c.GetPanel(ctx)
	if err != nil {
		n.logger.Error("unable to retrieve panel",
			zap.Error(err),
		)
		return nil, bridge.ErrInternal.Err()
	}
	resp := &bridge.ListDevicesResponse{
		Devices: []*bridge.Device{
			panelToDevice(panel),
		},
	}

	return resp, nil
}

// GetDevice retrieves the specified device.
func (n *Nanoleaf) GetDevice(ctx context.Context, req *bridge.GetDeviceRequest) (*bridge.Device, error) {
	panel, err := n.c.GetPanel(ctx)
	if err != nil {
		n.logger.Error("unable to retrieve panel",
			zap.Error(err),
		)
		return nil, bridge.ErrInternal.Err()
	}

	device := panelToDevice(panel)
	if device.Id == req.Id {
		return device, nil
	}

	return nil, bridge.ErrDeviceNotFound.Err()
}

// UpdateDeviceConfig exists to satisfy the domotics Bridge contract, but is not actually supported.
func (n *Nanoleaf) UpdateDeviceConfig(ctx context.Context, req *bridge.UpdateDeviceConfigRequest) (*bridge.Device, error) {
	return nil, bridge.ErrNotSupported.Err()
}

// UpdateDeviceState updates the specified device with the provided state.
func (n *Nanoleaf) UpdateDeviceState(ctx context.Context, req *bridge.UpdateDeviceStateRequest) (*bridge.Device, error) {
	if len(req.Id) < 1 || req.State == nil {
		return nil, bridge.ErrMissingParam.Err()
	} else if req.State.IsReachable == false {
		return nil, bridge.ErrNotSupported.Err()
	}

	panel, err := n.c.GetPanel(ctx)
	if err != nil {
		n.logger.Error("unable to retrieve panel",
			zap.Error(err),
		)
		return nil, bridge.ErrInternal.Err()
	}

	device := panelToDevice(panel)

	if device.Id != req.Id {
		return nil, bridge.ErrDeviceNotFound.Err()
	}

	if req.State.String() == device.State.String() {
		n.logger.Debug("noop write, ignoring",
			zap.String("device_id", req.Id),
		)
		return device, nil
	}

	if req.State.Binary != nil && req.State.Binary.IsOn != panel.State.On.Value {
		err = n.c.SetOn(ctx, req.State.Binary.IsOn)
		if err != nil {
			n.logger.Error("unable to set nanoleaf binary state",
				zap.Bool("is_on", req.State.Binary.IsOn),
				zap.Error(err),
			)
			return nil, bridge.ErrInternal.Err()
		}
	}
	if req.State.ColorHsb != nil && req.State.ColorHsb.Brightness != int32(panel.State.Brightness.Value) {
		err = n.c.SetBrightness(ctx, int(req.State.ColorHsb.Brightness), 0)
		if err != nil {
			n.logger.Error("unable to set nanoleaf brightness state",
				zap.Int32("brightness", req.State.ColorHsb.Brightness),
				zap.Error(err),
			)
			return nil, bridge.ErrInternal.Err()
		}
	}
	if req.State.ColorHsb != nil && req.State.ColorHsb.Hue != int32(panel.State.Hue.Value) {
		err = n.c.SetHue(ctx, int(req.State.ColorHsb.Hue))
		if err != nil {
			n.logger.Error("unable to set nanoleaf hue state",
				zap.Int32("hue", req.State.ColorHsb.Hue),
				zap.Error(err),
			)
			return nil, bridge.ErrInternal.Err()
		}
	}
	if req.State.ColorHsb != nil && req.State.ColorHsb.Saturation != int32(panel.State.Saturation.Value) {
		err = n.c.SetSaturation(ctx, int(req.State.ColorHsb.Saturation))
		if err != nil {
			n.logger.Error("unable to set nanoleaf saturation state",
				zap.Int32("saturation", req.State.ColorHsb.Saturation),
				zap.Error(err),
			)
			return nil, bridge.ErrInternal.Err()
		}
	}

	// Now that everything is set, refresh the values before returning the final value
	panel, err = n.c.GetPanel(ctx)
	if err != nil {
		n.logger.Error("unable to retrieve panel",
			zap.Error(err),
		)
		return nil, bridge.ErrInternal.Err()
	}

	device = panelToDevice(panel)

	n.updates.SendMessage(&bridge.Update{
		Action: bridge.Update_CHANGED,
		Update: &bridge.Update_DeviceUpdate{
			DeviceUpdate: &bridge.DeviceUpdate{
				Device:   device,
				BridgeId: n.id,
			},
		},
	})

	return device, nil
}

// StreamBridgeUpdates monitors changes for all changes which occur on the
// This will only pick up successful device writes.
func (n *Nanoleaf) StreamBridgeUpdates(req *bridge.StreamBridgeUpdatesRequest, stream bridge.BridgeService_StreamBridgeUpdatesServer) error {
	peer, isOk := peer.FromContext(stream.Context())

	addr := "unknown"
	if isOk {
		addr = peer.Addr.String()
	}

	logger := n.logger.With(zap.String("peer_addr", addr))

	logger.Debug("bridge update stream initiated")

	sink := n.updates.NewSink()

	// Send the device info to start.

	panel, err := n.c.GetPanel(stream.Context())
	if err != nil {
		n.logger.Error("unable to retrieve panel",
			zap.Error(err),
		)
		return bridge.ErrInternal.Err()
	}

	device := panelToDevice(panel)
	update := &bridge.Update{
		Action: bridge.Update_ADDED,
		Update: &bridge.Update_DeviceUpdate{
			DeviceUpdate: &bridge.DeviceUpdate{
				Device:   device,
				BridgeId: n.id,
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

	// Now we wait for updates
	for {
		update, ok := <-sink.Messages()
		if !ok {
			logger.Debug("stream closed")
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
}
