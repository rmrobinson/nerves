package domotics

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ProxyHub is a hub implementation that proxies requests to a specified service
type ProxyHub struct {
	conn   *grpc.ClientConn
	logger *zap.Logger

	hub       *Hub
	instances map[string]*proxyInstance
}

// NewProxyBridge creates a bridge implementation from a supplied bridge client.
func NewProxyBridge(logger *zap.Logger, hub *Hub, conn *grpc.ClientConn) *ProxyHub {
	return &ProxyHub{
		logger:    logger,
		hub:       hub,
		conn:      conn,
		instances: map[string]*proxyInstance{},
	}
}

// Run monitors the bridge and device channels for updates and propagates them to the monitors subscribed to the proxy.
func (p *ProxyHub) Run() {
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		p.runBridgeMonitor()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.runDeviceMonitor()
	}()

	wg.Wait()
}

func (p *ProxyHub) runBridgeMonitor() {
	bc := NewBridgeServiceClient(p.conn)
	stream, err := bc.StreamBridgeUpdates(context.Background(), &StreamBridgeUpdatesRequest{})
	if err != nil {
		return
	}

	for {
		if update, err := stream.Recv(); err == nil {
			p.logger.Debug("received bridge update",
				zap.String("bridge_info", update.Bridge.String()),
			)

			switch update.Action {
			case BridgeUpdate_ADDED:
				pi := newProxyInstance(p.conn, update.Bridge.Id)
				if err := p.hub.AddAsyncBridge(pi); err != nil {
					p.logger.Warn("error adding bridge",
						zap.String("bridge_id", update.Bridge.Id),
						zap.Error(err),
					)
				}
				p.instances[update.Bridge.Id] = pi
			case BridgeUpdate_CHANGED:
				pi, ok := p.instances[update.Bridge.Id]
				if !ok {
					p.logger.Warn("received update but wasn't registered",
						zap.String("bridge_id", update.Bridge.Id),
					)
					continue
				}
				if err := pi.notifier.BridgeUpdated(update.Bridge); err != nil {
					p.logger.Warn("error updating bridge",
						zap.String("bridge_id", update.Bridge.Id),
						zap.Error(err),
					)
				}
			case BridgeUpdate_REMOVED:
				pi, ok := p.instances[update.Bridge.Id]
				if !ok {
					p.logger.Warn("received remove but wasn't registered",
						zap.String("bridge_id", update.Bridge.Id),
					)
					continue
				}

				if err := p.hub.RemoveBridge(pi.bridgeID); err != nil {
					p.logger.Warn("error removing bridge",
						zap.String("bridge_id", update.Bridge.Id),
						zap.Error(err),
					)
				}
				delete(p.instances, pi.bridgeID)
			}
		} else {
			p.logger.Warn("error monitoring bridges",
				zap.Error(err),
			)

			for bridgeID := range p.instances {
				p.logger.Debug("removing bridge due to connection error",
					zap.String("bridge_id", bridgeID),
				)
				if err := p.hub.RemoveBridge(bridgeID); err != nil {
					p.logger.Warn("error removing bridge due to connection error",
						zap.String("bridge_id", bridgeID),
						zap.Error(err),
					)
				}
				delete(p.instances, bridgeID)
			}
			return
		}
	}
}

func (p *ProxyHub) runDeviceMonitor() {
	dc := NewDeviceServiceClient(p.conn)
	stream, err := dc.StreamDeviceUpdates(context.Background(), &StreamDeviceUpdatesRequest{})
	if err != nil {
		return
	}

	for {
		if update, err := stream.Recv(); err == nil {
			p.logger.Debug("received device update",
				zap.String("device_info", update.Device.String()),
			)

			switch update.Action {
			case DeviceUpdate_ADDED:
				if err := p.hub.DeviceAdded(update.BridgeId, update.Device); err != nil {
					p.logger.Warn("error adding device",
						zap.String("device_id", update.Device.Id),
						zap.Error(err),
					)
				}
			case DeviceUpdate_CHANGED:
				if err := p.hub.DeviceUpdated(update.BridgeId, update.Device); err != nil {
					p.logger.Warn("error updating device",
						zap.String("device_id", update.Device.Id),
						zap.Error(err),
					)
				}
			case DeviceUpdate_REMOVED:
				if err := p.hub.DeviceRemoved(update.BridgeId, update.Device); err != nil {
					p.logger.Warn("error removing device",
						zap.String("device_id", update.Device.Id),
						zap.Error(err),
					)
				}
			}
		} else {
			p.logger.Warn("error monitoring devices",
				zap.Error(err),
			)
			return
		}
	}
}
