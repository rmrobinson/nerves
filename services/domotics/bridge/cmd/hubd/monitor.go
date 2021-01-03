package main

import (
	"context"
	"sync"

	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

// HubMonitor is a hub-based implementation of a monitor.
// Changes in monitor state will trigger attempts to connect to the new bridges.
type HubMonitor struct {
	logger     *zap.Logger
	conns      map[string]*grpc.ClientConn
	connsMutex sync.Mutex

	hub *bridge.Hub

	brInfo *bridge.Bridge
}

// Alive is called when a bridge is reporting itself as alive
func (hm *HubMonitor) Alive(t string, id string, connStr string) {
	hm.connsMutex.Lock()
	defer hm.connsMutex.Unlock()

	// We only manage the bridge types for now
	if t != "falnet_nerves:bridge" {
		return
	}
	// Let's not loop on ourselves
	if id == "uuid:"+hm.brInfo.Id {
		return
	}

	if conn, exists := hm.conns[id]; exists {
		pingClient := bridge.NewPingServiceClient(conn)

		_, err := pingClient.Ping(context.Background(), &bridge.PingRequest{})
		if err == nil {
			_, err := hm.hub.Bridge(id)
			if err == bridge.ErrBridgeNotFound.Err() {
				hm.logger.Info("adding bridge on existing connection",
					zap.String("id", id),
				)

				hm.hub.AddBridge(bridge.NewBridgeServiceClient(conn))
				return
			}
			return
		}

		hm.logger.Info("unable to make ping request on active conn, closing",
			zap.String("id", id),
			zap.Error(err),
		)

		conn.Close()
		delete(hm.conns, id)
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(connStr, opts...)
	if err != nil {
		hm.logger.Error("unable to connect",
			zap.String("conn_str", connStr),
			zap.Error(err),
		)
		return
	}

	hm.hub.AddBridge(bridge.NewBridgeServiceClient(conn))
	hm.conns[id] = conn
}

// GoingAway is called when a bridge is reporting itself as going aways
func (hm *HubMonitor) GoingAway(id string) {
	var conn *grpc.ClientConn
	var exists bool

	hm.connsMutex.Lock()
	defer hm.connsMutex.Unlock()

	if conn, exists = hm.conns[id]; !exists {
		hm.logger.Debug("ignoring bye as id not registered",
			zap.String("id", id),
		)
		return
	}

	hm.hub.RemoveBridge(id)

	conn.Close()
	delete(hm.conns, id)
}

// The methods below allow the monitor to act as a virtual bridge for all the connected bridges.

// GetBridge retrieves the bridge info of this service.
func (hm *HubMonitor) GetBridge(ctx context.Context, req *bridge.GetBridgeRequest) (*bridge.Bridge, error) {
	ret := proto.Clone(hm.brInfo).(*bridge.Bridge)

	devices, err := hm.hub.ListDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		ret.Devices = append(ret.Devices, device)
	}

	return ret, nil
}

// ListDevices retrieves all registered devices.
func (hm *HubMonitor) ListDevices(ctx context.Context, req *bridge.ListDevicesRequest) (*bridge.ListDevicesResponse, error) {
	resp := &bridge.ListDevicesResponse{}

	devices, err := hm.hub.ListDevices()
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		resp.Devices = append(resp.Devices, device)
	}

	return resp, nil
}

// GetDevice retrieves the specified device.
func (hm *HubMonitor) GetDevice(ctx context.Context, req *bridge.GetDeviceRequest) (*bridge.Device, error) {
	return hm.hub.GetDevice(req.Id)
}

// UpdateDeviceConfig exists to satisfy the domotics Bridge contract, but is not actually supported.
func (hm *HubMonitor) UpdateDeviceConfig(ctx context.Context, req *bridge.UpdateDeviceConfigRequest) (*bridge.Device, error) {
	return hm.hub.UpdateDeviceConfig(ctx, req.Id, req.Config)
}

// UpdateDeviceState updates the specified device with the provided state.
func (hm *HubMonitor) UpdateDeviceState(ctx context.Context, req *bridge.UpdateDeviceStateRequest) (*bridge.Device, error) {
	return hm.hub.UpdateDeviceState(ctx, req.Id, req.State)
}

// StreamBridgeUpdates monitors changes for all changes which occur on the
// This will only pick up successful device writes.
func (hm *HubMonitor) StreamBridgeUpdates(req *bridge.StreamBridgeUpdatesRequest, stream bridge.BridgeService_StreamBridgeUpdatesServer) error {
	peer, isOk := peer.FromContext(stream.Context())

	addr := "unknown"
	if isOk {
		addr = peer.Addr.String()
	}

	logger := hm.logger.With(zap.String("peer_addr", addr))

	logger.Debug("bridge update stream initiated")

	// Send all of the devices to start.
	devices, err := hm.hub.ListDevices()
	if err != nil {
		hm.logger.Error("unable to get devices during stream initialization",
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
					DeviceId: device.Id,
					BridgeId: hm.brInfo.Id,
				},
			},
		}

		if err := stream.Send(update); err != nil {
			logger.Error("unable to send update",
				zap.Error(err),
			)
			return err
		}
	}

	updates := hm.hub.Updates()
	// Now we wait for updates
	for {
		update, ok := <-updates
		if !ok {
			logger.Debug("stream closed")
			// Channel has been closed; so we'll close the connection as well
			return nil
		}

		logger.Debug("sending update")

		if err := stream.Send(update); err != nil {
			return err
		}
	}
}
