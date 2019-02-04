package postgres

import (
	"database/sql"

	kitlog "github.com/go-kit/kit/log"
	"github.com/golang-migrate/migrate"
	"github.com/pkg/errors"
)

func MigrateUp(db *sql.DB, logger kitlog.Logger) error {
	logger.Log("msg", "migrating DB up")

	_, err := getMigrator(db, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create migrator")
	}

	return nil
}

func getMigrator(db *sql.DB, logger kitlog.Logger) (*migrate.Migrate, error) {
	return nil, nil
}
