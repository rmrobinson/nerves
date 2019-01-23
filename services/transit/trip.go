package transit

import (
	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

// Trip represents a single instance of a Route, with a specified set of scheduled arrivals.
type tripInfo struct {
	*gtfs.Trip

	route *routeInfo
	stops []*arrivalInfo
}

// Plan is the set of stops that this trip will visit, sorted by Sequence.
func (t *tripInfo) Plan() []*arrivalInfo {
	return t.stops
}
