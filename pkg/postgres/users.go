package postgres

import (
	"context"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/thingful/kuzu/pkg/logger"
)

type User struct {
	ID           int64  `db:"id"`
	UID          string `db:"uid"`
	ParrotID     string `db:"parrot_id"`
	AccessToken  string `db:"access_token"`
	RefreshToken string `db:"refresh_token"`
	Provider     string `db:"auth_provider"`
}

// SaveUser attempts to save a user into the database along with an associated
// identity. The concept of separate identities was originally intended to
// support multiple providers, but currently we only read from parrot.
func (d *DB) SaveUser(ctx context.Context, user *User) error {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log("msg", "saving user", "uid", user.UID, "parrotID", user.ParrotID)
	}

	sql := `WITH new_user AS (
		INSERT INTO users (uid, parrot_id)
		VALUES (:uid, :parrot_id)
		RETURNING id
	)
	INSERT INTO identities (owner_id, auth_provider, access_token, refresh_token)
	VALUES ((SELECT id FROM new_user), :auth_provider, :access_token, :refresh_token)`

	sql, args, err := d.DB.BindNamed(sql, user)
	if err != nil {
		return errors.Wrap(err, "failed to bind named parameters into query")
	}

	tx, err := d.DB.Beginx()
	if err != nil {
		return errors.Wrap(err, "failed to open transaction")
	}

	_, err = tx.Exec(sql, args...)
	if err != nil {
		tx.Rollback()
		if pqerr, ok := err.(*pq.Error); ok {
			if pqerr.Code == ConstraintError || pqerr.Code == UniqueViolationError {
				return errors.Wrap(ClientError, "failed to insert user")
			}
		}
		return errors.Wrap(err, "failed to insert user")
	}

	return tx.Commit()
}
