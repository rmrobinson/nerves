package weather

import (
	"context"
)

// API is an implementation of the WeatherService server.
type API struct {
	ecf *EnvironmentCanadaFeed
}

// NewAPI creates a new weather service server.
func NewAPI(ecf *EnvironmentCanadaFeed) *API {
	return &API{
		ecf: ecf,
	}
}

// GetCurrentReport gets a weather report
func (api *API) GetCurrentReport(ctx context.Context, req *GetCurrentReportRequest) (*WeatherReport, error) {
	return api.ecf.report, nil
}

// GetForecast gets a weather forecast.
func (api *API) GetForecast(ctx context.Context, req *GetForecastRequest) (*GetForecastResponse, error) {
	return &GetForecastResponse{
		ForecastRecords: api.ecf.forecast,
	}, nil
}
