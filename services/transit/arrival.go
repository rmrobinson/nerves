package transit

import (
	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

// Arrival is the occurence of a trip visiting a stop.
type Arrival struct {
	*gtfs.StopTime

	trip *Trip
	stop *Stop
}

// RouteID is the ID of the route the trip making the arrival is from.
func (a *Arrival) RouteID() string {
	return a.trip.RouteID
}

// Stop returns the stop that this arrival is occurring at.
func (a *Arrival) Stop() *Stop {
	return a.stop
}

// VehicleType returns the type of vehicle that will be making this arrival.
func (a *Arrival) VehicleType() gtfs.RouteType {
	return a.trip.route.Type
}

// VehicleHeadsign is the text that the vehicle will be displaying.
func (a *Arrival) VehicleHeadsign() string {
	if len(a.Headsign) > 0 {
		return a.Headsign
	} else if len(a.trip.Headsign) > 0 {
		return a.trip.Headsign
	} else {
		return a.trip.route.LongName
	}
}
