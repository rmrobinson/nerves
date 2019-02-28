package noaa

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rmrobinson/nerves/services/weather"
	"go.uber.org/zap"
)

// Station represents a NOAA station location
type Station struct {
	url   string
	title string

	latitude  float64
	longitude float64

	logger *zap.Logger

	currentReport *weather.WeatherReport
	forecast      []*weather.WeatherForecast
	lastRefreshed time.Time
}

// NewStation creates a new station.
func NewStation(logger *zap.Logger, url string, title string, latitude float64, longitude float64) *Station {
	return &Station{
		url:       url,
		title:     title,
		latitude:  latitude,
		longitude: longitude,
		logger:    logger,
	}
}

func (s *Station) shouldRefresh() bool {
	return time.Now().Add(refreshFrequency * -1).After(s.lastRefreshed)
}

func (s *Station) refresh(ctx context.Context) error {
	feature, err := s.getFeature(ctx)
	if err != nil {
		s.logger.Warn("error getting feature",
			zap.Error(err),
		)
		return err
	}

	report, forecast, err := s.parseFeature(feature)
	if err != nil {
		s.logger.Warn("error parsing feature",
			zap.Error(err),
		)
		return err
	}

	s.currentReport = report
	s.forecast = forecast
	s.lastRefreshed = time.Now()

	s.logger.Debug("refreshed station",
		zap.String("station_title", s.title),
	)

	return nil
}

func (s *Station) getFeature(ctx context.Context) (*feature, error) {
	req, err := http.NewRequest(http.MethodGet, s.url, nil)
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

	feature := &feature{
		logger: s.logger,
	}

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

	return feature, nil
}

func (s *Station) parseFeature(f *feature) (*weather.WeatherReport, []*weather.WeatherForecast, error) {
	windSpeed := f.getCurrentFloatFromProperty("windSpeed")

	report := &weather.WeatherReport{
		Conditions: &weather.WeatherCondition{
			Temperature: f.getCurrentFloatFromProperty("temperature"),
			DewPoint:    f.getCurrentFloatFromProperty("dewpoint"),
			Humidity:    f.getCurrentIntFromProperty("relativeHumidity"),
			WindSpeed:   int32(windSpeed),
		},
	}
	var forecasts []*weather.WeatherForecast

	return report, forecasts, nil
}

func (f *feature) getCurrentFloatFromProperty(propName string) float32 {
	prop, ok := f.Properties[propName]
	if !ok {
		f.logger.Info("error, property is unset")
		return 0
	}

	property := &propertyFloat{}
	err := json.Unmarshal(*prop, property)
	if err != nil {
		f.logger.Info("error unmarshaling property",
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

func (f *feature) getCurrentIntFromProperty(propName string) int32 {
	prop, ok := f.Properties[propName]
	if !ok {
		f.logger.Info("error, property is unset")
		return 0
	}

	property := &propertyInt{}
	err := json.Unmarshal(*prop, property)
	if err != nil {
		f.logger.Info("error unmarshaling property",
			zap.Error(err),
		)
		return 0
	}

	val := property.Values[0].Value
	return int32(val)
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
	ID         interface{}                 `json:"id,omitempty"`
	Type       string                      `json:"type"`
	Properties map[string]*json.RawMessage `json:"properties"`

	logger *zap.Logger
}
