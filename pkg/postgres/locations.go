package postgres

import (
	"context"

	sq "github.com/elgris/sqrl"
	"github.com/guregu/null"
	"github.com/pkg/errors"
	"github.com/thingful/kudzu/pkg/logger"
)

// Location is a type used to return location information from  the database.
type Location struct {
	ID                    int64     `db:"id"`
	UID                   string    `db:"uid"`
	Longitude             float64   `db:"long"`
	Latitude              float64   `db:"lat"`
	FirstSampleUTC        null.Time `db:"first_sample"`
	LastSampleUTC         null.Time `db:"last_sample"`
	LastUploadedSampleUTC null.Time `db:"last_uploaded_sample"`
	Nickname              string    `db:"nickname"`
	LocationID            string    `db:"location_identifier"`
	SerialNum             string    `db:"serial_num"`
}

// ListLocationis returns a list of locations with some optional filtering parameters applied.
func (d *DB) ListLocations(ctx context.Context, ownerUID string, invalidLocation, staleData bool) ([]Location, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log(
			"msg", "listing locations",
			"ownerUID", ownerUID,
			"invalidLocation", invalidLocation,
			"staleData", staleData,
		)
	}

	builder := sq.Select(
		"t.id", "t.uid", "t.long", "t.lat", "t.first_sample", "t.last_sample",
		"t.last_uploaded_sample", "t.nickname", "t.location_identifier", "t.serial_num",
	).
		From("things t").
		Join("users u ON u.id = t.owner_id").
		OrderBy("t.uid")

	if ownerUID != "" {
		builder = builder.Where(sq.Eq{"u.uid": ownerUID})
	}

	if invalidLocation {
		builder = builder.Where(sq.Eq{"t.long": 0}).Where(sq.Eq{"t.lat": 0})
	}

	if staleData {
		builder = builder.Where("t.last_sample < NOW() - interval '30 days' AND t.last_sample >= NOW() - interval '90 days'")
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build sql query")
	}

	sql = d.DB.Rebind(sql)

	rows, err := d.DB.Queryx(sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute locations query")
	}

	locations := []Location{}

	for rows.Next() {
		var l Location
		err = rows.StructScan(&l)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan location struct")
		}
		locations = append(locations, l)
	}

	return locations, nil
}

// UpdateGeolocation takes as input a thing UID, and a long/lat pair and then is
// responsible for updating the DB with the new values.
func (d *DB) UpdateGeolocation(ctx context.Context, thingUID string, longitude, latitude float64) (*Location, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log(
			"msg", "updating geolocation",
			"thingUID", thingUID,
			"longitude", longitude,
			"latitude", latitude,
		)
	}

	sql := `UPDATE things SET long = :long, lat = :lat WHERE uid = :uid
		RETURNING id, uid, long, lat, first_sample, last_sample, last_uploaded_sample,
			nickname, location_identifier, serial_num`

	mapArgs := map[string]interface{}{
		"long": longitude,
		"lat":  latitude,
		"uid":  thingUID,
	}

	tx, err := d.DB.Beginx()
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction when updating geolocation")
	}

	sql, args, err := tx.BindNamed(sql, mapArgs)
	if err != nil {
		tx.Rollback()
		return nil, errors.Wrap(err, "failed to bind named parameters when updating geolocation")
	}

	var loc Location

	err = tx.Get(&loc, sql, args...)
	if err != nil {
		tx.Rollback()
		return nil, errors.Wrap(err, "failed to execute update query to update geolocation")
	}

	return &loc, tx.Commit()
}
