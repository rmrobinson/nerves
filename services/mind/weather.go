package mind

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/rmrobinson/nerves/services/weather"
	"go.uber.org/zap"
)

const (
	weatherPrefix = "whats the weather"
)

// Weather is a weather-request handler
type Weather struct {
	logger *zap.Logger

	client weather.WeatherServiceClient

	currLatitude float64
	currLongitude float64
}

// NewWeather creates a new weather handler
func NewWeather(logger *zap.Logger, client weather.WeatherServiceClient, lat float64, lon float64) *Weather {
	return &Weather{
		logger: logger,
		client: client,
		currLatitude: lat,
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
	if !strings.HasPrefix(content, weatherPrefix) {
		return nil, ErrStatementNotHandled.Err()
	}

	report, err := w.client.GetCurrentReport(context.Background(), &weather.GetCurrentReportRequest{
		Latitude:  w.currLatitude,
		Longitude: w.currLongitude,
	})
	if err != nil {
		w.logger.Warn("unable to get weather",
			zap.Error(err),
		)

		return statementFromText("Can't get the weather right now :("), nil
	} else if report == nil {
		w.logger.Warn("unable to get weather (empty report)")

		return statementFromText("Not sure what the weather is right now"), nil
	}

	w.logger.Debug("processed message",
		zap.String("content", content),
	)

	respStmt := statementFromConditions(report.Report.Conditions)

	return respStmt, nil
}

func statementFromConditions(conditions *weather.WeatherCondition) *Statement {
	condText := fmt.Sprintf("It is currently %d °C and %s", int(conditions.Temperature), strings.ToLower(conditions.Summary))
	return statementFromText(condText)
}

func statementFromText(content string) *Statement {
	return &Statement{
		MimeType: "text/plain",
		Content: []byte(content),
		CreateAt: ptypes.TimestampNow(),
	}
}
