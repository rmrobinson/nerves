package noaa

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rmrobinson/nerves/services/weather"
	"go.uber.org/zap"
)

// Service represents the API calls to retrieve weather info from the NOAA API
type Service struct {
	URL    string
	logger *zap.Logger
}

// NewService creates a new instance of the noaa service.
func NewService(logger *zap.Logger, url string) *Service {
	return &Service{
		URL:    url,
		logger: logger,
	}
}

type propertyValueFloat struct {
	ValidTime string  `json:"validTime"`
	Value     float64 `json:"value"`
}

type propertyFloat struct {
	SourceUnit    string               `json:"sourceUnit"`
	UnitOfMeasure string               `json:"uom"`
	Values        []propertyValueFloat `json:"values"`
}

type propertyValueInt struct {
	ValidTime string `json:"validTime"`
	Value     int    `json:"value"`
}

type propertyInt struct {
	SourceUnit    string             `json:"sourceUnit"`
	UnitOfMeasure string             `json:"uom"`
	Values        []propertyValueInt `json:"values"`
}
type feature struct {
	ID          interface{}                 `json:"id,omitempty"`
	Type        string                      `json:"type"`
	Properties  map[string]*json.RawMessage `json:"properties"`
}

// GetCurrentReport gets the current weather report.
func (s *Service) GetCurrentReport(ctx context.Context) (*weather.WeatherCondition, error) {
	req, err := http.NewRequest(http.MethodGet, s.URL, nil)
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

	feature := &feature{}

	err = json.NewDecoder(resp.Body).Decode(feature)
	if err != nil {
		s.logger.Info("error unmarshaling feature",
			zap.Error(err),
		)
		return nil, err
	} else if feature.Type != "Feature" {
		s.logger.Info("unknown type detected",
			zap.String("type", feature.Type),
		)
		return nil, nil
	}

	windSpeed := s.getCurrentFloatFromProperty(feature.Properties["windSpeed"])
	return &weather.WeatherCondition{
		Temperature: s.getCurrentFloatFromProperty(feature.Properties["temperature"]),
		DewPoint:    s.getCurrentFloatFromProperty(feature.Properties["dewpoint"]),
		Humidity:    s.getCurrentIntFromProperty(feature.Properties["relativeHumidity"]),
		WindSpeed:   int32(windSpeed),
	}, nil
}

func (s *Service) getCurrentFloatFromProperty(prop *json.RawMessage) float32 {
	if prop == nil {
		s.logger.Info("error, property is nil")
		return 0
	}

	property := &propertyFloat{}
	err := json.Unmarshal(*prop, property)
	if err != nil {
		s.logger.Info("error unmarshaling property",
			zap.Error(err),
		)
		return 0
	}

	val := property.Values[0].Value
	if property.UnitOfMeasure == "unit:degF" {
		val = (val - 32) * 5 / 9
	}

	return float32(val)
}

func (s *Service) getCurrentIntFromProperty(prop *json.RawMessage) int32 {
	if prop == nil {
		s.logger.Info("error, property is nil")
		return 0
	}

	property := &propertyInt{}
	err := json.Unmarshal(*prop, property)
	if err != nil {
		s.logger.Info("error unmarshaling property",
			zap.Error(err),
		)
		return 0
	}

	val := property.Values[0].Value
	return int32(val)
}
