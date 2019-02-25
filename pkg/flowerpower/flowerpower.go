package flowerpower

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/pkg/errors"
	"github.com/thingful/kudzu/pkg/client"
)

var (
	locationsCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "grow",
			Name:      "parrot_retrieved_locations_count",
			Help:      "A counter of received locations from Parrot",
		},
	)

	readingsCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "grow",
			Name:      "parrot_retrieved_readings_count",
			Help:      "A counter of received readings from Parrot",
		},
	)
)

func init() {
	prometheus.MustRegister(locationsCounter)
	prometheus.MustRegister(readingsCounter)
}

const (
	// ProfileURL is parrot's user profile URL
	ProfileURL = "https://api-flower-power-pot.parrot.com/user/v4/profile"

	// ConfigurationURL is parrot's configuration info URL
	ConfigurationURL = "https://api-flower-power-pot.parrot.com/garden/v2/configuration"

	// StatusURL is parrot's status URL for a users sensors
	StatusURL = "https://api-flower-power-pot.parrot.com/garden/v1/status"

	// DataURL is parrot's URL from which we can retrieve data keyed by sensor serial number
	DataURL = "https://api-flower-power-pot.parrot.com/sensor_data/v6/sample/location/%s"
)

// statusData is a type used when parsing status response from Parrot
type statusData struct {
	Locations []statusLocation `json:"locations"`
}

// statusLocation is a specific location parsed from the Parrot status response
type statusLocation struct {
	LocationID     string    `json:"location_identifier"`
	FirstSampleUTC time.Time `json:"first_sample_utc"`
	LastSampleUTC  time.Time `json:"last_sample_utc"`
}

// configurationData is a type used when parsing the response from Parrot
type configurationData struct {
	Locations []configurationLocation `json:"locations"`
}

// configurationLocation is a specific location parsed from Parrot
type configurationLocation struct {
	LocationID string              `json:"location_identifier"`
	Sensor     configurationSensor `json:"sensor"`
	Nickname   string              `json:"plant_nickname"`
	Longitude  float64             `json:"longitude"`
	Latitude   float64             `json:"latitude"`
}

// configurationSensor captures the serial number in the response from parrot
type configurationSensor struct {
	SerialNum string `json:"sensor_serial"`
}

// userData is a type used when parsing the user profile response
type userData struct {
	Profile userProfile `json:"user_profile"`
}

// userProfile contains the one field we parse from the user profile data
type userProfile struct {
	Email string `json:"email"`
}

// GetUser attempts to read the Parrot user from their API. Returns an error if
// we cannot read the user.
func GetUser(ctx context.Context, client *client.Client, accessToken string) (*User, error) {
	profileBytes, err := client.Get(ctx, ProfileURL, accessToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve user profile data")
	}

	var u userData

	err = json.Unmarshal(profileBytes, &u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal user profile data")
	}

	return &User{
		ParrotID: u.Profile.Email,
	}, nil
}

// GetLocations attempts to return a slice containing all the locations owned by
// the user identified by the given access token. Will return an error if the
// given credential is not valid. Does not return any sensor values, these must
// be retrieved separately.
func GetLocations(ctx context.Context, client *client.Client, accessToken string) ([]Location, error) {
	statusBytes, err := client.Get(ctx, StatusURL, accessToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve status data")
	}

	var statusLocations statusData

	err = json.Unmarshal(statusBytes, &statusLocations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal status json")
	}

	configurationBytes, err := client.Get(ctx, ConfigurationURL, accessToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve configuration data")
	}

	var configurationLocations configurationData

	err = json.Unmarshal(configurationBytes, &configurationLocations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configuration json")
	}

	// create an empty slice for return valid locations
	locations := []Location{}

	for _, l := range statusLocations.Locations {
		if isValid(&l) {
			// find the corresponding configuration location
			cl := findConfigurationLocation(l.LocationID, configurationLocations.Locations)
			if cl != nil && cl.Sensor.SerialNum != "" {
				// create a new Location and append to slice
				location := Location{
					LocationID:     l.LocationID,
					SerialNum:      cl.Sensor.SerialNum,
					Nickname:       cl.Nickname,
					FirstSampleUTC: l.FirstSampleUTC,
					LastSampleUTC:  l.LastSampleUTC,
					Longitude:      cl.Longitude,
					Latitude:       cl.Latitude,
				}

				locations = append(locations, location)
			}
		}
	}

	locationsCounter.Add(float64(len(locations)))

	return locations, nil
}

// GetReadings reads a slice of sensor readings from flower power for a given
// location, between the specified start and end times. Returns either a slice
// of values or an error.
func GetReadings(ctx context.Context, client *client.Client, accessToken, locationID string, from, to time.Time) ([]Reading, error) {
	locationURL, err := url.Parse(fmt.Sprintf(DataURL, locationID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse location url")
	}
	q := locationURL.Query()
	q.Set("from_datetime_utc", from.Format(time.RFC3339))
	q.Set("to_datetime_utc", to.Format(time.RFC3339))

	locationURL.RawQuery = q.Encode()

	b, err := client.Get(ctx, locationURL.String(), accessToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch sensor data")
	}

	var data SampleData
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal sample data json")
	}

	readingsCounter.Add(float64(len(data.Readings)))

	return data.Readings, nil
}

// findConfigurationLocation attempts to find a configuration location from the
// passed in slice or returns nil if the location cannot be found
func findConfigurationLocation(locationID string, locations []configurationLocation) *configurationLocation {
	for _, l := range locations {
		if locationID == l.LocationID {
			return &l
		}
	}

	return nil
}

// isValid checks the validity of the location/sensor. We class a
// location/sensor as valid if it has a non-zero first and last sample
// timestamp, and they must not equal each other, and the serial number and
// location are non empty strings.
func isValid(location *statusLocation) bool {
	if location.FirstSampleUTC.IsZero() ||
		location.LastSampleUTC.IsZero() ||
		location.FirstSampleUTC.Equal(location.LastSampleUTC) {
		return false
	}
	return true
}
