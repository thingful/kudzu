package postgres

import (
	"context"

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
func (d *DB) SaveUser(ctx context.Context, user *User) (int64, error) {
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
	VALUES ((SELECT id FROM new_user), :auth_provider, :access_token, :refresh_token)
	RETURNING (SELECT id FROM new_user)`

	//mapArgs := map[string]interface{}{
	//	"uid":           user.UID,
	//	"auth_provider": user.Provider,
	//	"access_token":  user.AccessToken,
	//	"refresh_token": user.RefreshToken,
	//	"parrot_id":     user.ParrotID,
	//}

	sql, args, err := d.DB.BindNamed(sql, user)
	if err != nil {
		return 0, errors.Wrap(err, "failed to bind named parameters into query")
	}

	tx, err := d.DB.Beginx()
	if err != nil {
		return 0, errors.Wrap(err, "failed to open transaction")
	}

	var userID int64

	err = tx.Get(&userID, sql, args...)
	if err != nil {
		tx.Rollback()
		return 0, errors.Wrap(err, "failed to insert user")
	}

	return userID, tx.Commit()
}
