package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func OpenSQLite(
	ctx context.Context,
	path string,
) (*sql.DB, error) {
	dsn := "file:" + path +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening SQLite database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to SQLite database: %w", err)
	}

	return db, nil
}
