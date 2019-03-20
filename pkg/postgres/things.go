package postgres

import (
	"context"

	"github.com/guregu/null"
	"github.com/pkg/errors"

	"github.com/thingful/kudzu/pkg/logger"
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

	// TODO - delete when old server removed
	DataURL     null.String `db:"data_url"`
	ResourceURL null.String `db:"resource_url"`
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

// GetThingByUID returns a thing on being given it's Thingful generated UID.
// Added as a separate function as we do load Things in two different ways, in
// two different places.
func (d *DB) GetThingByUID(ctx context.Context, uid string) (*Thing, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log(
			"msg", "getting thing by uid",
			"uid", uid,
		)
	}

	sql := `SELECT * FROM things WHERE uid = $1`

	var thing Thing

	err := d.DB.Get(&thing, sql, uid)
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

// UpdateNickname updates just a devices nickname
func (d *DB) UpdateNickname(ctx context.Context, locationID, nickname string) error {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log(
			"msg", "updating nickname",
			"locationID", locationID,
			"nickname", nickname,
		)
	}

	sql := `UPDATE things SET
		nickname = :nickname,
		updated_at = NOW(),
		indexed_at = NOW()
	WHERE location_identifier = :location_identifier`

	mapArgs := map[string]interface{}{
		"nickname":            nickname,
		"location_identifier": locationID,
	}

	tx, err := d.DB.Beginx()
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}

	sql, args, err := tx.BindNamed(sql, mapArgs)
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

// ThingStats is a data structure used to pass
type ThingStats struct {
	All             float64 `db:"all_things"`
	Live            float64 `db:"live_things"`
	Stale           float64 `db:"stale_things"`
	Dead            float64 `db:"dead_things"`
	InvalidLocation float64 `db:"invalid_location_things"`
	Provider        string  `db:"provider"`
}

// GetThingStats returns a data structure containing some thing metrics for
// Prometheus.
func (d *DB) GetThingStats(ctx context.Context) ([]ThingStats, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log("msg", "counting things")
	}

	sql := `SELECT
		COUNT(*) AS all_things,
		COUNT(live) AS live_things,
		COUNT(stale) AS stale_things,
		COUNT(invalid_location) AS invalid_location_things,
		COUNT(dead) AS dead_things,
		provider
	FROM (
		SELECT
			CASE WHEN last_sample < NOW() - interval '30 days' AND last_sample >= NOW() - interval '90 days' THEN 1 END stale,
			CASE WHEN last_sample >= NOW() - interval '30 days' THEN 1 END live,
			CASE WHEN last_sample < NOW() - interval '90 days' THEN 1 END dead,
			CASE WHEN lat = 0 AND long = 0 AND last_sample >= NOW() - interval '90 days' THEN 1 END invalid_location,
			provider
		FROM things
	) things GROUP BY provider`

	rows, err := d.DB.Queryx(sql)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get thing stats")
	}

	stats := []ThingStats{}

	for rows.Next() {
		var stat ThingStats
		err = rows.StructScan(&stat)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan things stat struct")
		}

		stats = append(stats, stat)
	}

	return stats, nil
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
