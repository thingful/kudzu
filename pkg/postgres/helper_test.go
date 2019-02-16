package postgres_test

import (
	"testing"

	kitlog "github.com/go-kit/kit/log"

	"github.com/thingful/kuzu/pkg/postgres"
)

// PrepareDB migrates down the DB fully - ready to be migrated up
func PrepareDB(t *testing.T, connStr string, logger kitlog.Logger) *postgres.DB {
	t.Helper()

	d, err := postgres.Open(connStr)
	if err != nil {
		t.Fatalf("Failed to open db connection: %v", err)
	}

	postgres.MigrateDownAll(d.DB, logger)

	err = d.Close()
	if err != nil {
		t.Fatalf("Failed to close db connection: %v", err)
	}

	db := postgres.NewDB(connStr, true)

	err = db.Start()
	if err != nil {
		t.Fatalf("Failed to start db service")
	}

	return db
}

// CleanDB truncates tables, and stops the given DB
func CleanDB(t *testing.T, db *postgres.DB) {
	err := postgres.Truncate(db.DB)
	if err != nil {
		t.Fatalf("Failed to truncate tables: %v", err)
	}

	err = db.Stop()
	if err != nil {
		t.Fatalf("Failed to stop db service")
	}
}
