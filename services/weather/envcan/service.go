package envcan

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/rmrobinson/nerves/services/weather"
	"go.uber.org/zap"
)

var (
	// ErrInvalidDate is returned if an invalid date qualifier is supplied.
	ErrInvalidDate   = errors.New("invalid date supplied")
	refreshFrequency = time.Minute * 30
)

// Service is a very basic weather feed provided by Environment Canada
type Service struct {
	logger *zap.Logger

	api    *weather.API
	active []*Station
}

// NewService creates a new weather feed for Environment Canada.
func NewService(logger *zap.Logger, api *weather.API, weatherStationFile string) (*Service, error) {
	svc := &Service{
		logger: logger,
		api:    api,
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
		svc.api.RegisterStation(station, station.Latitude, station.Longitude)
	}
	if err := scanner.Err(); err != nil {
		logger.Warn("error scanning station file",
			zap.Error(err),
		)
		return nil, err
	}

	return svc, nil
}
