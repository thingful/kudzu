package postgres

import (
	"time"

	"github.com/pkg/errors"
)

type User struct {
	ID        int64     `db:"id" json:"-"`
	UID       string    `db:"uid" json:"Uid"`
	CreatedAt time.Time `db:"created_at" json:"-"`
}

// SaveUser attempts to save a user into the database
func (d *DB) SaveUser(uid string) error {
	sql := `INSERT INTO users (uid) VALUES (:uid)`
	mapArgs := map[string]interface{}{
		"uid": uid,
	}

	sql, args, err := d.DB.BindNamed(sql, mapArgs)
	if err != nil {
		return errors.Wrap(err, "failed to bind named parameters into query")
	}

	tx, err := d.DB.Beginx()
	if err != nil {
		return errors.Wrap(err, "failed to open transaction")
	}

	_, err = tx.Exec(sql, args)
	if err != nil {
		return tx.Rollback()
	}

	return tx.Commit()
}
