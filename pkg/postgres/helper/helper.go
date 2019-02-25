package helper

import (
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/thingful/kudzu/pkg/postgres"
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
	err := Truncate(t, db)
	if err != nil {
		t.Fatalf("Failed to truncate tables: %v", err)
	}

	err = db.Stop()
	if err != nil {
		t.Fatalf("Failed to stop db service")
	}
}

// Truncate is a helper function for cleaning the database to help with tests
func Truncate(t *testing.T, db *postgres.DB) error {
	t.Helper()
	sql := `
	TRUNCATE data_sources CASCADE;
	TRUNCATE things CASCADE;
	TRUNCATE users CASCADE;
	TRUNCATE applications CASCADE;
	`

	_, err := db.DB.Exec(sql)
	if err != nil {
		return errors.Wrap(err, "failed to truncate tables")
	}

	return nil
}
