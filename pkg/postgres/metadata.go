package postgres

import (
	"context"

	"github.com/guregu/null"
	"github.com/pkg/errors"
)

// Metadata is a struct used for returning metadata info from the DB
type Metadata struct {
	ID             int64     `db:"id"`
	DataSourceID   int64     `db:"data_source_id"`
	ThingUID       string    `db:"thing_uid"`
	FirstSampleUTC null.Time `db:"first_sample"`
	LastSampleUTC  null.Time `db:"last_sample"`
}

// GetMetadata returns a slice of all "metadata" in the db relating to available
// channels with timestamps
func (d *DB) GetMetadata(ctx context.Context) ([]Metadata, error) {
	sql := `SELECT c.id, c.data_source_id, c.thing_uid, t.first_sample, t.last_sample
		FROM channels c
		LEFT OUTER JOIN things t
		ON t.uid = c.thing_uid
		ORDER BY c.id`

	metadata := []Metadata{}

	rows, err := d.DB.Queryx(sql)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read metadata from the DB")
	}

	for rows.Next() {
		var m Metadata
		err = rows.StructScan(&m)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan struct from DB")
		}

		metadata = append(metadata, m)
	}

	return metadata, nil
}
