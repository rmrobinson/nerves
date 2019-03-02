package policy

import (
	"github.com/rmrobinson/nerves/services/domotics"
	"go.uber.org/zap"
)

// State represents the current state of the system this policy engine is monitoring.
// The engine will subscribe to updates from this state, and will execute operations against this state
// to change the state of the system.
type State struct {
	logger *zap.Logger

	bridgeState map[string]*domotics.Bridge
	deviceState map[string]*domotics.Device

	bridgeServiceClient domotics.BridgeServiceClient
	deviceServiceClient domotics.DeviceServiceClient
}

// NewState creates a new state entity to manage.
func NewState(logger *zap.Logger) *State {
	return &State{
		logger: logger,
		bridgeState: map[string]*domotics.Bridge{},
		deviceState: map[string]*domotics.Device{},
	}
}
