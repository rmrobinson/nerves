package policy

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/weather"
	crontab "github.com/robfig/cron"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	// the amount of time a cron entry will remain active after triggering
	cronEntryActiveDuration = time.Second * 5
	// the amount of time a timer entry will remain active after triggering
	timerEntryActiveDuration = time.Second * 5
)

var (
	// ErrInvalidCondition is returned if a condition is supplied that doesn't meet the requirements of the method.
	// For example, providing a condition without a cron field to the addCronEntry rule would yield this error.
	ErrInvalidCondition = errors.New("invalid condition supplied")
	// ErrInvalidAction is returned if an action is supplied that doesn't meet the requirements of the method.
	// For example, providing a timer action with an empty body would yield this error.
	ErrInvalidAction = errors.New("invalid action supplied")
)

type cronEntry struct {
	condition *Condition
	cron      *crontab.Cron
	triggered bool
}

type timerEntry struct {
	id        string
	timer     *time.Timer
	active    bool
	triggered bool
}

// State represents the current state of the system this policy engine is monitoring.
// The engine will subscribe to updates from this state, and will execute operations against this state
// to change the state of the system.
type State struct {
	logger *zap.Logger

	refresh chan<- bool

	weatherState map[string]*weather.WeatherReport

	bridgeState map[string]*domotics.Bridge
	deviceState map[string]*domotics.Device
	deviceLock  sync.Mutex

	cronsByCond map[*Condition]*cronEntry

	timersByID map[string]*timerEntry
	timerLock  sync.Mutex

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
		weatherState: map[string]*weather.WeatherReport{},
		cronsByCond:  map[*Condition]*cronEntry{},
		timersByID:   map[string]*timerEntry{},
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

	entry := &cronEntry{
		cron:      crontab.NewWithLocation(loc),
		condition: c,
		triggered: false,
	}
	err = entry.cron.AddFunc(c.Cron.Entry, func() {
		entry := s.cronsByCond[c]

		s.logger.Debug("timer triggered",
			zap.String("name", entry.condition.Name),
			zap.String("rule", entry.condition.Cron.Entry),
		)

		entry.triggered = true
		s.refresh <- true

		go func(entry *cronEntry) {
			// We need to 'turn off' the entry at some point in the future.
			// We don't know when the execution triggered by the refresh action will be true
			// so we use the entry active duration to control this.
			time.Sleep(cronEntryActiveDuration)
			entry.triggered = false
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

func (s *State) activateTimer(ta *TimerAction) error {
	s.timerLock.Lock()
	defer s.timerLock.Unlock()

	var te *timerEntry
	var ok bool

	if ta.Timer == nil {
		return ErrInvalidAction
	} else if te, ok = s.timersByID[ta.Id]; !ok {
		te = &timerEntry{}
	}

	// We don't want to have >1 ticker running
	// This just means the triggering action already started
	if te.active {
		return nil
	}
	te.timer = time.NewTimer(time.Duration(ta.Timer.IntervalMs) * time.Millisecond)
	s.timersByID[ta.Id] = te

	go func(id string) {
		<- te.timer.C

		s.timerLock.Lock()
		defer s.timerLock.Unlock()

		entry := s.timersByID[id]

		s.logger.Debug("timer triggered",
			zap.String("id", id),
		)

		entry.triggered = true
		s.refresh <- true

		go func(entry *timerEntry) {
			// We need to 'turn off' the entry at some point in the future.
			// We don't know when the execution triggered by the refresh action will be true
			// so we use the entry active duration to control this.
			time.Sleep(timerEntryActiveDuration)
			entry.active = false
			entry.triggered = false
			s.refresh <- true
		}(entry)
	}(ta.Id)

	return nil
}
