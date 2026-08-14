package schema

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"
)

const SchemaVersion = "0001"

//go:embed init.sql
var initialSchema string

func EnsureSchema(ctx context.Context, db *sql.DB) (string, error) {
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return "", fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA busy_timeout = 5000"); err != nil {
		return "", fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := db.ExecContext(ctx, initialSchema); err != nil {
		return "", fmt.Errorf("apply schema %s: %w", SchemaVersion, err)
	}
	if _, err := db.ExecContext(ctx,
		"INSERT OR IGNORE INTO schema_versions(version, applied_at) VALUES (?, ?)",
		SchemaVersion, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return "", fmt.Errorf("record schema %s: %w", SchemaVersion, err)
	}
	return SchemaVersion, nil
}
