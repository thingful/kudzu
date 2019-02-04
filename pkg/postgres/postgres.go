package postgres

import (
	kitlog "github.com/go-kit/kit/log"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // required by go sql driver
)

// Open is a simple helper function to return a new sqlx.DB instance or an error
func Open(connStr string) (*sqlx.DB, error) {
	return sqlx.Open("postgres", connStr)
}

// DB is our exported type that wraps a sqlx.DB instance
type DB struct {
	DB *sqlx.DB

	logger  kitlog.Logger
	connStr string
}

// NewDB returns a new DB instance which is not yet connected to the database
func NewDB(connStr string, logger kitlog.Logger) *DB {
	logger = kitlog.With(logger, "module", "postgres")

	logger.Log("msg", "configuring postgres service")

	return &DB{
		connStr: connStr,
		logger:  logger,
	}
}

// Start attempts to connect to the configured database, returning any error.
func (d *DB) Start() error {
	d.logger.Log("msg", "starting postgres service")

	db, err := Open(d.connStr)
	if err != nil {
		return err
	}

	d.DB = db

	return nil
}
