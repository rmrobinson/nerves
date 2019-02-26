package weather

import (
	"context"
)

// Service defines methods to be implemented by a weather service.
type Service interface {
	GetReport(context context.Context, latitude float64, longitude float64) (*WeatherReport, error)
	GetForecast(context context.Context, latitude float64, longitude float64) ([]*WeatherForecast, error)
}

// API is an implementation of the WeatherService server.
type API struct {
	service Service
}

// NewAPI creates a new weather service server.
func NewAPI(feed Service) *API {
	return &API{
		service: feed,
	}
}

// GetCurrentReport gets a weather report
func (api *API) GetCurrentReport(ctx context.Context, req *GetCurrentReportRequest) (*GetCurrentReportResponse, error) {
	report, err := api.service.GetReport(ctx, req.Latitude, req.Longitude)
	if err != nil {
		return nil, err
	}

	return &GetCurrentReportResponse{
		Report: report,
	}, nil
}

// GetForecast gets a weather forecast.
func (api *API) GetForecast(ctx context.Context, req *GetForecastRequest) (*GetForecastResponse, error) {
	forecast, err := api.service.GetForecast(ctx, req.Latitude, req.Longitude)
	if err != nil {
		return nil, err
	}
	return &GetForecastResponse{
		ForecastRecords: forecast,
}, nil
}
