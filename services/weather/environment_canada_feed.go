package weather

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
)

// EnvironmentCanadaFeed is a very basic weather feed provided by Environment Canada
type EnvironmentCanadaFeed struct {
	logger *zap.Logger
	path string

	report *WeatherReport
	forecast []*WeatherForecast
}

// NewEnvironmentCanadaFeed creates a new weather feed for Environment Canada.
func NewEnvironmentCanadaFeed(logger *zap.Logger, path string) *EnvironmentCanadaFeed {
	return &EnvironmentCanadaFeed{
		logger: logger,
		path: path,
	}
}

// Run begins a loop that will poll EC for changes.
func (ecf *EnvironmentCanadaFeed) Run(ctx context.Context) {
	ecf.logger.Info("run started")
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <- ctx.Done():
			ecf.logger.Info("context closed, completing run")
			ticker.Stop()
			return
		case <- ticker.C:
			ecf.logger.Debug("refreshing data")
			feed, err := ecf.getFeed(ctx)
			if err != nil {
				continue
			}

			report, forecast, err := ecf.parseFeed(feed)
			if err != nil {
				ecf.logger.Warn("error parsing feed",
					zap.Error(err),
				)
				continue
			}

			ecf.report = report
			ecf.forecast = forecast

			ecf.logger.Debug("report",
				zap.String("value", ecf.report.String()),
			)
		}
	}
}

func (ecf *EnvironmentCanadaFeed) getFeed(ctx context.Context) (*gofeed.Feed, error) {
	req, err := http.NewRequest(http.MethodGet, ecf.path, nil)
	if err != nil {
		ecf.logger.Warn("error creating new request",
			zap.Error(err),
		)
		return nil, err
	}

	req.WithContext(ctx)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		ecf.logger.Warn("error performing request",
			zap.Error(err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ecf.logger.Info("received non-OK response",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, nil
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		ecf.logger.Warn("error parsing feed",
			zap.Error(err),
		)
		return nil, err
	}

	return feed, nil
}

func (ecf *EnvironmentCanadaFeed) parseFeed(feed *gofeed.Feed) (*WeatherReport, []*WeatherForecast, error) {
	report := &WeatherReport{
		Conditions: &WeatherCondition{},
	}
	var forecasts []*WeatherForecast

	for _, item := range feed.Items {
		for _, category := range item.Categories {
			if category == "Current Conditions" {
				report.ObservedAt = ptypes.TimestampNow()
				report.ObservationId = item.GUID
				report.CreatedAt, _ = ptypes.TimestampProto(*item.PublishedParsed)
				report.UpdatedAt, _ = ptypes.TimestampProto(*item.UpdatedParsed)

				report.Conditions = currentConditionsToCondition(item.Description)
			} else if category == "Weather Forecasts" {
				forecast := &WeatherForecast{
					ForecastId: item.GUID,

				}

				forecasts = append(forecasts, forecast)
			}
		}
	}

	return report, forecasts, nil
}

func currentConditionsToCondition(cc string) *WeatherCondition {
	cond := &WeatherCondition{}

	records := strings.Split(cc, "<br/>\n")
	for _, record := range records {
		record = strings.Replace(record, "<b>", "", -1)
		record = strings.Replace(record, "</b>", "", -1)
		recordParts := strings.Split(record, ":")
		switch recordParts[0] {
		case "Condition":
			cond.Summary = strings.TrimSpace(recordParts[1])
		case "Temperature":
			str := strings.TrimSpace(recordParts[1])
			str = strings.Replace(str, "&deg;C", "", -1)

			val, _ := strconv.ParseFloat(str, 32)
			cond.Temperature = float32(val)
		case "Wind Chill":
			str := strings.TrimSpace(recordParts[1])
			str = strings.Replace(str, "&deg;C", "", -1)

			val, _ := strconv.ParseFloat(str, 32)
			cond.WindChill = float32(val)
		case "Dewpoint":
			str := strings.TrimSpace(recordParts[1])
			str = strings.Replace(str, "&deg;C", "", -1)

			val, _ := strconv.ParseFloat(str, 32)
			cond.DewPoint = float32(val)
		case "Pressure":
			str := strings.TrimSpace(recordParts[1])
			str = strings.Replace(str, " kPa", "", -1)

			val, _ := strconv.ParseFloat(str, 32)
			cond.Pressure = float32(val)
		case "Visibility":
			str := strings.TrimSpace(recordParts[1])
			str = strings.Replace(str, " km", "", -1)

			val, _ := strconv.ParseFloat(str, 32)
			cond.Visibility = int32(val)
		case "Humidity":
			str := strings.TrimSpace(recordParts[1])
			str = strings.Replace(str, " %", "", -1)

			val, _ := strconv.ParseInt(str, 10, 32)
			cond.Humidity = int32(val)
		case "Wind":
			str := strings.TrimSpace(recordParts[1])
			str = strings.Replace(str, " km/h", "", -1)
			str = str[3:]

			val, _ := strconv.ParseInt(str, 10, 32)
			cond.WindSpeed = int32(val)
		}
	}

	return cond
}
