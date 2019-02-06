package flowerpower

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/client"
)

const (
	// ConfigurationURL is parrot's configuration info URL
	ConfigurationURL = "https://api-flower-power-pot.parrot.com/garden/v2/configuration"

	// StatusURL is parrot's status URL
	StatusURL = "https://api-flower-power-pot.parrot.com/garden/v1/status"
)

// configuration is a type used when parsing the output from the app
type configuration struct {
	Locations []configurationLocation `json:"locations"`
}

type configurationLocation struct {
	LocationID string `json:"location_identifier"`
}

type status struct {
	Locations []statusLocation `json:"locations"`
}

type statusLocation struct {
	FirstSampleUTC time.Time `json:"first_sample_utc"`
	LastSampleUTC  time.Time `json:"last_sample_utc"`
}

// SensorCount attempts to return a count of the number of sensors that a user
// identified by the given access token owns. Will return an error if the given
// credential is not valid.
func SensorCount(client *client.Client, accessToken string) (int, error) {
	return countValidSensors(client, accessToken)
}

func countValidSensors(client *client.Client, accessToken string) (int, error) {
	b, err := client.Get(StatusURL, accessToken)
	if err != nil {
		return 0, errors.Wrap(err, "faield to retrieve status information")
	}

	var sts status

	err = json.Unmarshal(b, &sts)
	if err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal status json")
	}

	count := 0

	for _, location := range sts.Locations {
		if isValid(&location) {
			count = count + 1
		}
	}

	return count, nil
}

// isValid checks the validity of the location/sensor. We class a
// location/sensor as valid if it has a non-zero first and last sample
// timestamp, and they must not equal each other.
func isValid(location *statusLocation) bool {
	if location.FirstSampleUTC.IsZero() || location.LastSampleUTC.IsZero() || location.FirstSampleUTC.Equal(location.LastSampleUTC) {
		return false
	}
	return true
}
