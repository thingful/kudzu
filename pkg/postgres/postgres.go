package postgres

import (
	kitlog "github.com/go-kit/kit/log"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // required by go sql driver
	"github.com/pkg/errors"
)

// Open is a simple helper function to return a new sqlx.DB instance or an error
func Open(connStr string) (*sqlx.DB, error) {
	return sqlx.Open("postgres", connStr)
}

// DB is our exported type that wraps a sqlx.DB instance
type DB struct {
	DB *sqlx.DB

	connStr string
	verbose bool
}

// NewDB returns a new DB instance which is not yet connected to the database
func NewDB(connStr string, verbose bool) *DB {
	return &DB{
		connStr: connStr,
		verbose: verbose,
	}
}

// Start attempts to connect to the configured database, running up migrations
// and returning any error.
func (d *DB) Start() error {
	db, err := Open(d.connStr)
	if err != nil {
		return errors.Wrap(err, "failed to open db connection")
	}

	d.DB = db

	log := kitlog.NewNopLogger()

	return MigrateUp(d.DB.DB, log)
}

// Stop stops the postgres connection pool
func (d *DB) Stop() error {
	return d.DB.Close()
}

// Truncate is a helper function for cleaning the database to help with tests
func Truncate(db *sqlx.DB) error {
	sql := `
	TRUNCATE data_sources CASCADE;
	TRUNCATE things CASCADE;
	TRUNCATE users CASCADE;
	TRUNCATE applications CASCADE;
	`

	_, err := db.Exec(sql)
	if err != nil {
		return errors.Wrap(err, "failed to truncate tables")
	}

	return nil
}
