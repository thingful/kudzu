package flowerpower

import "time"

// Location is a type
type Location struct {
	Nickname       string
	LocationID     string
	SerialNum      string
	FirstSampleUTC time.Time
	LastSampleUTC  time.Time
	Longitude      float64
	Latitude       float64
}
