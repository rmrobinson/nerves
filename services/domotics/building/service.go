package building

import (
	"context"
	"sync"

	"github.com/rmrobinson/nerves/lib/stream"

	"github.com/google/uuid"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StatePersister allows for the persistence and retrieval of State.
// The implementation of persister may choose not to persist point-in-time data (i.e. whether a light is on or off)
// however it should persist all configuration-related data (which devices are in which rooms).
type StatePersister interface {
	Persist(context.Context, *State) error
	Load(context.Context) (*State, error)
}

type stateChange int

const (
	stateChangeUpdated stateChange = iota
	stateChangeAdded
	stateChangeRemoved
)

type stateWatcher interface {
	updated(stateChange, interface{})
}

// Service contains the active collection of buildings, rooms, floors and their associated bridges and devices.
// In a given instance of a domotics process it is expected there is only one active 'Service'.
// How does the service work?
// It loads its State from the supplied StatePersister
// For each bridge it has configured, it establishes a connection to this bridge and retrieves the current state.
// For each building it has configured, it builds the mappings and begins sending updates from the linked bridges and/or devices (once the bridge connection has been established).
// The service then accepts incoming requests to query and manipulate its state (and persists as appropriate).
// The service will stream updates from its configured bridges when received to registered subscribers.
// NOTE: clients need to perform write actions against the devices directly. The device and bridge services will be registered on the same listener but this service
// assumes that all updates coming from the devices/bridges are initiated independently and doesn't deal with partial write situations to simplify callflows.
// TODO: decide on how state syncing will happen.
type Service struct {
	logger *zap.Logger
	m      sync.Mutex

	state     *State
	persister StatePersister
	watcher   stateWatcher

	bridge *bridge.Bridge
	// TODO: put this into the watcher potentially?
	bridgeUpdatesSource *stream.Source
}

// NewService creates a new Service
func NewService(logger *zap.Logger, persister StatePersister) *Service {
	return &Service{
		logger:    logger,
		persister: persister,
		state: &State{
			buildings: map[string]*Building{},
			floors:    map[string]*Floor{},
			rooms:     map[string]*Room{},
		},

		bridge: &bridge.Bridge{
			Id:           "test-id",
			ModelId:      "falnet-domotics1",
			Manufacturer: "faltung systems",
			Config: &bridge.BridgeConfig{
				Name:        "domotics bridge",
				Description: "a virtual bridge which aggregates all linked bridges in the house",
			},
		},
		bridgeUpdatesSource: stream.NewSource(logger),
	}
}

var (
	// ErrBuildingNotFound is returned if the requested building cannot be found
	ErrBuildingNotFound = status.New(codes.NotFound, "building not found")
	// ErrFloorNotFound is returned if the requested floor cannot be found
	ErrFloorNotFound = status.New(codes.NotFound, "floor not found")
)

// Setup initializes the state from persistence and gets the service ready to listen for updates.
func (s *Service) Setup(ctx context.Context) error {
	if s.persister == nil {
		panic("persister missing")
	}

	state, err := s.persister.Load(ctx)
	if err != nil {
		return err
	}

	s.state = state
	return nil
}

// Run listens for updates from registered endpoints and propogates them to subscribed listeners.
func (s *Service) Run() {

}

// AddBuilding creates a new building and persists it.
func (s *Service) AddBuilding(ctx context.Context, b *Building) error {
	s.m.Lock()
	defer s.m.Unlock()

	b.Id = uuid.New().String()
	s.state.buildings[b.Id] = b

	err := s.persister.Persist(ctx, s.state)
	if err != nil {
		return err
	}

	if s.watcher != nil {
		s.watcher.updated(stateChangeAdded, b)
	}
	return nil
}

// GetBuilding retrieves a building from the state store.
func (s *Service) GetBuilding(ctx context.Context, bid string) (*Building, error) {
	if b, ok := s.state.buildings[bid]; ok {
		return b, nil
	}
	return nil, ErrBuildingNotFound.Err()
}

// GetBuildings retrieves all buildings from the state store.
func (s *Service) GetBuildings(ctx context.Context) ([]*Building, error) {
	var ret []*Building
	for _, b := range s.state.buildings {
		ret = append(ret, b)
	}

	return ret, nil
}

// AddFloor creates a new floor and adds it to its building, then persists it.
func (s *Service) AddFloor(ctx context.Context, f *Floor, bid string) error {
	s.m.Lock()
	defer s.m.Unlock()

	var b *Building
	var ok bool
	if b, ok = s.state.buildings[bid]; !ok {
		return ErrBuildingNotFound.Err()
	}

	f.Id = uuid.New().String()
	s.state.floors[f.Id] = f

	b.Floors = append(b.Floors, f)

	err := s.persister.Persist(ctx, s.state)
	if err != nil {
		return err
	}

	if s.watcher != nil {
		s.watcher.updated(stateChangeAdded, f)
		s.watcher.updated(stateChangeUpdated, b)
	}
	return nil
}

// GetFloor retrieves a floor from the state store.
func (s *Service) GetFloor(ctx context.Context, fid string) (*Floor, error) {
	if f, ok := s.state.floors[fid]; ok {
		return f, nil
	}
	return nil, ErrFloorNotFound.Err()
}

// GetFloors retrieves all floors in a building from the state store.
func (s *Service) GetFloors(ctx context.Context, bid string) ([]*Floor, error) {
	if b, ok := s.state.buildings[bid]; ok {
		return b.Floors, nil
	}

	return nil, ErrBuildingNotFound.Err()
}

// AddRoom creates a new room and adds it to its floor, then persists it.
func (s *Service) AddRoom(ctx context.Context, r *Room, fid string) error {
	s.m.Lock()
	defer s.m.Unlock()

	var f *Floor
	var ok bool
	if f, ok = s.state.floors[fid]; !ok {
		return ErrFloorNotFound.Err()
	}

	r.Id = uuid.New().String()
	s.state.rooms[r.Id] = r

	f.Rooms = append(f.Rooms, r)

	err := s.persister.Persist(ctx, s.state)
	if err != nil {
		return err
	}

	if s.watcher != nil {
		s.watcher.updated(stateChangeAdded, r)
		s.watcher.updated(stateChangeUpdated, f)
	}
	return nil
}

// GetDevices returns all the devices this service is aware of
func (s *Service) GetDevices(ctx context.Context) ([]*bridge.Device, error) {
	return nil, nil
}

// GetDevice returns the specified device, if found
func (s *Service) GetDevice(ctx context.Context, deviceID string) (*bridge.Device, error) {
	return nil, nil
}
