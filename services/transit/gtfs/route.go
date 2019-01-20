package gtfs

import (
	"fmt"
	"strconv"
	"strings"
)

// RouteType represents the possible set of route types
type RouteType int

const (
	// RouteTypeLRT is a route served by an LRT or streetcar
	RouteTypeLRT RouteType = 0
	// RouteTypeSubway is a route served by a subway
	RouteTypeSubway = 1
	// RouteTypeRail is a route served by a heavy rail system
	RouteTypeRail = 2
	// RouteTypeBus is a route served by a bus
	RouteTypeBus = 3
	// RouteTypeFerry is a route served by a ferry
	RouteTypeFerry = 4
	// RouteTypeCableTram is a route served by a cable-driven tram system
	RouteTypeCableTram = 5
	// RouteTypeAerialLift is a route served by an aerial lift system
	RouteTypeAerialLift = 6
	// RouteTypeFunicular is a route served by a funicular system
	RouteTypeFunicular = 7
)

// String presents the caller with a human readable version of this enum.
func (rt *RouteType) String() string {
	switch *rt {
	case RouteTypeLRT:
		return "LRT/Streetcar"
	case RouteTypeSubway:
		return "Subway"
	case RouteTypeRail:
		return "Rail"
	case RouteTypeBus:
		return "Bus"
	case RouteTypeFerry:
		return "Ferry"
	case RouteTypeCableTram:
		return "Tram"
	case RouteTypeAerialLift:
		return "Aerial Lift"
	case RouteTypeFunicular:
		return "Funicular"
	default:
		return "Unknown"
	}
}

// MarshalCSV converts this enum into a string for CSV writing.
func (rt *RouteType) MarshalCSV() (string, error) {
	return fmt.Sprintf("%d", rt), nil
}

// UnmarshalCSV attempts to convert a string value from a CSV file into the enum value.
func (rt *RouteType) UnmarshalCSV(csv string) error {
	val, err := strconv.ParseInt(strings.TrimSpace(csv), 10, 32)
	if err != nil {
		return err
	}

	*rt = RouteType(val)
	return nil
}

// Route represents a logical run of a vehicle.
type Route struct {
	ID          string    `csv:"route_id"`
	AgencyID    string    `csv:"agency_id"`
	ShortName   string    `csv:"route_short_name"`
	LongName    string    `csv:"route_long_name"`
	Description string    `csv:"route_desc"`
	Type        RouteType `csv:"route_type"`
	URL         string    `csv:"route_url"`
	Color       string    `csv:"route_color"`
	TextColor   string    `csv:"route_text_color"`
	SortOrder   CSVInt    `csv:"route_sort_order"`
}
