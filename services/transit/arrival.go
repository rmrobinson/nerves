package transit

import (
	"time"

	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

// arrivalDetails is the occurrence of a trip visiting a stop.
type arrivalDetails struct {
	*gtfs.StopTime

	arrivalTime time.Time
	departureTime time.Time

	estimatedArrivalTime *time.Time
	estimatedDepartureTime *time.Time

	trip *tripDetails
	stop *stopDetails
}

func gtfsTimeToCalendarTime(in gtfs.CSVTime, loc *time.Location) time.Time {
	now := time.Now()

	day := now.Day()
	hour := in.Hour
	minute := in.Minute
	second := in.Second

	if hour > 24 {
		hour %= 24
		day++
	}

	return time.Date(now.Year(), now.Month(), day, hour, minute, second, 0, loc)
}

func newArrivalDetails(st *gtfs.StopTime, loc *time.Location) *arrivalDetails {
	return &arrivalDetails{
		StopTime: st,
		arrivalTime: gtfsTimeToCalendarTime(st.ArrivalTime, loc),
		departureTime: gtfsTimeToCalendarTime(st.DepartureTime, loc),
	}
}

// RouteID is the ID of the route the trip making the arrival is from.
func (a *arrivalDetails) RouteID() string {
	return a.trip.RouteID
}

// Stop returns the stop that this arrival is occurring at.
func (a *arrivalDetails) Stop() *stopDetails {
	return a.stop
}

// VehicleType returns the type of vehicle that will be making this arrival.
func (a *arrivalDetails) VehicleType() gtfs.RouteType {
	return a.trip.route.Type
}

// VehicleHeadsign is the text that the vehicle will be displaying.
func (a *arrivalDetails) VehicleHeadsign() string {
	if len(a.Headsign) > 0 {
		return a.Headsign
	} else if len(a.trip.Headsign) > 0 {
		return a.trip.Headsign
	} else {
		return a.trip.route.LongName
	}
}
