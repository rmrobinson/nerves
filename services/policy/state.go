package policy

import (
	"context"
	"sync"

	"github.com/rmrobinson/nerves/services/domotics"
	"go.uber.org/zap"
)

// State represents the current state of the system this policy engine is monitoring.
// The engine will subscribe to updates from this state, and will execute operations against this state
// to change the state of the system.
type State struct {
	logger *zap.Logger

	refresh chan<- bool

	bridgeState map[string]*domotics.Bridge
	deviceState map[string]*domotics.Device
	deviceLock  sync.Mutex

	bridgeClient domotics.BridgeServiceClient
	deviceClient domotics.DeviceServiceClient
}

// NewState creates a new state entity to manage.
func NewState(logger *zap.Logger, refresh chan<- bool) *State {
	return &State{
		logger:      logger,
		refresh:     refresh,
		bridgeState: map[string]*domotics.Bridge{},
		deviceState: map[string]*domotics.Device{},
	}
}

// Monitor is used to track changes to devices
func (s *State) Monitor(ctx context.Context) {
	stream, err := s.deviceClient.StreamDeviceUpdates(ctx, &domotics.StreamDeviceUpdatesRequest{})
	if err != nil {
		s.logger.Info("error creating device update stream",
			zap.Error(err),
		)
		return
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			s.logger.Info("error receiving update",
				zap.Error(err),
			)
			return
		}

		s.handleDeviceUpdate(update)
	}
}

func (s *State) handleDeviceUpdate(update *domotics.DeviceUpdate) {
	s.deviceLock.Lock()
	defer s.deviceLock.Unlock()

	s.deviceState[update.Device.Id] = update.Device
	s.refresh <- true
}
