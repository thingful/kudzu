package postgres

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"github.com/thingful/kudzu/pkg/logger"
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
		ORDER BY indexed_at ASC NULLS FIRST, created_at DESC
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

// IdentityStat is used for exporting identity stats from the DB
type IdentityStat struct {
	All     float64 `db:"all_identities"`
	Pending float64 `db:"pending_identities"`
	Stale   float64 `db:"stale_identities"`
}

// GetIdentityStats returns an IdentityStat instance containing some info about
// the state of the identities table suitable for rendering in Prometheus
func (d *DB) GetIdentityStats(ctx context.Context) (*IdentityStat, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log("msg", "counting identities")
	}

	sql := `SELECT
			COUNT(*) AS all_identities,
			COUNT(pending) AS pending_identities,
			COUNT(stale) AS stale_identities
		FROM (
			SELECT
			CASE WHEN indexed_at IS NULL THEN 1 END pending,
			CASE WHEN indexed_at < NOW() - interval '2 days' THEN 1 END stale
			 FROM identities
		) identities`

	var stat IdentityStat

	err := d.DB.Get(&stat, sql)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read identity stats")
	}

	return &stat, nil
}
