package thingful

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thingful/kudzu/pkg/client"
	"github.com/thingful/kudzu/pkg/flowerpower"
	"github.com/thingful/kudzu/pkg/logger"
	"github.com/thingful/kudzu/pkg/postgres"
	registry "github.com/thingful/retryable-registry-prometheus"
	"github.com/thingful/thingfulx"
	"github.com/thingful/thingfulx/schema"
)

var (
	channelsCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "grow",
			Name:      "thingful_retreived_channel_count",
			Help:      "A counter of channels read back from Thingful when reading timeseries data",
		},
	)

	observationsCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "grow",
			Name:      "thingful_retrieved_observations_count",
			Help:      "A counter of observations read back from Thingful when reading timeseries data",
		},
	)
)

func init() {
	registry.MustRegister(channelsCount)
	registry.MustRegister(observationsCount)
}

// Thingful is our thingful client instance
type Thingful struct {
	client      *client.Client
	apiBase     string
	apiKey      string
	verbose     bool
	concurrency int
}

// NewClient creates a new Thingful client instance.
func NewClient(c *client.Client, apiBase, apiKey string, verbose bool, concurrency int) *Thingful {
	return &Thingful{
		client:      c,
		apiBase:     apiBase,
		apiKey:      apiKey,
		verbose:     verbose,
		concurrency: concurrency,
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

	url := fmt.Sprintf("%s/things/%s", t.apiBase, th.UID.String)

	_, err = t.client.Patch(ctx, url, t.apiKey, bytes.NewBuffer(b))
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
			ID:               "fertilizer_level",
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

// Thing is a single Thing response parsed from Thingful
type Thing struct {
	ID         string          `json:"id"`
	Attributes ThingAttributes `json:"attributes"`
}

// ThingAttributes sis a struct we return from the GetData function, which we also
// use to parse the returned JSON from Thingful.
type ThingAttributes struct {
	Title    string     `json:"title"`
	Location Location   `json:"location"`
	Metadata []Metadata `json:"metadata"`
	Channels []Channel  `json:"channels"`
}

// Metadata is used to parse metadata from the response
type Metadata struct {
	Prop string `json:"prop"`
	Val  string `json:"val"`
}

// Location is used to parse geolocation from the response
type Location struct {
	Longitude float64 `json:"long"`
	Latitude  float64 `json:"lat"`
}

// Channel is a unique data channel for a sensor. Contains some metadata and a
// list of observation values
type Channel struct {
	ID           string        `json:"id"`
	Unit         string        `json:"unit"`
	DataType     string        `json:"dataType"`
	Observations []Observation `json:"observations"`
}

// Observation is an individual recording of data at a location.
type Observation struct {
	RecordedAt time.Time `json:"recordedAt"`
	Value      string    `json:"value"`
}

// wrappedResponse is a simple container to handle responses sent back from
// goroutines indexing Thingful that holds either a slice of bytes or an error
type wrappedResponse struct {
	Data []byte
	Err  error
}

// GetData returns a slice of Thing instances retrieved from Thingful for the
// given time interval. We request all channels for a thing, and then filter
// rather inefficiently.
func (t *Thingful) GetData(ctx context.Context, uids []string, from, to time.Time, ascending bool) ([]Thing, error) {
	log := logger.FromContext(ctx)

	var wg sync.WaitGroup

	wrappedResponseChan := make(chan wrappedResponse, t.concurrency)

	// build a url and spawn a goroutine to fetch the data and return down channel
	for _, uid := range uids {
		u := fmt.Sprintf("%s/things/%s", t.apiBase, uid)
		parsedURL, err := url.Parse(u)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse url")
		}

		query := parsedURL.Query()
		query.Set("from", from.Format(time.RFC3339))
		query.Set("to", to.Format(time.RFC3339))

		parsedURL.RawQuery = query.Encode()

		wg.Add(1)

		// spawn goroutine to fetch data - note we have a buffered response channel,
		// so there is some backpressure here which I'm calling "concurrency"
		// (incorrectly)
		go func() {
			defer wg.Done()
			if t.verbose {
				log.Log(
					"msg", "fetching time series data from Thingful",
					"url", parsedURL.String(),
				)
			}

			// make the request to upstream Thingful service
			b, err := t.client.Get(ctx, parsedURL.String(), t.apiKey)

			// send any response back down our buffered channel
			wrappedResponseChan <- wrappedResponse{
				Data: b,
				Err:  err,
			}
		}()
	}

	// wait on all wait group elements reporting they are finished, and then close
	// the response channel
	go func() {
		wg.Wait()
		close(wrappedResponseChan)
	}()

	things := []Thing{}

	for wr := range wrappedResponseChan {
		if wr.Err != nil {
			return nil, errors.Wrap(wr.Err, "failed to receive value from channel")
		}

		thing, err := buildThing(wr.Data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build new thing to respond")
		}

		things = append(things, *thing)
	}

	return things, nil
}

// buildThing returns a thingful.Thing retreived from the Thingful database over
// the network.
func buildThing(b []byte) (*Thing, error) {
	var thingfulResp struct {
		Data Thing `json:"data"`
	}

	err := json.Unmarshal(b, &thingfulResp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal json into anonymous struct")
	}

	data := thingfulResp.Data

	// increment a couple of counters
	for _, ch := range data.Attributes.Channels {
		channelsCount.Inc()
		observationsCount.Add(float64(len(ch.Observations)))
	}

	return &data, nil
}
