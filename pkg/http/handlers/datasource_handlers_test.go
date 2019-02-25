package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/guregu/null"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/thingful/kudzu/pkg/http/handlers"
	"github.com/thingful/kudzu/pkg/logger"
	"github.com/thingful/kudzu/pkg/postgres"
	"github.com/thingful/kudzu/pkg/postgres/helper"
	goji "goji.io"
)

type DatasourceHandlersSuite struct {
	suite.Suite
	db     *postgres.DB
	logger kitlog.Logger
}

func (s *DatasourceHandlersSuite) SetupTest() {
	log := kitlog.NewNopLogger()
	connStr := os.Getenv("KUDZU_DATABASE_URL")

	s.logger = log
	s.db = helper.PrepareDB(s.T(), connStr, s.logger)
}

func (s *DatasourceHandlersSuite) TearDownTest() {
	helper.CleanDB(s.T(), s.db)
}

func (s *DatasourceHandlersSuite) TestGetDatasources() {
	ctx := logger.ToContext(context.Background(), s.logger)

	datasources := []struct {
		name     string
		unit     null.String
		dataType string
	}{
		{
			name:     "air_temperature",
			unit:     null.StringFrom("m3-lite:DegreeCelsius"),
			dataType: "xsd:double",
		},
		{
			name:     "fertilizer_level",
			dataType: "xsd:double",
		},
		{
			name:     "light",
			unit:     null.StringFrom("m3-lite:Lux"),
			dataType: "xsd:double",
		},
		{
			name:     "soil_moisture",
			unit:     null.StringFrom("m3-lite:Percent"),
			dataType: "xsd:double",
		},
		{
			name:     "calibrated_soil_moisture",
			unit:     null.StringFrom("m3-lite:Percent"),
			dataType: "xsd:double",
		},
		{
			name:     "water_tank_level",
			unit:     null.StringFrom("m3-lite:Percent"),
			dataType: "xsd:double",
		},
		{
			name:     "battery_level",
			unit:     null.StringFrom("m3-lite:Percent"),
			dataType: "xsd:double",
		},
	}

	for _, d := range datasources {
		_, err := s.db.DB.Exec(
			`INSERT INTO data_sources (name, unit, data_type) VALUES ($1, $2, $3)`,
			d.name, d.unit, d.dataType,
		)
		assert.Nil(s.T(), err)
	}

	mux := goji.NewMux()
	handlers.RegisterDataSourceHandlers(mux, s.db)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/entity/dataSourceVariables/get", nil)
	assert.Nil(s.T(), err)
	req = req.WithContext(ctx)

	mux.ServeHTTP(recorder, req)
	assert.Equal(s.T(), http.StatusOK, recorder.Code)

	expected := `
	{
  	"DataSourceVariables": {
  	  "1": {
  	    "DataSourceVariableId": 1,
  	    "VariableCode": "Thingful.Connectors.GROWSensors.air_temperature",
  	    "DataSourceCode": "Thingful.Connectors.GROWSensors",
  	    "Name": "Air Temperature",
  	    "Code": "air_temperature",
  	    "UnitCode": "C",
  	    "DataType": "Double",
  	    "MathematicalType": "NotSummable",
  	    "MeasurementType": "Instantaneous",
  	    "State": 1,
  	    "IsCumulative": false
  	  },
  	  "2": {
  	    "DataSourceVariableId": 2,
  	    "VariableCode": "Thingful.Connectors.GROWSensors.fertilizer_level",
  	    "DataSourceCode": "Thingful.Connectors.GROWSensors",
  	    "Name": "Fertilizer Level",
  	    "Code": "fertilizer_level",
  	    "UnitCode": "mS/cm",
  	    "DataType": "Double",
  	    "MathematicalType": "NotSummable",
  	    "MeasurementType": "Instantaneous",
  	    "State": 1,
  	    "IsCumulative": false
  	  },
  	  "3": {
  	    "DataSourceVariableId": 3,
  	    "VariableCode": "Thingful.Connectors.GROWSensors.light",
  	    "DataSourceCode": "Thingful.Connectors.GROWSensors",
  	    "Name": "Light",
  	    "Code": "light",
  	    "UnitCode": "mol/m2/d",
  	    "DataType": "Double",
  	    "MathematicalType": "NotSummable",
  	    "MeasurementType": "Instantaneous",
  	    "State": 1,
  	    "IsCumulative": false
  	  },
  	  "4": {
  	    "DataSourceVariableId": 4,
  	    "VariableCode": "Thingful.Connectors.GROWSensors.soil_moisture",
  	    "DataSourceCode": "Thingful.Connectors.GROWSensors",
  	    "Name": "Soil Moisture",
  	    "Code": "soil_moisture",
  	    "UnitCode": "%",
  	    "DataType": "Double",
  	    "MathematicalType": "NotSummable",
  	    "MeasurementType": "Instantaneous",
  	    "State": 1,
  	    "IsCumulative": false
  	  },
  	  "5": {
  	    "DataSourceVariableId": 5,
  	    "VariableCode": "Thingful.Connectors.GROWSensors.calibrated_soil_moisture",
  	    "DataSourceCode": "Thingful.Connectors.GROWSensors",
  	    "Name": "Calibrated Soil Moisture",
  	    "Code": "calibrated_soil_moisture",
  	    "UnitCode": "%",
  	    "DataType": "Double",
  	    "MathematicalType": "NotSummable",
  	    "MeasurementType": "Instantaneous",
  	    "State": 1,
  	    "IsCumulative": false
  	  },
  	  "6": {
  	    "DataSourceVariableId": 6,
  	    "VariableCode": "Thingful.Connectors.GROWSensors.water_tank_level",
  	    "DataSourceCode": "Thingful.Connectors.GROWSensors",
  	    "Name": "Water Tank Level",
  	    "Code": "water_tank_level",
  	    "UnitCode": "%",
  	    "DataType": "Double",
  	    "MathematicalType": "NotSummable",
  	    "MeasurementType": "Instantaneous",
  	    "State": 1,
  	    "IsCumulative": false
  	  },
  	  "7": {
  	    "DataSourceVariableId": 7,
  	    "VariableCode": "Thingful.Connectors.GROWSensors.battery_level",
  	    "DataSourceCode": "Thingful.Connectors.GROWSensors",
  	    "Name": "Battery Level",
  	    "Code": "battery_level",
  	    "UnitCode": "%",
  	    "DataType": "Double",
  	    "MathematicalType": "NotSummable",
  	    "MeasurementType": "Instantaneous",
  	    "State": 1,
  	    "IsCumulative": false
  	  }
  	}
	}`

	assert.JSONEq(s.T(), expected, recorder.Body.String())
}

func TestDatasourceHandlersSuite(t *testing.T) {
	suite.Run(t, new(DatasourceHandlersSuite))
}
