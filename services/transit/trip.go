package transit

import (
	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

// tripDetails represents a single instance of a route, with a specified set of scheduled arrivals.
type tripDetails struct {
	*gtfs.Trip

	route *routeDetails
	stops []*arrivalDetails
}

// Plan is the set of stops that this trip will visit, sorted by Sequence.
func (t *tripDetails) Plan() []*arrivalDetails {
	return t.stops
}
