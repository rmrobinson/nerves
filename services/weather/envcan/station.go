package envcan

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/mmcdole/gofeed"
	"github.com/rmrobinson/nerves/services/weather"
	"go.uber.org/zap"
)

// Station contains the data about a single weather location reported on by Environment Canada
type Station struct {
	URL   string `json:"url"`
	Title string `json:"title"`

	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	SiteType         string  `json:"site_type"`
	SiteProvinceCode string  `json:"site_province_code"`

	logger *zap.Logger

	currentReport *weather.WeatherReport
	forecast      []*weather.WeatherForecast
	lastRefreshed time.Time
}

func (s *Station) shouldRefresh() bool {
	return time.Now().Add(refreshFrequency * -1).After(s.lastRefreshed)
}

func (s *Station) refresh(ctx context.Context) error {
	feed, err := s.getFeed(ctx)
	if err != nil {
		s.logger.Warn("error getting feed",
			zap.Error(err),
		)
		return err
	}

	report, forecast, err := s.parseFeed(feed)
	if err != nil {
		s.logger.Warn("error parsing feed",
			zap.Error(err),
		)
		return err
	}

	s.currentReport = report
	s.forecast = forecast
	s.lastRefreshed = time.Now()

	s.logger.Debug("refreshed station",
		zap.String("station_title", s.Title),
	)

	return nil
}

func (s *Station) getFeed(ctx context.Context) (*gofeed.Feed, error) {
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

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		s.logger.Warn("error parsing feed",
			zap.Error(err),
		)
		return nil, err
	}

	return feed, nil
}

func (s *Station) parseFeed(feed *gofeed.Feed) (*weather.WeatherReport, []*weather.WeatherForecast, error) {
	report := &weather.WeatherReport{
		Conditions: &weather.WeatherCondition{},
	}
	var forecasts []*weather.WeatherForecast

	for _, item := range feed.Items {
		for _, category := range item.Categories {
			if category == "Current Conditions" {
				report.ObservedAt, _ = ptypes.TimestampProto(*item.UpdatedParsed)
				report.ObservationId = item.GUID
				report.CreatedAt, _ = ptypes.TimestampProto(*item.PublishedParsed)
				report.UpdatedAt, _ = ptypes.TimestampProto(*item.UpdatedParsed)

				report.Conditions = currentConditionsToCondition(item.Description)
			} else if category == "Weather Forecasts" {
				forecast := &weather.WeatherForecast{
					ForecastId: item.GUID,
					Conditions: forecastConditionToCondition(item.Description),
				}

				forecast.CreatedAt, _ = ptypes.TimestampProto(*item.PublishedParsed)
				forecast.UpdatedAt, _ = ptypes.TimestampProto(*item.UpdatedParsed)

				forecastDayOfWeek := strings.Split(item.Title, ":")
				forecastDayOfWeek = strings.Split(forecastDayOfWeek[0], " ")
				forecastFor, err := futureDateFromFeedDate(*item.PublishedParsed, forecastDayOfWeek[0])

				if err == nil {
					if len(forecastDayOfWeek) > 1 && forecastDayOfWeek[1] == "night" {
						forecastFor = time.Date(forecastFor.Year(), forecastFor.Month(), forecastFor.Day(), 23, 0, 0, 0, forecastFor.Location())
					} else {
						forecastFor = time.Date(forecastFor.Year(), forecastFor.Month(), forecastFor.Day(), 12, 0, 0, 0, forecastFor.Location())
					}

					forecast.ForecastedFor, _ = ptypes.TimestampProto(forecastFor)
				}

				forecasts = append(forecasts, forecast)
			}
		}
	}

	return report, forecasts, nil
}

func currentConditionsToCondition(cc string) *weather.WeatherCondition {
	cond := &weather.WeatherCondition{}

	records := strings.Split(cc, "<br/>\n")
	for _, record := range records {
		record = strings.Replace(record, "<b>", "", -1)
		record = strings.Replace(record, "</b>", "", -1)
		recordParts := strings.Split(record, ":")
		switch recordParts[0] {
		case "Condition":
			cond.Summary = strings.TrimSpace(recordParts[1])
			cond.SummaryIcon = iconFromFeedText(cond.Summary)
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

func forecastConditionToCondition(fc string) *weather.WeatherCondition {
	cond := &weather.WeatherCondition{}
	records := strings.Split(fc, ".")
	for idx, record := range records {
		record = strings.TrimSpace(record)

		if idx == 0 {
			cond.Summary = record
			cond.SummaryIcon = iconFromFeedText(record)
			continue
		}

		if strings.HasPrefix(record, "Wind chill") {
			val, err := floatFromFeedText(record)
			if err == nil {
				cond.WindChill = val
			}
		} else if strings.HasPrefix(record, "Wind") {
			strippedRecord := strings.TrimPrefix(record, "Wind ")
			fields := strings.Split(strippedRecord, " ")
			for fieldIdx, field := range fields {
				if field == "km/h" && fieldIdx != 0 {
					val, err := strconv.ParseInt(fields[fieldIdx-1], 10, 32)
					if err == nil {
						cond.WindSpeed = int32(val)
						break
					}
				}
			}
		} else if strings.HasPrefix(record, "UV index") {
			strippedRecord := strings.TrimPrefix(record, "UV index ")
			fields := strings.Split(strippedRecord, " ")

			val, err := strconv.ParseInt(fields[0], 10, 8)
			if err == nil {
				cond.UvIndex = int32(val)
			}
		} else if strings.HasPrefix(record, "High") ||
			strings.HasPrefix(record, "Low") ||
			strings.HasPrefix(record, "Temperature") {
			val, err := floatFromFeedText(record)
			if err == nil {
				cond.Temperature = val
			}
		}
	}

	return cond
}

func iconFromFeedText(text string) weather.WeatherIcon {
	text = strings.ToLower(text)
	if strings.Contains(text, "snow") || strings.Contains(text, "flurries") {
		return weather.WeatherIcon_SNOW
	}

	if strings.Contains(text, "rain") {
		if strings.Contains(text, "chance") || strings.Contains(text, "partially") {
			return weather.WeatherIcon_CHANCE_OF_RAIN
		} else if strings.Contains(text, "storm") || strings.Contains(text, "lightning") {
			return weather.WeatherIcon_THUNDERSTORMS
		}
		return weather.WeatherIcon_RAIN
	}

	if strings.Contains(text, "thunder") {
		return weather.WeatherIcon_THUNDERSTORMS
	}

	if strings.Contains(text, "cloud") {
		if strings.Contains(text, "partially") {
			return weather.WeatherIcon_PARTIALLY_CLOUDY
		} else if strings.Contains(text, "sun") {
			return weather.WeatherIcon_MOSTLY_CLOUDY
		}
		return weather.WeatherIcon_CLOUDY
	}

	if strings.Contains(text, "fog") {
		return weather.WeatherIcon_FOG
	}
	if strings.Contains(text, "sunny") {
		if strings.Contains(text, "partially") {
			return weather.WeatherIcon_PARTIALLY_CLOUDY
		}
		return weather.WeatherIcon_SUNNY
	}

	return weather.WeatherIcon_SUNNY
}

func floatFromFeedText(input string) (float32, error) {
	ret := float32(0)
	retSet := false

	fields := strings.Split(input, " ")
	for fieldIdx, field := range fields {
		val, err := strconv.ParseFloat(field, 32)
		if err != nil {
			continue
		}

		if fieldIdx != 0 && fields[fieldIdx-1] == "minus" {
			val *= -1
		}

		ret = float32(val)
		retSet = true
	}

	if retSet {
		return ret, nil
	}
	return 0, errors.New("no value present")
}

func futureDateFromFeedDate(startDate time.Time, futureDayOfWeek string) (time.Time, error) {
	futureDayOfWeek = strings.ToLower(futureDayOfWeek)
	startDayOfWeek := strings.ToLower(startDate.Format("Monday"))

	futureDayIdx := -1
	for idx, day := range dayOfWeek {
		if day == futureDayOfWeek {
			futureDayIdx = idx
			break
		}
	}

	startDayIdx := -1
	for idx, day := range dayOfWeek {
		if day == startDayOfWeek {
			startDayIdx = idx
			break
		}
	}

	if futureDayIdx < 0 || startDayIdx < 0 {
		return startDate, ErrInvalidDate
	}

	delta := deltaBetweenDays[startDayIdx][futureDayIdx]

	futureDate := startDate.AddDate(0, 0, delta)
	return futureDate, nil
}

var dayOfWeek = []string{
	"sunday",
	"monday",
	"tuesday",
	"wednesday",
	"thursday",
	"friday",
	"saturday",
}

var deltaBetweenDays = [][]int{
	// sunday
	{
		0,
		1,
		2,
		3,
		4,
		5,
		6,
	},
	// monday
	{
		6,
		0,
		1,
		2,
		3,
		4,
		5,
	},
	// tuesday
	{
		5,
		6,
		0,
		1,
		2,
		3,
		4,
	},
	// wednesday
	{
		4,
		5,
		6,
		0,
		1,
		2,
		3,
	},
	// thursday
	{
		3,
		4,
		5,
		6,
		0,
		1,
		2,
	},
	// friday
	{
		2,
		3,
		4,
		5,
		6,
		0,
		1,
	},
	// saturday
	{
		1,
		2,
		3,
		4,
		5,
		6,
		0,
	},
}
