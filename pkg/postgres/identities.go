package postgres

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/logger"
)

// Identity is a type containing both access token and user id we return when
// looking for the next token to index
type Identity struct {
	OwnerID     int64  `db:"owner_id"`
	AccessToken string `db:"access_token"`
}

// NextIdentity returns the next identity we want to attempt to index
// locations for. We load one that has either never been indexed before (i.e.
// with nil indexed_at), or the oldest one that hasn't already been indexed
// today.
func (d *DB) NextIdentity(ctx context.Context) (*Identity, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log("msg", "retrieving next access token")
	}

	query := `WITH next_identity AS (
		SELECT id FROM identities
		WHERE indexed_at IS NULL OR indexed_at < NOW() - interval '24 hours'
		ORDER BY indexed_at DESC NULLS FIRST
		LIMIT 1
	) UPDATE identities SET indexed_at = NOW()
	WHERE id = (SELECT id FROM next_identity)
	RETURNING owner_id, access_token`

	tx, err := d.DB.Beginx()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open transaction")
	}

	var identity Identity

	err = tx.Get(&identity, query)
	if err != nil {
		if err != sql.ErrNoRows {
			tx.Rollback()
			return nil, errors.Wrap(err, "failed to execute update query")
		}
	}

	return &identity, tx.Commit()
}
