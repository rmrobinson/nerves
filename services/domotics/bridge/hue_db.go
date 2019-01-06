package bridge

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// We require the blank import to properly link against sqlite
	_ "github.com/mattn/go-sqlite3"
)

// HuePersister is an interface to persisting bridge profiles.
type HuePersister interface {
	Profile(ctx context.Context, bridgeID string) (string, error)
	SaveProfile(ctx context.Context, bridgeID string, username string) error
	Close() error
}

type hueProfile struct {
	ID             string
	Username       string
	LastModifiedAt time.Time
}

// HueDB is an implementation of the Hue persister.
type HueDB struct {
	db *sql.DB
}

// Open takes the supplied file path and attempts to open a SQLite DB at that place;
// and it will create a DB if one doesn't exist.
func (s *HueDB) Open(fname string) error {
	db, err := sql.Open("sqlite3", fname)
	if err != nil {
		return fmt.Errorf("unable to open config db: %s", err)
	}

	s.db = db
	return s.setupDb()
}

// Close cleans up the handle to the DB.
func (s *HueDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *HueDB) setupDb() error {
	cmd := `CREATE TABLE IF NOT EXISTS hue_profiles(
		id TEXT NOT NULL PRIMARY KEY,
		username TEXT,
		lastModifiedTime DATETIME
		);`

	_, err := s.db.Exec(cmd)
	return err
}

// Profile retrieves the username for the specified bridge ID.
func (s *HueDB) Profile(ctx context.Context, bridgeID string) (string, error) {
	cmd := `SELECT id, username, lastModifiedTime FROM hue_profiles
		WHERE id=?;`
	p := hueProfile{}

	err := s.db.QueryRowContext(ctx, cmd, bridgeID).Scan(&p.ID, &p.Username, &p.LastModifiedAt)

	switch {
	case err == sql.ErrNoRows:
		return "", fmt.Errorf("id not present: %s", bridgeID)
	case err != nil:
		return "", err
	default:
		return p.Username, nil
	}
}

// SaveProfile saves the specified username on the specified bridge ID.
func (s *HueDB) SaveProfile(ctx context.Context, bridgeID string, username string) error {
	cmd := `INSERT OR REPLACE INTO hue_profiles(
		id,
		username,
		lastModifiedTime
		) VALUES
		(?, ?, CURRENT_TIMESTAMP);`

	stmt, err := s.db.Prepare(cmd)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, bridgeID, username)

	return err
}
