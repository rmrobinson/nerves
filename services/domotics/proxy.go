package domotics

import (
	"context"
	"log"
	"sync"

	"google.golang.org/grpc"
)

// ProxyHub is a hub implementation that proxies requests to a specified service
type ProxyHub struct {
	conn *grpc.ClientConn

	hub       *Hub
	instances map[string]*proxyInstance
}

// NewProxyBridge creates a bridge implementation from a supplied bridge client.
func NewProxyBridge(hub *Hub, conn *grpc.ClientConn) *ProxyHub {
	return &ProxyHub{
		hub:  hub,
		conn: conn,
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
			log.Printf("Received bridge update %+v\n", update.Bridge)

			switch update.Action {
			case BridgeUpdate_ADDED:
				pi := newProxyInstance(p.conn, update.Bridge.Id)
				if err := p.hub.AddAsyncBridge(pi); err != nil {
					log.Printf("Error adding bridge %s: %s\n", update.Bridge.Id, err.Error())
				}
				p.instances[update.Bridge.Id] = pi
			case BridgeUpdate_CHANGED:
				pi, ok := p.instances[update.Bridge.Id]
				if !ok {
					log.Printf("Received update for %s but wasn't registered", update.Bridge.Id)
					continue
				}
				if err := pi.notifier.BridgeUpdated(update.Bridge); err != nil {
					log.Printf("Error updating bridge %s: %s\n", update.Bridge.Id, err.Error())
				}
			case BridgeUpdate_REMOVED:
				pi, ok := p.instances[update.Bridge.Id]
				if !ok {
					log.Printf("Received remove for %s but wasn't registered", update.Bridge.Id)
					continue
				}

				if err := p.hub.RemoveBridge(pi.bridgeID); err != nil {
					log.Printf("Error removing bridge %s: %s\n", update.Bridge.Id, err.Error())
				}
				delete(p.instances, pi.bridgeID)
			}
		} else {
			log.Printf("Error while monitoring bridges: %s\n", err.Error())

			for bridgeID := range p.instances {
				log.Printf("Removing bridge ID %s due to connection error\n", bridgeID)
				if err := p.hub.RemoveBridge(bridgeID); err != nil {
					log.Printf("Error removing bridge %s: %s\n", update.Bridge.Id, err.Error())
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
			log.Printf("Received device update %+v\n", update.Device)

			switch update.Action {
			case DeviceUpdate_ADDED:
				if err := p.hub.DeviceAdded(update.BridgeId, update.Device); err != nil {
					log.Printf("Error adding device %v: %s\n", update.Device, err.Error())
				}
			case DeviceUpdate_CHANGED:
				if err := p.hub.DeviceUpdated(update.BridgeId, update.Device); err != nil {
					log.Printf("Error updating device %v: %s\n", update.Device, err.Error())
				}
			case DeviceUpdate_REMOVED:
				if err := p.hub.DeviceRemoved(update.BridgeId, update.Device); err != nil {
					log.Printf("Error removing device %v: %s\n", update.Device, err.Error())
				}
			}
		} else {
			log.Printf("Error while monitoring devices: %s\n", err.Error())
			return
		}
	}
}
