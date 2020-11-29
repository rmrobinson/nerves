package building

import (
	"context"
	"database/sql"

	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

// InMemoryPersister satisfies the requirements of the 'StatePersister' interface in memory.
type InMemoryPersister struct {
	s *State
}

// NewInMemoryPersister creates a new instance of an in-memory persister
func NewInMemoryPersister() *InMemoryPersister {
	return &InMemoryPersister{
		s: &State{
			buildings: map[string]*Building{},
			floors:    map[string]*Floor{},
			rooms:     map[string]*Room{},
		},
	}
}

// Persist takes the supplied state and creates a copy if it in memory for retrieval.
func (imp *InMemoryPersister) Persist(ctx context.Context, s *State) error {
	imp.s = s.Dup()

	spew.Dump(imp.s)
	return nil
}

// Load retrieves whatever is currently stored and returns it.
func (imp *InMemoryPersister) Load(ctx context.Context) (*State, error) {
	return imp.s.Dup(), nil
}

// SQLPersister satisfies the requirements of the 'StatePersister' interface in a SQL DB
type SQLPersister struct {
	logger *zap.Logger
	db     *sql.DB
}

const (
	selectFloorRoomsQuery     = `SELECT id, name, description FROM room WHERE floor_id=?;`
	upsertRoomQuery           = `INSERT OR REPLACE INTO room(id, name, description, floor_id) VALUES (?, ?, ?, ?)`
	selectBuildingFloorsQuery = `SELECT id, name, description, level FROM floor WHERE building_id=?;`
	upsertFloorQuery          = `INSERT OR REPLACE INTO floor(id, name, description, level, building_id) VALUES (?, ?, ?, ?, ?)`
	selectBuildingsQuery      = `SELECT id, name, description, address FROM building;`
	upsertBuildingQuery       = `INSERT OR REPLACE INTO building(id, name, description, address) VALUES (?, ?, ?, ?)`
)

// NewSQLPersister creates a new persister backed by a SQL DB
func NewSQLPersister(logger *zap.Logger, db *sql.DB) *SQLPersister {
	return &SQLPersister{
		logger: logger,
		db:     db,
	}
}

// Persist takes the supplied state and saves it to a database.
func (p *SQLPersister) Persist(ctx context.Context, s *State) error {
	for _, b := range s.buildings {
		err := p.persistBuilding(ctx, b)
		if err != nil {
			p.logger.Info("error peristing building, continuing",
				zap.String("building_id", b.Id),
				zap.Error(err),
			)
		}
	}

	return nil
}

func (p *SQLPersister) persistBuilding(ctx context.Context, b *Building) error {
	buildingStmt, err := p.db.PrepareContext(ctx, upsertBuildingQuery)
	if err != nil {
		return err
	}
	defer buildingStmt.Close()

	_, err = buildingStmt.ExecContext(ctx, b.Id, b.Name, b.Description, b.Address)
	if err != nil {
		p.logger.Info("unable to save building",
			zap.String("building_id", b.Id),
			zap.Error(err),
		)
		return err
	}

	for _, f := range b.Floors {
		err = p.persistFloor(ctx, f, b.Id)
		if err != nil {
			p.logger.Info("unable to save floor, continuing",
				zap.String("building_id", b.Id),
				zap.String("floor_id", f.Id),
				zap.Error(err),
			)
		}
	}

	return nil
}

func (p *SQLPersister) persistFloor(ctx context.Context, f *Floor, buildingID string) error {
	floorStmt, err := p.db.PrepareContext(ctx, upsertFloorQuery)
	if err != nil {
		return err
	}
	defer floorStmt.Close()

	roomStmt, err := p.db.PrepareContext(ctx, upsertRoomQuery)
	if err != nil {
		return err
	}
	defer roomStmt.Close()

	_, err = floorStmt.ExecContext(ctx, f.Id, f.Name, f.Description, f.Level, buildingID)
	if err != nil {
		p.logger.Info("unable to save floor",
			zap.String("floor_id", f.Id),
			zap.Error(err),
		)
		return err
	}

	for _, r := range f.Rooms {
		_, err = roomStmt.ExecContext(ctx, r.Id, r.Name, r.Description, f.Id)
		if err != nil {
			p.logger.Info("unable to save room, continuing",
				zap.String("floor_id", f.Id),
				zap.String("room_id", r.Id),
				zap.Error(err),
			)
		}
	}

	return nil
}

// Load retrieves whatever is currently stored and returns it.
func (p *SQLPersister) Load(ctx context.Context) (*State, error) {
	buildings, err := p.loadBuildings(ctx)
	if err != nil {
		return nil, err
	}

	s := &State{
		buildings: map[string]*Building{},
		floors:    map[string]*Floor{},
		rooms:     map[string]*Room{},
	}

	for _, b := range buildings {
		s.buildings[b.Id] = b

		for _, f := range b.Floors {
			s.floors[f.Id] = f

			for _, r := range f.Rooms {
				s.rooms[r.Id] = r
			}
		}
	}

	return s, nil
}

func (p *SQLPersister) loadBuildings(ctx context.Context) ([]*Building, error) {
	// Get the buildings
	rows, err := p.db.QueryContext(ctx, selectBuildingsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buildings []*Building
	for rows.Next() {
		b := &Building{}
		err = rows.Scan(&b.Id, &b.Name, &b.Description, &b.Address)
		if err != nil {
			return nil, err
		}
		buildings = append(buildings, b)
	}

	// Now we get the floors in the building
	for _, building := range buildings {
		floors, err := p.loadBuildingFloors(ctx, building.Id)
		if err != nil {
			p.logger.Info("unable to load floors for building",
				zap.String("building_id", building.Id),
				zap.Error(err),
			)
			continue
		}

		building.Floors = floors
	}

	return buildings, nil
}

func (p *SQLPersister) loadBuildingFloors(ctx context.Context, buildingID string) ([]*Floor, error) {
	// Get the floors for the building
	rows, err := p.db.QueryContext(ctx, selectBuildingFloorsQuery, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var floors []*Floor
	for rows.Next() {
		f := &Floor{}
		err = rows.Scan(&f.Id, &f.Name, &f.Description, &f.Level)
		if err != nil {
			return nil, err
		}
		floors = append(floors, f)
	}

	// Now we get the rooms on the floor
	for _, floor := range floors {
		rooms, err := p.loadFloorRooms(ctx, floor.Id)
		if err != nil {
			p.logger.Info("unable to load rooms for floor",
				zap.String("building_id", buildingID),
				zap.String("floor_id", floor.Id),
				zap.Error(err),
			)
			continue
		}

		floor.Rooms = rooms
	}

	// TODO: get devices on the floor

	return floors, nil
}

func (p *SQLPersister) loadFloorRooms(ctx context.Context, floorID string) ([]*Room, error) {
	// Get the rooms on the floor
	rows, err := p.db.QueryContext(ctx, selectFloorRoomsQuery, floorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*Room
	for rows.Next() {
		r := &Room{}
		err = rows.Scan(&r.Id, &r.Name, &r.Description)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, r)
	}

	// TODO: Get the devices in each room

	return rooms, nil
}
