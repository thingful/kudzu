package handlers

import "github.com/guregu/null"

var (
	datatypes = map[string]string{
		"xsd:double":  "Double",
		"xsd:integer": "Integer",
		"xsd:string":  "String",
	}

	units = map[string]string{
		"WaterTankLevel":        "%",
		"Water_Tank_Level":      "%",
		"SoilMoisture":          "%",
		"soil_moisture":         "%",
		"Light":                 "mol/m2/d",
		"light":                 "mol/m2/d",
		"FertilizerLevel":       "mS/cm",
		"fertilizer_level":      "mS/cm",
		"AirTemperature":        "C",
		"air_temperature":       "C",
		"m3-lite:DegreeCelsius": "C",
		"BatteryLevel":          "%",
		"battery_level":         "%",
		"m3-lite:Percent":       "%",
	}
)

// unitToHN4 attempts to return a HydroNet4 version of a unit or the original if
// we don't have a value
func unitToHN4(datasource string, unit null.String) string {
	if convertedUnit, ok := units[datasource]; ok {
		return convertedUnit
	}

	if unit.Valid {
		if convertedUnit, ok := units[unit.String]; ok {
			return convertedUnit
		}
		return unit.String
	}

	return ""
}

// dataTypeToHN4 attempts to return a HydroNet4 version of a datatype or the
// original if we don't have a value
func dataTypeToHN4(datatype string) string {
	if convertedDataType, ok := datatypes[datatype]; ok {
		return convertedDataType
	}
	return datatype
}
