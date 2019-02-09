package thingful

import (
	"context"

	"github.com/thingful/kuzu/pkg/postgres"
)

type Thingful struct {
}

func NewThingful() *Thingful {
	return &Thingful{}
}

type Thing struct {
	UID string
}

func (t *Thingful) CreateThing(ctx context.Context, location *postgres.Location) (*Thing, error) {
	return nil, nil
}
