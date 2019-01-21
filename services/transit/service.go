package transit

import (
	"github.com/rmrobinson/nerves/lib/geoset"
	"go.uber.org/zap"
)

// Service is a really simple service that retrieves a transit feed
type Service struct {
	logger *zap.Logger

	stops *geoset.GeoSet

	feeds []*Feed
}

// NewService creates a new service
func NewService(logger *zap.Logger) *Service {
	return &Service{
		logger: logger,
		stops:  geoset.NewGeoSet(),
	}
}

// AddFeed adds a GTFS dataset to this transit instance.
func (s *Service) AddFeed(feed *Feed) {
	s.feeds = append(s.feeds, feed)
	for _, stop := range feed.stops {
		s.stops.Add(float64(stop.Latitude), float64(stop.Longitude), stop)
	}
}

// StopClosestTo retrieves the closest stop to the specified lat/lon.
func (s *Service) StopClosestTo(lat float64, lon float64) *Stop {
	return s.stops.Closest(lat, lon).(*Stop)
}

// StopByID returns the first stop with the specified ID.
func (s *Service) StopByID(id string) *Stop {
	for _, feed := range s.feeds {
		if stop, ok := feed.stops[id]; ok {
			return stop
		}
	}

	return nil
}
