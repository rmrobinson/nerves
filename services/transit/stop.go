package transit

import (
	"fmt"
	"strconv"
	"strings"
)

// Float is a CSV marshalable float64 value
type Float struct {
	float64
}

// MarshalCSV marshals the value into a string format
func (f *Float) MarshalCSV() (string, error) {
	return fmt.Sprintf("%f", f), nil
}
// UnmarshalCSV takes the string representation from a CSV file and attempts to convert it to a float64.
func (f *Float) UnmarshalCSV(csv string) (err error) {
	f.float64, err = strconv.ParseFloat(strings.TrimSpace(csv), 64)
	return
}

// LocationType represents the possible set of location types
type LocationType int

const (
	// LocationTypeStop is a standard stop
	LocationTypeStop                LocationType = 0
	// LocationTypeStation is a transit station
	LocationTypeStation                          = 1
	// LocationTypeStationEntranceExit is the entrance/exit of a station.
	LocationTypeStationEntranceExit              = 2
)

// String presents the caller with a human readable version of this enum.
func (lt *LocationType) String() string {
	switch *lt {
	case LocationTypeStop:
		return "Stop"
	case LocationTypeStation:
		return "Station"
	case LocationTypeStationEntranceExit:
		return "Station Entrance/Exit"
	default:
		return "Unknown"
	}
}

// MarshalCSV converts this enum into a string for CSV writing.
func (lt *LocationType) MarshalCSV() (string, error) {
	return fmt.Sprintf("%d", lt), nil
}

// UnmarshalCSV attempts to convert a string value from a CSV file into the enum value.
func (lt *LocationType) UnmarshalCSV(csv string) error {
	val, err := strconv.ParseInt(strings.TrimSpace(csv), 10, 32)
	if err != nil {
		return err
	}

	*lt = LocationType(val)
	return nil
}

// Stop represents a single point that one or more trips may visit.
type Stop struct {
	ID                 string       `csv:"stop_id"`
	Code               string       `csv:"stop_code"`
	Name               string       `csv:"stop_name"`
	Description        string       `csv:"stop_desc"`
	Latitude           Float        `csv:"stop_lat"`
	Longitude          Float        `csv:"stop_lon"`
	ZoneID             string       `csv:"zone_id"`
	StopURL            string       `csv:"stop_url"`
	LocationType       LocationType `csv:"location_type"`
	ParentStation      string       `csv:"parent_station"`
	StopTZ             string       `csv:"stop_timezone"`
	WheelchairBoarding string       `csv:"wheelchair_boarding"`
}
