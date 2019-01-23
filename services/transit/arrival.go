package transit

import (
	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

// Arrival is the occurrence of a trip visiting a stop.
type arrivalInfo struct {
	*gtfs.StopTime

	trip *tripInfo
	stop *stopInfo
}

// RouteID is the ID of the route the trip making the arrival is from.
func (a *arrivalInfo) RouteID() string {
	return a.trip.RouteID
}

// Stop returns the stop that this arrival is occurring at.
func (a *arrivalInfo) Stop() *stopInfo {
	return a.stop
}

// VehicleType returns the type of vehicle that will be making this arrival.
func (a *arrivalInfo) VehicleType() gtfs.RouteType {
	return a.trip.route.Type
}

// VehicleHeadsign is the text that the vehicle will be displaying.
func (a *arrivalInfo) VehicleHeadsign() string {
	if len(a.Headsign) > 0 {
		return a.Headsign
	} else if len(a.trip.Headsign) > 0 {
		return a.trip.Headsign
	} else {
		return a.trip.route.LongName
	}
}
