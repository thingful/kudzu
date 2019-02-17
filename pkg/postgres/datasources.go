package postgres

import (
	"context"

	"github.com/guregu/null"
	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/logger"
)

// DataSource is used to retrieve data source information from the database.
type DataSource struct {
	ID       int64       `db:"id"`
	Unit     null.String `db:"unit"`
	Code     string      `db:"name"`
	DataType string      `db:"data_type"`
}

// GetDataSources returns a list of all available data sources.
func (d *DB) GetDataSources(ctx context.Context) ([]DataSource, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log(
			"msg", "getting data sources",
		)
	}

	sql := `SELECT * FROM data_sources`

	rows, err := d.DB.Queryx(sql)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read data sources")
	}

	datasources := []DataSource{}

	for rows.Next() {
		var ds DataSource
		err = rows.StructScan(&ds)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan datasource struct")
		}
		datasources = append(datasources, ds)
	}

	return datasources, nil
}
