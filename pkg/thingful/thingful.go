package thingful

import (
	"context"

	"github.com/thingful/kuzu/pkg/client"
	"github.com/thingful/kuzu/pkg/flowerpower"
	"github.com/thingful/kuzu/pkg/postgres"
)

// Thingful is our thingful client instance
type Thingful struct {
	client *client.Client
	apiURL string
	apiKey string
}

// NewClient creates a new Thingful client instance.
func NewClient(c *client.Client, apiURL, apiKey string) *Thingful {
	return &Thingful{
		client: c,
		apiURL: apiURL,
		apiKey: apiKey,
	}
}

// CreateThing sends a POST request to the Thingful API to create a new Thing.
// We also include in this request the first chunk of observations. We return
// the newly created UID for the Thing.
func (t *Thingful) CreateThing(ctx context.Context, thing *postgres.Thing, readings []flowerpower.Reading) (string, error) {
	return "", nil
}

// UpdateThing sends a PATCH request to Thingful API to update a Thing,
// including updating it's location and writing any observations.
func (t *Thingful) UpdateThing(ctx context.Context, thing *postgres.Thing, readings []flowerpower.Reading) error {
	return nil
}
