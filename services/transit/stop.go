package transit

import (
	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

// Stop represents a single stop that one or more route trips may visit.
type Stop struct {
	*gtfs.Stop

	arrivals []*Arrival
}

// Arrivals is the set of trips that will visit this location, sorted by arrival time.
func (s *Stop) Arrivals() []*Arrival {
	return s.arrivals
}
