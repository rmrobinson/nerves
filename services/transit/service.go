package transit

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/rmrobinson/nerves/lib/geoset"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrStopNotFound is returned if the queried stop isn't found
	ErrStopNotFound = status.New(codes.NotFound, "stop not found")
	// ErrStopCutoffInvalid is returned if the stop cutoff is invalid
	ErrStopCutoffInvalid = status.New(codes.InvalidArgument, "stop cutoff not a valid time")
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

// GetStopArrivals returns the arrival info for the specified stop.
func (s *Service) GetStopArrivals(ctx context.Context, req *GetStopArrivalsRequest) (*GetStopArrivalsResponse, error) {
	if req.Location == nil && len(req.StopCode) < 1 {
		return nil, ErrStopNotFound.Err()
	}

	var stop *stopDetails
	if req.Location != nil {
		stop = s.stops.Closest(req.Location.Latitude, req.Location.Longitude).(*stopDetails)
	} else {
		for _, feed := range s.feeds {
			if feedStop, ok := feed.stops[req.StopCode]; ok {
				stop = feedStop
				break
			}
		}

		if stop == nil {
			return nil, ErrStopNotFound.Err()
		}
	}

	var cutoff time.Time
	if req.ExcludeArrivalsBefore == nil {
		cutoff = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	} else {
		var err error
		cutoff, err = ptypes.Timestamp(req.ExcludeArrivalsBefore)
		if err != nil {
			return nil, ErrStopCutoffInvalid.Err()
		}
		cutoff = cutoff.In(time.Now().Location())
	}

	arrivals := stop.arrivalsForDay(cutoff)

	resp := &GetStopArrivalsResponse{
		Stop: &Stop{
			Id:        stop.ID,
			Code:      stop.Code,
			Name:      stop.Name,
			Latitude:  float64(stop.Latitude),
			Longitude: float64(stop.Longitude),
		},
	}

	for _, arrival := range arrivals {
		a := &Arrival{
			ScheduledArrivalTime:   arrival.ArrivalTime.String(),
			ScheduledDepartureTime: arrival.DepartureTime.String(),
			RouteId:                arrival.RouteID(),
			Headsign:               arrival.VehicleHeadsign(),
		}

		resp.Arrivals = append(resp.Arrivals, a)
	}

	return resp, nil
}
