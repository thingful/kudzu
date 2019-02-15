package postgres

import (
	"context"

	"github.com/guregu/null"
	"github.com/pkg/errors"

	"github.com/thingful/kuzu/pkg/logger"
)

// Thing is our representation of a sensor read from Postgres
type Thing struct {
	ID              int64       `db:"id"`
	UID             null.String `db:"uid"`
	OwnerID         int64       `db:"owner_id"`
	Provider        null.String `db:"provider"`
	SerialNum       string      `db:"serial_num"`
	Latitude        float64     `db:"lat"`
	Longitude       float64     `db:"long"`
	FirstSampleUTC  null.Time   `db:"first_sample"`
	LastSampleUTC   null.Time   `db:"last_sample"`
	CreatedAt       null.Time   `db:"created_at"`
	UpdatedAt       null.Time   `db:"updated_at"`
	IndexedAt       null.Time   `db:"indexed_at"`
	Nickname        null.String `db:"nickname"`
	LastUploadedUTC null.Time   `db:"last_uploaded_sample"`
	LocationID      string      `db:"location_identifier"`
	Channels        []Channel
}

// Channel is used to persist channel information to the database
type Channel struct {
	Name         string      `db:"name"`
	Unit         null.String `db:"unit"`
	DataType     string      `db:"data_type"`
	ThingUID     string      `db:"thing_uid"`
	DataSourceID int64       `db:"data_source_id"`
}

// GetThing attempts to load a thing identified by the location ID from the
// database. Clients can unwrap the returned error to check for an sql.ErrNoRows
// error to determine if no record exist.
func (d *DB) GetThing(ctx context.Context, locationID string) (*Thing, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log("msg", "loading thing by location identifier", "locationID", locationID)
	}

	sql := `SELECT * FROM things WHERE location_identifier = $1`

	var thing Thing

	err := d.DB.Get(&thing, sql, locationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load thing from DB")
	}

	return &thing, nil
}

// CreateThing inserts new thing record, then upserts data sources before
// inserting channels
func (d *DB) CreateThing(ctx context.Context, thing *Thing) error {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log(
			"msg", "saving new thing",
			"locationID", thing.LocationID,
			"lastUploadedUTC", thing.LastUploadedUTC.Time,
		)
	}

	sql := `INSERT INTO things
		(uid, owner_id, provider, serial_num, lat, long, first_sample, last_sample, created_at,
		 	indexed_at, updated_at, nickname, last_uploaded_sample, location_identifier)
		VALUES (:uid, :owner_id, :provider, :serial_num, :lat, :long, :first_sample, :last_sample,
			:created_at, :indexed_at, :updated_at, :nickname, :last_uploaded_sample, :location_identifier)
		RETURNING id`

	tx, err := d.DB.Beginx()
	if err != nil {
		return errors.Wrap(err, "failed to open transaction")
	}

	sql, args, err := tx.BindNamed(sql, thing)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to bind named thing query")
	}

	var thingID int64

	err = tx.Get(&thingID, sql, args...)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to execute thing insertion")
	}

	// upsert data_sources
	datasourceSQL := `INSERT INTO data_sources
			(name, unit, data_type)
		VALUES (:name, :unit, :data_type)
		ON CONFLICT (name)
		DO UPDATE SET unit = EXCLUDED.unit, data_type = EXCLUDED.data_type
		RETURNING id`

	channelSQL := `INSERT INTO channels
		(thing_uid, data_source_id)
	VALUES (:thing_uid, :data_source_id)`

	channels := makeChannels()

	for _, c := range channels {
		sql, args, err = tx.BindNamed(datasourceSQL, c)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to bind named datasource query")
		}

		var datasourceID int64
		err = tx.Get(&datasourceID, sql, args...)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to execute datasource query")
		}

		channelArgs := map[string]interface{}{
			"thing_uid":      thing.UID,
			"data_source_id": datasourceID,
		}

		sql, args, err = tx.BindNamed(channelSQL, channelArgs)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to bind named channel query")
		}

		_, err := tx.Exec(sql, args...)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to execute channel query")
		}
	}

	return tx.Commit()
}

// UpdateThing updates a thing record in the database - here we just update the
// nickname and timestamp
func (d *DB) UpdateThing(ctx context.Context, thing *Thing) error {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log(
			"msg", "updating existing thing",
			"locationID", thing.LocationID,
			"lastUploadedUTC", thing.LastUploadedUTC.Time,
		)
	}

	sql := `UPDATE things SET
		nickname = :nickname,
		first_sample = :first_sample,
		last_sample = :last_sample,
		updated_at = :updated_at,
		indexed_at = :indexed_at,
		last_uploaded_sample = :last_uploaded_sample
	WHERE location_identifier = :location_identifier`

	tx, err := d.DB.Beginx()
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}

	sql, args, err := tx.BindNamed(sql, thing)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to bind named query")
	}

	_, err = tx.Exec(sql, args...)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update thing")
	}

	return tx.Commit()
}

// UpdateGeolocation takes as input a Thing with UID, long and lat, and updates
// the value stored in the DB for that thing
func (d *DB) UpdateGeolocation(ctx context.Context, thing *Thing) error {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log(
			"msg", "updating geolocation",
			"uid", thing.UID,
			"longitude", thing.Longitude,
			"latitude", thing.Latitude,
		)
	}

	sql := `UPDATE things SET long = :long, lat = :lat WHERE uid = :uid`

	tx, err := d.DB.Beginx()
	if err != nil {
		return errors.Wrap(err, "failed to open transaction when updating geolocation")
	}

	sql, args, err := tx.BindNamed(sql, thing)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to bind named transaction when updating geolocation")
	}

	_, err = tx.Exec(sql, args...)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update geolocation")
	}

	return tx.Commit()
}

// makeChannels returns static list of channels for flowerpower devices
func makeChannels() []Channel {
	return []Channel{
		{
			Name:     "air_temperature",
			Unit:     null.StringFrom("m3-lite:DegreeCelsius"),
			DataType: "xsd:double",
		},
		{
			Name:     "fertilizer_level",
			DataType: "xsd:double",
		},
		{
			Name:     "light",
			Unit:     null.StringFrom("m3-lite:Lux"),
			DataType: "xsd:double",
		},
		{
			Name:     "soil_moisture",
			Unit:     null.StringFrom("m3-lite:Percent"),
			DataType: "xsd:double",
		},
		{
			Name:     "calibrated_soil_moisture",
			Unit:     null.StringFrom("m3-lite:Percent"),
			DataType: "xsd:double",
		},
		{
			Name:     "water_tank_level",
			Unit:     null.StringFrom("m3-lite:Percent"),
			DataType: "xsd:double",
		},
		{
			Name:     "battery_level",
			Unit:     null.StringFrom("m3-lite:Percent"),
			DataType: "xsd:double",
		},
	}
}
