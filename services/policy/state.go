package policy

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rmrobinson/nerves/services/domotics"
	crontab "github.com/robfig/cron"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	// the amount of time a cron entry will remain active after triggering
	cronEntryActiveDuration = time.Second * 5
)

var (
	// ErrInvalidCondition is returned if a condition is supplied that doesn't meet the requirements of the method.
	// For example, providing a condition without a cron field to the addCronEntry rule would yield this error.
	ErrInvalidCondition = errors.New("invalid condition supplied")
)

type cronEntry struct {
	id        string
	cron      *crontab.Cron
	condition *Condition
	active    bool
}

// State represents the current state of the system this policy engine is monitoring.
// The engine will subscribe to updates from this state, and will execute operations against this state
// to change the state of the system.
type State struct {
	logger *zap.Logger

	refresh chan<- bool

	bridgeState map[string]*domotics.Bridge
	deviceState map[string]*domotics.Device
	deviceLock  sync.Mutex

	cronsByCond map[*Condition]*cronEntry

	bridgeClient domotics.BridgeServiceClient
	deviceClient domotics.DeviceServiceClient
}

// NewState creates a new state entity to manage.
func NewState(logger *zap.Logger, conn *grpc.ClientConn) *State {
	return &State{
		logger:       logger,
		bridgeClient: domotics.NewBridgeServiceClient(conn),
		deviceClient: domotics.NewDeviceServiceClient(conn),
		bridgeState:  map[string]*domotics.Bridge{},
		deviceState:  map[string]*domotics.Device{},
		cronsByCond:  map[*Condition]*cronEntry{},
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

func (s *State) addCronEntry(c *Condition) error {
	var loc *time.Location
	var err error

	if c.Cron == nil {
		return ErrInvalidCondition
	}
	if len(c.Cron.Tz) > 0 {
		loc, err = time.LoadLocation(c.Cron.Tz)
		if err != nil {
			return err
		}
	} else {
		loc = time.Local
	}

	id := uuid.New().String()
	entry := &cronEntry{
		id:        id,
		cron:      crontab.NewWithLocation(loc),
		condition: c,
		active:    false,
	}
	err = entry.cron.AddFunc(c.Cron.Entry, func() {
		entry := s.cronsByCond[c]

		s.logger.Debug("timer triggered",
			zap.String("name", entry.condition.Name),
			zap.String("rule", entry.condition.Cron.Entry),
		)

		entry.active = true
		s.refresh <- true

		go func(entry *cronEntry) {
			// We need to 'turn off' the entry at some point in the future.
			// We don't know when the execution triggered by the refresh action will be true
			// so we use the entry active duration to control this.
			time.Sleep(cronEntryActiveDuration)
			entry.active = false
			s.refresh <- true
		}(entry)
	})
	if err != nil {
		return err
	}

	s.logger.Debug("adding cron entry",
		zap.String("name", c.Name),
		zap.String("rule", c.Cron.Entry),
	)
	s.cronsByCond[c] = entry
	entry.cron.Start()

	return nil
}
