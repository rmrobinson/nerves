package mind

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/rmrobinson/nerves/services/weather"
	"go.uber.org/zap"
)

const (
	weatherRegex = "what('| i)?s the weather( forecast)?"
)

// Weather is a weather-request handler
type Weather struct {
	logger *zap.Logger

	client weather.WeatherServiceClient

	currLatitude  float64
	currLongitude float64
}

// NewWeather creates a new weather handler
func NewWeather(logger *zap.Logger, client weather.WeatherServiceClient, lat float64, lon float64) *Weather {
	return &Weather{
		logger:        logger,
		client:        client,
		currLatitude:  lat,
		currLongitude: lon,
	}
}

// ProcessStatement implements the handler interface. Logs and returns the statement.
func (w *Weather) ProcessStatement(ctx context.Context, stmt *Statement) (*Statement, error) {
	if stmt.MimeType != "text/plain" {
		return nil, ErrStatementNotHandled.Err()
	}

	content := string(stmt.Content)
	content = strings.ToLower(content)
	if ok, _ := regexp.MatchString(weatherRegex, content); !ok {
		return nil, ErrStatementNotHandled.Err()
	}

	var resp *Statement
	if strings.Contains(content, "forecast") {
		resp = w.getForecast()
	} else {
		resp = w.getConditions()
	}

	return resp, nil
}

func (w *Weather) getForecast() *Statement {
	report, err := w.client.GetForecast(context.Background(), &weather.GetForecastRequest{
		Latitude:  w.currLatitude,
		Longitude: w.currLongitude,
	})
	if err != nil {
		w.logger.Warn("unable to get weather forecast",
			zap.Error(err),
		)

		return statementFromText("Can't get the weather right now :(")
	} else if report == nil {
		w.logger.Warn("unable to get weather (empty report)")

		return statementFromText("Not sure what the weather is right now")
	}

	return statementFromForecast(report.ForecastRecords)
}

func (w *Weather) getConditions() *Statement {
	report, err := w.client.GetCurrentReport(context.Background(), &weather.GetCurrentReportRequest{
		Latitude:  w.currLatitude,
		Longitude: w.currLongitude,
	})
	if err != nil {
		w.logger.Warn("unable to get weather",
			zap.Error(err),
		)

		return statementFromText("Can't get the weather right now :(")
	} else if report == nil {
		w.logger.Warn("unable to get weather (empty report)")

		return statementFromText("Not sure what the weather is right now")
	}

	return statementFromConditions(report.Report.Conditions)
}

func statementFromForecast(forecast []*weather.WeatherForecast) *Statement {
	forecastText := "The current forecast is: ```"
	for _, record := range forecast {
		forecastedFor, _ := ptypes.Timestamp(record.ForecastedFor)

		// This gives us our 'low' temperature
		if forecastedFor.Hour() == 23 {
			forecastText += fmt.Sprintf("%s evening the low will be %2.f°C\n", forecastedFor.Format("Monday"), record.Conditions.Temperature)
		} else {
			forecastText += fmt.Sprintf("%s the high will be %2.f°C\n", forecastedFor.Format("Monday"), record.Conditions.Temperature)
		}
	}
	forecastText += "```"

	return statementFromText(forecastText)
}

func statementFromConditions(conditions *weather.WeatherCondition) *Statement {
	condText := fmt.Sprintf("It is currently %d °C and %s", int(conditions.Temperature), strings.ToLower(conditions.Summary))
	return statementFromText(condText)
}
