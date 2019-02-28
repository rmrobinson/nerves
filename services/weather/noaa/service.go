package noaa

import (
	"context"
	"errors"
	"time"

	"github.com/rmrobinson/nerves/lib/geoset"
	"github.com/rmrobinson/nerves/services/weather"
	"go.uber.org/zap"
)

var (
	// ErrInvalidDate is returned if an invalid date qualifier is supplied.
	ErrInvalidDate = errors.New("invalid date supplied")
	// ErrLocationNotFound is returned if the supplied lat/lon value can't be found.
	ErrLocationNotFound = errors.New("location not found")

	refreshFrequency = time.Minute * 30
)

// Service represents the API calls to retrieve weather info from the NOAA API
type Service struct {
	logger *zap.Logger

	stations *geoset.GeoSet
}

// NewService creates a new instance of the noaa service.
func NewService(logger *zap.Logger) *Service {
	return &Service{
		logger:   logger,
		stations: geoset.NewGeoSet(),
	}
}

// AddStation registers the supplied station in the service.
func (f *Service) AddStation(station *Station) {
	f.stations.Add(station.latitude, station.longitude, station)
}

// GetReport retrieves the current weather report of the supplied location. The data returned may not be slightly
// out of date as the service caches results for 30 minutes.
func (f *Service) GetReport(ctx context.Context, latitude float64, longitude float64) (*weather.WeatherReport, error) {
	s := f.stations.Closest(latitude, longitude).(*Station)
	if s == nil {
		return nil, ErrLocationNotFound
	}

	if !s.shouldRefresh() {
		return s.currentReport, nil
	}

	err := s.refresh(ctx)
	if err != nil {
		return nil, err
	}

	return s.currentReport, nil
}
