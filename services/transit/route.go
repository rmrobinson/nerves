package transit

import (
	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

// routeDetails is a logical unit of service that visits a collection of stops.
// The physical instantiation of a route is a trip.
type routeDetails struct {
	*gtfs.Route

	trips []*tripDetails
}

// Trips returns the set of trips that are following this route.
func (r *routeDetails) Trips() []*tripDetails {
	return r.trips
}
