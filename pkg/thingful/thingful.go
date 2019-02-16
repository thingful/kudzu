package thingful

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/thingful/thingfulx"
	"github.com/thingful/thingfulx/schema"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/flowerpower"
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/postgres"
)

// Thingful is our thingful client instance
type Thingful struct {
	client  *client.Client
	apiBase string
	apiKey  string
	verbose bool
}

// NewClient creates a new Thingful client instance.
func NewClient(c *client.Client, apiBase, apiKey string, verbose bool) *Thingful {
	return &Thingful{
		client:  c,
		apiBase: apiBase,
		apiKey:  apiKey,
		verbose: verbose,
	}
}

// request is a type used for marshaling data to send to Thingful
type request struct {
	Data *data `json:"data"`
}

// data is a type used for sending post requests to Thingful to create resources
type data struct {
	Type       string `json:"type"`
	Attributes *thing `json:"attributes"`
}

// response is a type used for parsing the response from Thingful and extracting
// the created ID
type response struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

// updateRequest is a type used for marshalling data to send to Thingful to
// update a Thing
type updateRequest struct {
	Data *updateData `json:"data"`
}

// updateData is the child type used when sending data to Thingful
type updateData struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes *thing `json:"attributes"`
}

// thing wraps a thingfulx.Thing to add time series recording flag.
type thing struct {
	thingfulx.Thing
	Channels []channel `json:"channels"`
}

// channel wraps a thingfulx.Channel to add the flag
type channel struct {
	thingfulx.Channel
}

// MarshalJSON is our custom marshal implementation that adds extra required
// field for recording time series data.
func (c channel) MarshalJSON() ([]byte, error) {
	type C channel
	return json.Marshal(&struct {
		RecordTimeseries bool `json:"recordTimeSeries"`
		C
	}{
		RecordTimeseries: true,
		C:                (C)(c),
	})
}

// CreateThing sends a POST request to the Thingful API to create a new Thing.
// We also include in this request the first chunk of observations. We return
// the newly created UID for the Thing.
func (t *Thingful) CreateThing(ctx context.Context, th *postgres.Thing, readings []flowerpower.Reading) (string, error) {
	log := logger.FromContext(ctx)

	if t.verbose {
		log.Log(
			"msg", "creating thing on Thingful",
			"locationID", th.LocationID,
			"serialNum", th.SerialNum,
			"numReadings", len(readings),
		)
	}

	req := &request{
		Data: &data{
			Type: "thing",
			Attributes: &thing{
				Thing: thingfulx.Thing{
					Title:       th.Nickname.String,
					Description: "Soil sensor data produced by the GROW observatory. For more info see https://growobservatory.org",
					IndexedAt:   th.IndexedAt.Time,
					Webpage:     "http://global.parrot.com/au/products/flower-power/",
					Visibility:  thingfulx.Shared,
					Category:    thingfulx.Environment,
					Endpoint: &thingfulx.Endpoint{
						URL:         fmt.Sprintf("https://api-flower-power-pot.parrot.com/sensor_data/v6/sample/location/%s", th.LocationID),
						ContentType: "application/json",
					},
					Metadata: []thingfulx.Metadata{
						{
							Prop: "schema:serialNumber",
							Val:  th.SerialNum,
						},
						{
							Prop: "sem:hasEndTimeStamp",
							Val:  th.LastSampleUTC.Time.Format(time.RFC3339),
						},
					},
					ThingType: schema.Expand("thingful:ConnectedDevice"),
					Location: &thingfulx.Location{
						Lng: th.Longitude,
						Lat: th.Latitude,
					},
					DataLicense: thingfulx.GetDataLicense(thingfulx.CC0V1URL),
					Provider: &thingfulx.Provider{
						ID:          "flowerpower",
						Name:        "Parrot - Flower Power",
						Description: "Parrot SA is a french wireless products manufacturer company specialized in technologies involving voice recognition, signal processing for embedded products and drones.",
						URL:         "https://www.parrot.com/",
					},
				},
				Channels: buildChannels(readings, th.Longitude, th.Latitude),
			},
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal thing data")
	}

	respBytes, err := t.client.Post(ctx, fmt.Sprintf("%s/things", t.apiBase), t.apiKey, bytes.NewBuffer(b))
	if err != nil {
		return "", errors.Wrap(err, "failed to post thing data")
	}

	var resp response

	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal create response")
	}

	return path.Base(resp.Data.ID), nil
}

// UpdateThing sends a PATCH request to Thingful API to update a Thing,
// including updating it's location and writing any observations.
func (t *Thingful) UpdateThing(ctx context.Context, th *postgres.Thing, readings []flowerpower.Reading) error {
	log := logger.FromContext(ctx)

	if t.verbose {
		log.Log(
			"msg", "updating thing on Thingful",
			"locationID", th.LocationID,
			"uid", th.UID.String,
			"numReadings", len(readings),
		)
	}

	req := &updateRequest{
		Data: &updateData{
			Type: "thing",
			ID:   fmt.Sprintf("%s/things/%s", t.apiBase, th.UID.String),
			Attributes: &thing{
				Thing: thingfulx.Thing{
					Title: th.Nickname.String,
					Location: &thingfulx.Location{
						Lng: th.Longitude,
						Lat: th.Latitude,
					},
				},
				Channels: buildChannels(readings, th.Longitude, th.Latitude),
			},
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "failed to marshal thing data")
	}

	_, err = t.client.Patch(ctx, fmt.Sprintf("%s/things/%s", t.apiBase, th.UID.String), t.apiKey, bytes.NewBuffer(b))
	if err != nil {
		return errors.Wrap(err, "failed to patch thing data")
	}

	return nil
}

// buildChannels builds a slice of our custom channel type ready for sending to Thingful
func buildChannels(readings []flowerpower.Reading, long, lat float64) []channel {
	airObs := []thingfulx.Observation{}
	fertilizerObs := []thingfulx.Observation{}
	lightObs := []thingfulx.Observation{}
	soilObs := []thingfulx.Observation{}
	calibratedSoilObs := []thingfulx.Observation{}
	waterObs := []thingfulx.Observation{}
	batteryObs := []thingfulx.Observation{}

	for _, reading := range readings {
		location := &thingfulx.Location{
			Lng: long,
			Lat: lat,
		}

		airObs = append(airObs, thingfulx.Observation{
			RecordedAt: reading.Timestamp,
			Location:   location,
			Val:        strconv.FormatFloat(reading.AirTemperature, 'f', -1, 64),
		})

		fertilizerObs = append(fertilizerObs, thingfulx.Observation{
			RecordedAt: reading.Timestamp,
			Location:   location,
			Val:        strconv.FormatFloat(reading.FertilizerLevel, 'f', -1, 64),
		})

		lightObs = append(lightObs, thingfulx.Observation{
			RecordedAt: reading.Timestamp,
			Location:   location,
			Val:        strconv.FormatFloat(reading.Light, 'f', -1, 64),
		})

		soilObs = append(soilObs, thingfulx.Observation{
			RecordedAt: reading.Timestamp,
			Location:   location,
			Val:        strconv.FormatFloat(reading.SoilMoisture, 'f', -1, 64),
		})

		calibratedSoilObs = append(calibratedSoilObs, thingfulx.Observation{
			RecordedAt: reading.Timestamp,
			Location:   location,
			Val:        strconv.FormatFloat(reading.CalibratedSoilMoisture, 'f', -1, 64),
		})

		waterObs = append(waterObs, thingfulx.Observation{
			RecordedAt: reading.Timestamp,
			Location:   location,
			Val:        strconv.FormatFloat(reading.WaterTankLevel, 'f', -1, 64),
		})

		batteryObs = append(batteryObs, thingfulx.Observation{
			RecordedAt: reading.Timestamp,
			Location:   location,
			Val:        strconv.FormatFloat(reading.BatteryLevel, 'f', -1, 64),
		})
	}

	return []channel{
		airTemperatureChannel(airObs),
		fertilizerChannel(fertilizerObs),
		lightChannel(lightObs),
		soilChannel(soilObs),
		calibratedSoilChannel(calibratedSoilObs),
		waterChannel(waterObs),
		batteryChannel(batteryObs),
	}
}

func airTemperatureChannel(obs []thingfulx.Observation) channel {
	return channel{
		Channel: thingfulx.Channel{
			ID:               "air_temperature",
			MeasuredBy:       schema.Expand("m3-lite:AirThermometer"),
			QuantityKind:     schema.Expand("m3-lite:AirTemperature"),
			DomainOfInterest: []string{schema.Expand("m3-lite:Weather")},
			Unit:             schema.Expand("m3-lite:DegreeCelsius"),
			Type:             schema.DoubleType,
			Observations:     obs,
		},
	}
}

func fertilizerChannel(obs []thingfulx.Observation) channel {
	return channel{
		Channel: thingfulx.Channel{
			ID:               "fertilizer_channel",
			MeasuredBy:       schema.Expand("thingfulqu:FertilizerSensor"),
			QuantityKind:     schema.Expand("thingfulqu:FertilizerLevel"),
			DomainOfInterest: []string{schema.Expand("m3-lite:Environment"), schema.Expand("m3-lite:Agriculture")},
			Type:             schema.DoubleType,
			Observations:     obs,
		},
	}
}

func lightChannel(obs []thingfulx.Observation) channel {
	return channel{
		Channel: thingfulx.Channel{
			ID:               "light",
			MeasuredBy:       schema.Expand("m3-lite:LightSensor"),
			QuantityKind:     schema.Expand("m3-lite:Illuminance"),
			DomainOfInterest: []string{schema.Expand("m3-lite:Environment")},
			Unit:             schema.Expand("m3-lite:Lux"),
			Type:             schema.DoubleType,
			Observations:     obs,
		},
	}
}

func soilChannel(obs []thingfulx.Observation) channel {
	return channel{
		Channel: thingfulx.Channel{
			ID:               "soil_moisture",
			MeasuredBy:       schema.Expand("m3-lite:SoilHumiditySensor"),
			QuantityKind:     schema.Expand("m3-lite:SoilHumidity"),
			DomainOfInterest: []string{schema.Expand("m3-lite:Environment"), schema.Expand("m3-lite:Agriculture")},
			Unit:             schema.Expand("m3-lite:Percent"),
			Type:             schema.DoubleType,
			Observations:     obs,
		},
	}
}

func calibratedSoilChannel(obs []thingfulx.Observation) channel {
	return channel{
		Channel: thingfulx.Channel{
			ID:               "calibrated_soil_moisture",
			MeasuredBy:       schema.Expand("m3-lite:SoilHumiditySensor"),
			QuantityKind:     schema.Expand("m3-lite:SoilHumidity"),
			DomainOfInterest: []string{schema.Expand("m3-lite:Environment"), schema.Expand("m3-lite:Agriculture")},
			Unit:             schema.Expand("m3-lite:Percent"),
			Type:             schema.DoubleType,
			Observations:     obs,
		},
	}
}

func waterChannel(obs []thingfulx.Observation) channel {
	return channel{
		Channel: thingfulx.Channel{
			ID:               "water_tank_level",
			MeasuredBy:       schema.Expand("thingfulqu:WaterLevelSensor"),
			QuantityKind:     schema.Expand("m3-lite:WaterLevel"),
			DomainOfInterest: []string{schema.Expand("m3-lite:Environment")},
			Unit:             schema.Expand("m3-lite:Percent"),
			Type:             schema.DoubleType,
			Observations:     obs,
		},
	}
}

func batteryChannel(obs []thingfulx.Observation) channel {
	return channel{
		Channel: thingfulx.Channel{
			ID:               "battery_level",
			QuantityKind:     schema.Expand("m3-lite:BatteryLevel"),
			DomainOfInterest: []string{schema.Expand("m3-lite:EnergyDOI")},
			Unit:             schema.Expand("m3-lite:Percent"),
			Type:             schema.DoubleType,
			Observations:     obs,
		},
	}
}
