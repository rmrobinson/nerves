package bridge

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // Blank import for sql drivers is "standard"
	"github.com/rmrobinson/nerves/services/domotics"
)

var (
	// ErrDatabaseNotSetup is returned if an operation is performed on a created but not configured database
	ErrDatabaseNotSetup = errors.New("database not setup")
	// ErrNotSupported is returned for operations made against the database that are not supported.
	ErrNotSupported = errors.New("not supported by database")
)

// Persister exposes an interface to allow the state of a bridge to be persisted.
// Not all bridge implementations allow for persisting all the relevant fields,
// so a bridge can use this in order to keep some bridge or device state across process restarts.
// This API inherits from the Hub Bridge interface as it needs to support the same operations.
// It has been extended to allow bridges and devices to be created, something a bridge normally does internally.
type Persister interface {
	domotics.SyncBridge

	CreateBridge(context.Context, *domotics.BridgeConfig) (string, error)
	CreateDevice(context.Context, *domotics.Device) error
}

// DB is a persistence layer for a bridge.
// Some bridges may not be able to persist everything we expect, and this layer allows for implementations
// to back certain operations by the bridge and persist the rest in a consistent way.
type DB struct {
	db       *sql.DB
	bridgeID string
}

// Open attempts to load the sqlite file at the specified path.
// Once Open succeeds the caller should be sure to invoke Close when it is finished with the handle.
func (db *DB) Open(fname string) error {
	sqldb, err := sql.Open("sqlite3", fname)
	if err != nil {
		return err
	}

	db.db = sqldb
	return db.setupDB()
}

// Close releases the handle to sqlite.
func (db *DB) Close() {
	if db.db != nil {
		db.db.Close()
	}
}

func (db *DB) setupDB() error {
	setupCmd := `CREATE TABLE IF NOT EXISTS devices(
		id TEXT NOT NULL PRIMARY KEY,
		addr TEXT NOT NULL,
		name TEXT,
		description TEXT,
		is_available BOOLEAN,
		is_binary BOOLEAN,
		is_on BOOLEAN,
		is_range BOOLEAN,
		range_value INT
		);
		CREATE TABLE IF NOT EXISTS bridges(
		id TEXT NOT NULL PRIMARY KEY,
		name TEXT
		);`

	_, err := db.db.Exec(setupCmd)
	if err != nil {
		return err
	}

	b, err := db.Bridge(context.Background())

	if err == ErrDatabaseNotSetup {
		return nil
	} else if err != nil {
		return err
	}

	db.bridgeID = b.Id
	return nil
}

// CreateBridge is called to load a bridge profile into the database.
// This will create an ID and return it.
func (db *DB) CreateBridge(ctx context.Context, config *domotics.BridgeConfig) (string, error) {
	// Populate the bridge
	cmd := `INSERT INTO bridges(
		id,
		name
		) VALUES
		(?, ?);`

	stmt, err := db.db.Prepare(cmd)
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	db.bridgeID = uuid.New().String()
	_, err = stmt.ExecContext(ctx, db.bridgeID, config.Name)
	return db.bridgeID, err
}

// Bridge retrieves the saved properties from the db and returns them.
// The data returned here should be merged with other data as it will not be complete.
// Currently only a small portion of the profile is persisted.
func (db *DB) Bridge(ctx context.Context) (*domotics.Bridge, error) {
	cmd := `SELECT id, name FROM bridges;`

	b := &domotics.Bridge{
		Config: &domotics.BridgeConfig{},
	}

	err := db.db.QueryRowContext(ctx, cmd).Scan(&b.Id, &b.Config.Name)
	if err == sql.ErrNoRows {
		return nil, ErrDatabaseNotSetup
	} else if err != nil {
		return nil, err
	}

	return b, nil
}

// SetBridgeConfig saves the supplied config to the database.
// Currently only the bridge name is saved.
func (db *DB) SetBridgeConfig(ctx context.Context, config *domotics.BridgeConfig) error {
	if len(db.bridgeID) < 1 {
		return ErrDatabaseNotSetup
	}

	cmd := `UPDATE bridges SET name=? WHERE id=?;`

	stmt, err := db.db.Prepare(cmd)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, config.Name, db.bridgeID)
	return err
}

// SetBridgeState is not supported.
func (db *DB) SetBridgeState(ctx context.Context, state *domotics.BridgeState) error {
	return ErrNotSupported
}

func (db *DB) devicesCustomQuery(ctx context.Context, query string) ([]*domotics.Device, error) {
	cmd := "SELECT id, addr, name, description, is_available, is_binary, is_on, is_range, range_value FROM devices"
	if len(query) > 0 {
		cmd += " " + query
	}
	cmd += ";"

	rows, err := db.db.QueryContext(ctx, cmd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*domotics.Device
	for rows.Next() {
		d := &domotics.Device{
			Config: &domotics.DeviceConfig{},
			State:  &domotics.DeviceState{},
		}

		isBinary := false
		bs := &domotics.DeviceState_BinaryState{}
		isRange := false
		rs := &domotics.DeviceState_RangeState{}

		err = rows.Scan(&d.Id, &d.Address, &d.Config.Name, &d.Config.Description, &d.IsActive, &isBinary, &bs.IsOn, &isRange, &rs.Value)
		if err != nil {
			return nil, err
		}

		if isBinary {
			d.State.Binary = bs
		}
		if isRange {
			d.State.Range = rs
		}

		devices = append(devices, d)
	}

	return devices, nil
}

// SearchForAvailableDevices is not supported.
func (db *DB) SearchForAvailableDevices(context.Context) error {
	return ErrNotSupported
}

// AvailableDevices returns any devices created that are currently available.
func (db *DB) AvailableDevices(ctx context.Context) ([]*domotics.Device, error) {
	return db.devicesCustomQuery(ctx, "WHERE is_available=1")
}

// Devices returns any devices created that are in use.
func (db *DB) Devices(ctx context.Context) ([]*domotics.Device, error) {
	return db.devicesCustomQuery(ctx, "WHERE is_available=0")
}

// Device returns the requested device, if present.
func (db *DB) Device(ctx context.Context, id string) (*domotics.Device, error) {
	ret, err := db.devicesCustomQuery(ctx, "WHERE id="+id)
	if err != nil {
		return nil, err
	} else if len(ret) < 1 {
		return nil, domotics.ErrDeviceNotRegistered
	}

	return ret[0], nil
}

// CreateDevice is used to seed a device into the database.
// Only some properties are currently persisted, including:
//  - ID
//  - Address
//  - IsActive (setting false will mark this device as an 'available' device
//  - Name (Config)
//  - Description (Config)
//  - Whether this is a binary device
//  - IsOn (Binary State)
//  - Whether this is a range device
//  - Value (Range State)
func (db *DB) CreateDevice(ctx context.Context, device *domotics.Device) error {
	cmd := `INSERT OR REPLACE INTO devices(
		id,
		addr,
		name,
		description,
		is_available,
		is_binary,
		is_on,
		is_range,
		range_value
		) VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?);`

	stmt, err := db.db.Prepare(cmd)
	if err != nil {
		return err
	}

	defer stmt.Close()

	if len(device.Id) < 1 {
		device.Id = uuid.New().String()
	}
	if device.Config == nil {
		device.Config = &domotics.DeviceConfig{}
	}
	if device.State == nil {
		device.State = &domotics.DeviceState{}
	}

	isBinary := false
	isOn := false
	isRange := false
	rangeValue := 0

	if device.State.Binary != nil {
		isBinary = true
		isOn = device.State.Binary.IsOn
	}
	if device.State.Range != nil {
		isRange = true
		rangeValue = int(device.State.Range.Value)
	}

	_, err = stmt.ExecContext(ctx, device.Id, device.Address, device.Config.Name,
		device.Config.Description, !device.IsActive,
		isBinary, isOn, isRange, rangeValue)
	return err
}

// SetDeviceConfig persists the available config options (name, description) to the database
func (db *DB) SetDeviceConfig(ctx context.Context, dev *domotics.Device, config *domotics.DeviceConfig) error {
	cmd := `UPDATE devices SET name=?,description=? WHERE id=?;`

	stmt, err := db.db.Prepare(cmd)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, config.Name, config.Description, dev.Id)
	return err

}

// SetDeviceState persists the available state options (isOn, range value) to the database.
func (db *DB) SetDeviceState(ctx context.Context, dev *domotics.Device, state *domotics.DeviceState) error {
	isBinary := false
	isOn := false
	isRange := false
	rangeValue := 0

	if state.Binary != nil {
		isBinary = true
		isOn = state.Binary.IsOn
	}
	if state.Range != nil {
		isRange = true
		rangeValue = int(state.Range.Value)
	}

	cmd := `UPDATE devices SET is_binary=?,is_on=?,is_range=?,range_value=? WHERE id=?;`

	stmt, err := db.db.Prepare(cmd)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, isBinary, isOn, isRange, rangeValue, dev.Id)
	return err
}

// AddDevice is used to move a device from 'available' to 'in use'
func (db *DB) AddDevice(ctx context.Context, id string) error {
	cmd := `UPDATE devices SET is_available=false WHERE id=?;`

	stmt, err := db.db.Prepare(cmd)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, id)
	return err
}

// DeleteDevice is used to move a device from 'in use' to 'available'
func (db *DB) DeleteDevice(ctx context.Context, id string) error {
	cmd := `UPDATE devices SET is_available=true, name="", description="", is_on=false, range_value=0 WHERE id=?;`

	stmt, err := db.db.Prepare(cmd)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, id)
	return err
}
