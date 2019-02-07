package mind

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/rmrobinson/nerves/services/transit"
	"go.uber.org/zap"
)

const (
	transitPrefix = "whens the bus coming"
)

// Transit is a transit-request handler
type Transit struct {
	logger *zap.Logger

	client transit.TransitServiceClient
}

// NewTransit creates a new transit handler
func NewTransit(logger *zap.Logger, client transit.TransitServiceClient) *Transit {
	return &Transit{
		logger: logger,
		client: client,
	}
}

// ProcessStatement implements the handler interface. Logs and returns the statement.
func (t *Transit) ProcessStatement(ctx context.Context, stmt *Statement) (*Statement, error) {
	if stmt.MimeType != "text/plain" {
		return nil, ErrStatementNotHandled.Err()
	}

	content := string(stmt.Content)
	content = strings.ToLower(content)
	if !strings.HasPrefix(content, transitPrefix) {
		return nil, ErrStatementNotHandled.Err()
	}

	stopID := strings.TrimPrefix(content, transitPrefix)
	stopID = strings.TrimPrefix(stopID, " to stop ")

	return t.getTransitStop(stopID), nil
}

func (t *Transit) getTransitStop(stopID string) *Statement {
	resp, err := t.client.GetStopArrivals(context.Background(), &transit.GetStopArrivalsRequest{
		StopCode:              stopID,
		ExcludeArrivalsBefore: ptypes.TimestampNow(),
	})
	if err != nil {
		t.logger.Warn("unable to get transit arrivals",
			zap.String("stop_code", stopID),
			zap.Error(err),
		)

		return statementFromText("Can't get the stop schedule right now :(")
	} else if resp == nil {
		t.logger.Warn("unable to get stop schedule (empty response)")

		return statementFromText("Not sure what the upcoming stop arrivals are right now")
	}

	return statementFromArrivals(resp.Stop, resp.Arrivals)
}

func statementFromArrivals(stop *transit.Stop, records []*transit.Arrival) *Statement {
	transitText := "Stop " + stop.Name + " is expecting the following arrivals: ```"
	for idx, record := range records {
		if idx > 10 {
			break
		}

		var err error
		var scheduledArrivalTime time.Time
		var arrivalTime time.Time

		scheduledArrivalTime, err = ptypes.Timestamp(record.ScheduledArrivalTime)

		if record.EstimatedArrivalTime != nil {
			arrivalTime, err = ptypes.Timestamp(record.EstimatedArrivalTime)
		} else {
			arrivalTime = scheduledArrivalTime
		}

		if err != nil {
			transitText += "Err: " + err.Error() + "\n"
			continue
		}

		scheduledArrivalTime = scheduledArrivalTime.In(time.Now().Location())
		arrivalTime = arrivalTime.In(time.Now().Location())

		transitText += fmt.Sprintf("%s %s arrival scheduled for %s",
			record.RouteId,
			record.Headsign,
			scheduledArrivalTime.Format("15:04:05"),
		)

		if record.EstimatedArrivalTime != nil {
			transitText += "; estimated " + getTextForTimeDiff(time.Now(), arrivalTime)
		}

		transitText += "\n"
	}

	transitText += "```"

	return statementFromText(transitText)
}

func getTextForTimeDiff(t1 time.Time, t2 time.Time) string {
	if t1.After(t2) {
		duration := t1.Sub(t2)
		return fmt.Sprintf("%d mins ago", int64(duration.Minutes()))
	}

	duration := t2.Sub(t1)

	if duration < 1 {
		return "now"
	}

	return fmt.Sprintf("in %d mins", int64(duration.Minutes()))
}
