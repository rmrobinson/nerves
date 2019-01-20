package transit

import (
	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

// Route is a logical unit of service that visits a collection of stops.
// The physical instantiation of a route is a trip.
type Route struct {
	*gtfs.Route

	trips []*Trip
}

// Trips returns the set of trips that are following this route.
func (r *Route) Trips() []*Trip {
	return r.trips
}
