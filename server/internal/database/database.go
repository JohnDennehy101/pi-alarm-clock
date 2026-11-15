package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
)

type Database struct {
	DB              *sql.DB
	AlarmRepository *AlarmRepository
}

func New(ctx context.Context, dataDir string) (*Database, error) {
	db, err := OpenSQLite(
		ctx,
		filepath.Join(dataDir, "alarms.db"),
	)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &Database{
		DB:              db,
		AlarmRepository: NewAlarmRepository(db),
	}, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}
