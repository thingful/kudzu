package postgres

import (
	"context"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/thingful/kudzu/pkg/logger"
)

// User is a struct used to pass data into the SaveUser function - captures all
// the data we receive when creating a new user record.
type User struct {
	ID           int64  `db:"id"`
	UID          string `db:"uid"`
	ParrotID     string `db:"parrot_id"`
	AccessToken  string `db:"access_token"`
	RefreshToken string `db:"refresh_token"`
	Provider     string `db:"auth_provider"`
}

// UserStat is a type used to export metrics from the DB to prometheus
type UserStat struct {
	Count    float64 `db:"count"`
	Provider string  `db:"auth_provider"`
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
		if pqerr, ok := err.(*pq.Error); ok {
			if pqerr.Code == ConstraintError || pqerr.Code == UniqueViolationError {
				return 0, errors.Wrap(ClientError, "failed to insert user")
			}
		}
		return 0, errors.Wrap(err, "failed to insert user")
	}

	return userID, tx.Commit()
}

// DeleteUser attempts to delete the user identified by the supplied uid
func (d *DB) DeleteUser(ctx context.Context, uid string) error {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log("msg", "deleting user", "uid", uid)
	}

	sqlQuery := `DELETE FROM users WHERE uid = $1`

	tx, err := d.DB.Beginx()
	if err != nil {
		return errors.Wrap(err, "failed to open transaction")
	}

	_, err = tx.Exec(sqlQuery, uid)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "faield to delete user")
	}

	return tx.Commit()
}

// CountUsers returns a count of registered users partitioned by provider
// (currently just Parrot)
func (d *DB) CountUsers(ctx context.Context) ([]UserStat, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log("msg", "counting users")
	}

	sql := `SELECT COUNT(u.*), i.auth_provider
		FROM users u
		JOIN identities i ON i.owner_Id = u.id
		GROUP BY i.auth_provider`

	rows, err := d.DB.Queryx(sql)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count users")
	}

	stats := []UserStat{}

	for rows.Next() {
		var stat UserStat
		err = rows.StructScan(&stat)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row into UserStat struct")
		}
		stats = append(stats, stat)
	}

	return stats, nil
}
