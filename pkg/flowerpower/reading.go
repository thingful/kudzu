package flowerpower

import "time"

// Reading is a struct used when parsing sensor data from parrot
type Reading struct {
	Timestamp              time.Time `json:"capture_datetime_utc"`
	Light                  float64   `json:"light"`
	FertilizerLevel        float64   `json:"fertilizer_level"`
	AirTemperature         float64   `json:"air_temperature_celsius"`
	SoilMoisture           float64   `json:"soil_moisture_percent"`
	BatteryLevel           float64   `json:"battery_percent"`
	WaterTankLevel         float64   `json:"water_tank_level_percent"`
	CalibratedSoilMoisture float64   `json:"calibrated_soil_moisture_percent"`
}

// SampleData is the whole response we get when reading samples, contains an
// array of readings for the location.
type SampleData struct {
	Readings []Reading `json:"samples"`
}
