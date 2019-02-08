package postgres

import (
	"context"

	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/flowerpower"
	"github.com/thingful/kuzu/pkg/logger"
)

// SaveLocations saves a slice of flowerpower Locations which the indexer will
// in the background churn through to index. This function is typically called
// when we first create a new user record so at this point all we know are the
// Parrot details.
func (d *DB) SaveLocations(ctx context.Context, ownerID int64, locations []flowerpower.Location) error {
	log := logger.FromContext(ctx)

	log.Log("msg", "saving locations")

	baseSQL := `INSERT INTO things (
		owner_id, nickname, location_identifier, serial_num, long, lat, first_sample, last_sample
	) VALUES (
		:owner_id, :nickname, :location_identifier, :serial_num, :long, :lat, :first_sample, :last_sample
	)`

	tx, err := d.DB.Beginx()
	if err != nil {
		return errors.Wrap(err, "failed to open transaction")
	}

	for _, l := range locations {
		mapArgs := map[string]interface{}{
			"owner_id":            ownerID,
			"nickname":            l.Nickname,
			"location_identifier": l.LocationID,
			"serial_num":          l.SerialNum,
			"long":                l.Longitude,
			"lat":                 l.Latitude,
			"first_sample":        l.FirstSampleUTC,
			"last_sample":         l.LastSampleUTC,
		}

		sql, args, err := tx.BindNamed(baseSQL, mapArgs)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to bind query parameters")
		}

		_, err = tx.Exec(sql, args...)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to insert location")
		}
	}

	return tx.Commit()
}
