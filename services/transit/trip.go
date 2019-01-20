package transit

import (
	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

// Trip represents a single instance of a Route, with a specified set of scheduled arrivals.
type Trip struct {
	*gtfs.Trip

	route *Route
	stops []*Arrival
}

// Plan is the set of stops that this trip will visit, sorted by Sequence.
func (t *Trip) Plan() []*Arrival {
	return t.stops
}
