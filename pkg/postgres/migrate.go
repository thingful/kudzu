package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/pkg/errors"
	"github.com/serenize/snaker"

	"github.com/thingful/kudzu/pkg/migrations"
)

// MigrateUp attempts to run all up migrations against the database
func MigrateUp(db *sql.DB, logger kitlog.Logger) error {
	logger.Log("msg", "migrating postgres up")

	m, err := getMigrator(db, logger)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != migrate.ErrNoChange {
		return errors.Wrap(err, "failed to run up migrations")
	}

	return nil
}

// MigrateDown attempts to run the specified number of down migrations against
// the database.
func MigrateDown(db *sql.DB, steps int, logger kitlog.Logger) error {
	logger.Log("msg", "migrating postgres down", "steps", steps)

	m, err := getMigrator(db, logger)
	if err != nil {
		return err
	}

	return m.Steps(-steps)
}

// MigrateDownAll attempts to run all down migrations.
func MigrateDownAll(db *sql.DB, logger kitlog.Logger) error {
	logger.Log("msg", "migrating postgres down all")

	m, err := getMigrator(db, logger)
	if err != nil {
		return err
	}

	return m.Down()
}

// NewMigration creates a new pair of matched migration files.
func NewMigration(dirName, migrationName string, logger kitlog.Logger) error {
	if migrationName == "" {
		return errors.New("Must specify a name when creating a migration")
	}

	re := regexp.MustCompile(`\A[a-zA-Z]+\z`)
	if !re.MatchString(migrationName) {
		return errors.New("Name must be a single CamelCased string with no numbers or special characters")
	}

	migrationID := time.Now().Format("20060102150405") + "_" + snaker.CamelToSnake(migrationName)
	upFilename := fmt.Sprintf("%s.up.sql", migrationID)
	downFilename := fmt.Sprintf("%s.down.sql", migrationID)

	logger.Log("upFile", upFilename, "downFile", downFilename, "migrationDir", dirName, "msg", "creating migration files")

	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		return errors.Wrap(err, "failed to make directory for migrations")
	}

	upFile, err := os.Create(filepath.Join(dirName, upFilename))
	defer upFile.Close()
	if err != nil {
		return errors.Wrap(err, "failed to make up migration file")
	}

	downFile, err := os.Create(filepath.Join(dirName, downFilename))
	defer downFile.Close()
	if err != nil {
		return errors.Wrap(err, "failed to make down migration file")
	}

	return nil
}

func getMigrator(db *sql.DB, logger kitlog.Logger) (*migrate.Migrate, error) {
	dbDriver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: "kudzu_schema_migrations",
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to get postgres db driver for migrations")
	}

	source := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	)

	sourceDriver, err := bindata.WithInstance(source)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create source driver")
	}

	migrator, err := migrate.NewWithInstance(
		"go-bindata",
		sourceDriver,
		"postgres",
		dbDriver,
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create migrator")
	}

	migrator.Log = newLogAdapter(logger, true)

	return migrator, nil
}

// newLogAdapter is a helper constructor to create an instance of our log
// adapter.
func newLogAdapter(logger kitlog.Logger, verbose bool) migrate.Logger {
	return &logAdapter{
		logger:  logger,
		verbose: verbose,
	}
}

// logAdapter wraps our kitlog.Logger and exposes an API that the migrator
// library can use
type logAdapter struct {
	logger  kitlog.Logger
	verbose bool
}

// Printf is the required method to be exposed. The semantics are the same as
// the `fmt.Printf ` family. Here we simply pass the output via a `msg` key to
// the wrapped logger.
func (l *logAdapter) Printf(format string, v ...interface{}) {
	l.logger.Log("msg", fmt.Sprintf(format, v...))
}

// Verbose returns true when verbose logging is enabled
func (l *logAdapter) Verbose() bool {
	return l.verbose
}
