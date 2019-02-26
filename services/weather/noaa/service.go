package noaa

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/paulmach/go.geojson"
	"github.com/rmrobinson/nerves/services/weather"
	"go.uber.org/zap"
)

type Service struct {
	URL string
	logger *zap.Logger
}

func NewService(logger *zap.Logger, url string) *Service {
	return &Service{
		URL: url,
		logger: logger,
	}
}

type propertyValueFloat struct {
	ValidTime string
	Value float64
}

func (p *propertyValueFloat) fromMap(fieldsInterface interface{}) {
	fields := fieldsInterface.(map[string]interface{})
	p.ValidTime = fields["validTime"].(string)
	p.Value = fields["value"].(float64)
}

type propertyFloat struct {
	SourceUnit string `json:"sourceUnit"`
	UnitOfMeasure string `json:"uom"`
	Values []propertyValueFloat `json:"values"`
}

func (p *propertyFloat) fromMap(fields map[string]interface{}) {
	p.SourceUnit = fields["sourceUnit"].(string)
	p.UnitOfMeasure = fields["uom"].(string)
	rawValues := fields["values"].([]interface{})
	for _, rawValue := range rawValues {
		pv := propertyValueFloat{}
		pv.fromMap(rawValue)
		p.Values = append(p.Values, pv)
	}
}

func (s *Service) GetCurrentReport(ctx context.Context) (*weather.WeatherCondition, error){
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

	feature := geojson.NewFeature(nil)

	err = json.NewDecoder(resp.Body).Decode(feature)
	if err != nil {
		s.logger.Info("error unmarshaling feature",
			zap.Error(err),
		)
		return nil, err
	}

	temperatureProperty := propertyFloat{}
	temperatureProperty.fromMap(feature.Properties["temperature"].(map[string]interface{}))

	temperature := temperatureProperty.Values[0].Value
	if temperatureProperty.UnitOfMeasure == "unit:degF" {
		temperature = (temperature - 32) * 5/9
	}

	return &weather.WeatherCondition{
		Temperature: float32(temperature),
	}, nil
}
