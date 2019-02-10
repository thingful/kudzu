package postgres

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/logger"
)

// NextAccessToken returns the next access token we want to attempt to index
// locations for. We load one that has either never been indexed before (i.e.
// with nil indexed_at), or the oldest one that hasn't already been indexed
// today.
func (d *DB) NextAccessToken(ctx context.Context) (string, error) {
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
	RETURNING access_token`

	tx, err := d.DB.Beginx()
	if err != nil {
		return "", errors.Wrap(err, "failed to open transaction")
	}

	var accessToken string

	err = tx.Get(&accessToken, query)
	if err != nil {
		if err != sql.ErrNoRows {
			tx.Rollback()
			return "", errors.Wrap(err, "failed to execute update query")
		}
	}

	return accessToken, tx.Commit()
}
