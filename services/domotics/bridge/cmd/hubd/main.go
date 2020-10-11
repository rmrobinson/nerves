package main

import (
	"context"
	"sync"

	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// HubMonitor is a hub-based implementation of a monitor.
// Changes in monitor state will trigger attempts to connect to the new bridges.
type HubMonitor struct {
	logger     *zap.Logger
	conns      map[string]*grpc.ClientConn
	connsMutex sync.Mutex

	hub *bridge.Hub
}

// Alive is called when a bridge is reporting itself as alive
func (hm *HubMonitor) Alive(id string, connStr string) {
	hm.connsMutex.Lock()
	defer hm.connsMutex.Unlock()

	if _, exists := hm.conns[id]; exists {
		hm.logger.Debug("ignoring advertisement as id already registered",
			zap.String("id", id),
		)
		return
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

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	hub := bridge.NewHub(logger, &bridge.Bridge{
		Id: "todo",
	})

	hm := &HubMonitor{
		logger: logger,
		conns:  map[string]*grpc.ClientConn{},
		hub:    hub,
	}
	m := bridge.NewMonitor(logger, hm)

	logger.Info("monitoring for bridges")
	m.Run(context.Background())
}
