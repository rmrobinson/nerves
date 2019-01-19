package transit

import (
	"archive/zip"
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/golang/protobuf/proto"
	"github.com/rmrobinson/nerves/lib/geoset"
	"go.uber.org/zap"
)

// Service is a really simple service that retrieves a transit feed
type Service struct {
	logger *zap.Logger

	agencies  []*Agency
	trips     []*Trip
	stopTimes []*StopTime
	calendar  []*Calendar

	stops *geoset.GeoSet
}

// NewService creates a new service
func NewService(logger *zap.Logger) *Service {
	return &Service{
		logger: logger,
		stops:  geoset.NewGeoSet(),
	}
}

// GetRealtimeFeed retrieves the GTFS realtime data.
func (s *Service) GetRealtimeFeed(ctx context.Context, path string) (*FeedMessage, error) {
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		s.logger.Warn("error creating new request",
			zap.Error(err),
		)
		return nil, err
	}

	req.WithContext(ctx)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Warn("error performing request",
			zap.Error(err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Info("received non-OK response",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.logger.Warn("error reading body",
			zap.Error(err),
		)
		return nil, err
	}

	feed := &FeedMessage{}
	err = proto.Unmarshal(body, feed)
	if err != nil {
		s.logger.Warn("error unmarshaling body",
			zap.Error(err),
		)
		return nil, err
	}

	return feed, nil
}

// GetFeed retrieves the GTFS feed data. It currently saves the stops.
func (s *Service) GetFeed(ctx context.Context, path string) error {
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		s.logger.Warn("error creating new request",
			zap.Error(err),
		)
		return err
	}

	req.WithContext(ctx)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Warn("error performing request",
			zap.Error(err),
		)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Info("received non-OK response",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.logger.Warn("error reading body",
			zap.Error(err),
		)
		return err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		s.logger.Warn("error reading zip file",
			zap.Error(err),
		)
		return err
	}

	// Read all the files from zip archive
	for _, zipFile := range zipReader.File {
		s.logger.Debug("handling file",
			zap.String("file_name", zipFile.Name),
		)

		switch zipFile.Name {
		case "stops.txt":
			var stops []*Stop
			err = parseCSVFile(zipFile, &stops)
			if err != nil {
				s.logger.Warn("error reading stops file",
					zap.String("file_name", zipFile.Name),
					zap.Error(err),
				)
				continue
			}

			for _, stop := range stops {
				s.stops.Add(stop.Latitude.float64, stop.Longitude.float64, stop)
			}
			s.logger.Debug("parsed stops")
		case "agency.txt":
			err = parseCSVFile(zipFile, &s.agencies)
			if err != nil {
				s.logger.Warn("error reading stops file",
					zap.String("file_name", zipFile.Name),
					zap.Error(err),
				)
				continue
			}
			s.logger.Debug("parsed agencies")
		case "trips.txt":
			err = parseCSVFile(zipFile, &s.trips)
			if err != nil {
				s.logger.Warn("error reading trips file",
					zap.String("file_name", zipFile.Name),
					zap.Error(err),
				)
				continue
			}
			s.logger.Debug("parsed trips")
		case "stop_times.txt":
			err = parseCSVFile(zipFile, &s.stopTimes)
			if err != nil {
				s.logger.Warn("error reading stop times file",
					zap.String("file_name", zipFile.Name),
					zap.Error(err),
				)
				continue
			}
			s.logger.Debug("parsed stop times")
		case "calendar.txt":
			err = parseCSVFile(zipFile, &s.calendar)
			if err != nil {
				s.logger.Warn("error reading calendar file",
					zap.String("file_name", zipFile.Name),
					zap.Error(err),
				)
				continue
			}
			s.logger.Debug("parsed calendars")
		default:
			if !strings.HasSuffix(zipFile.Name, "stops.txt") {
				s.logger.Debug("skipping file",
					zap.String("file_name", zipFile.Name),
				)
			}
		}
	}
	return nil
}

// GetAgencies returns the set of agencies in this service.
func (s *Service) GetAgencies() []*Agency {
	return s.agencies
}

// GetClosestStop returns the closest stop to the supplied location
func (s *Service) GetClosestStop(lat float64, lon float64) *Stop {
	return s.stops.Closest(lat, lon).(*Stop)
}

func parseCSVFile(zf *zip.File, out interface{}) error {
	f, err := zf.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	err = gocsv.Unmarshal(f, out)
	if err != nil {
		return err
	}

	return nil
}
