package envcan

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
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

// Service is a very basic weather feed provided by Environment Canada
type Service struct {
	logger *zap.Logger

	stations *geoset.GeoSet

	active []*Station
}

// NewService creates a new weather feed for Environment Canada.
func NewService(logger *zap.Logger, weatherStationFile string) (*Service, error) {
	feed := &Service{
		logger:   logger,
		stations: geoset.NewGeoSet(),
	}

	stationFile, err := os.Open(weatherStationFile)
	if err != nil {
		logger.Error("unable to open weather station file",
			zap.String("file_name", weatherStationFile),
			zap.Error(err),
		)
		return nil, err
	}
	defer stationFile.Close()

	scanner := bufio.NewScanner(stationFile)
	for scanner.Scan() {
		station := &Station{}
		err = json.NewDecoder(strings.NewReader(scanner.Text())).Decode(&station)
		if err != nil {
			logger.Info("error decoding station",
				zap.Error(err),
			)
			continue
		}

		station.logger = logger.With(zap.String("title", station.Title))
		feed.stations.Add(station.Latitude, station.Longitude, station)
	}
	if err := scanner.Err(); err != nil {
		logger.Warn("error scanning station file",
			zap.Error(err),
		)
		return nil, err
	}

	return feed, nil
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

// GetForecast retrieves the current weather forecast of the supplied location. The data returned may not be slightly
// out of date as the service caches results for 30 minutes.
func (f *Service) GetForecast(ctx context.Context, latitude float64, longitude float64) ([]*weather.WeatherForecast, error) {
	s := f.stations.Closest(latitude, longitude).(*Station)
	if s == nil {
		return nil, ErrLocationNotFound
	}

	if !s.shouldRefresh() {
		return s.forecast, nil
	}

	err := s.refresh(ctx)
	if err != nil {
		return nil, err
	}

	return s.forecast, nil
}

// Run begins a loop that will poll EC for changes.
func (f *Service) Run(ctx context.Context) {
	f.logger.Info("run started")
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ctx.Done():
			f.logger.Info("context closed, completing run")
			ticker.Stop()
			return
		case <-ticker.C:
			f.logger.Debug("refreshing stations")
			for _, station := range f.active {
				if station.shouldRefresh() {
					err := station.refresh(ctx)
					if err != nil {
						f.logger.Info("error refreshing station",
							zap.String("station", station.Title),
							zap.Error(err),
						)
					}
				}
			}
		}
	}
}
