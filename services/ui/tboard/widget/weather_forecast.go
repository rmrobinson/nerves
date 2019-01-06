package widget

import (
	"fmt"

	"github.com/golang/protobuf/ptypes"
	"github.com/rivo/tview"
	"github.com/rmrobinson/nerves/services/weather"
)

type weatherForecastRecord struct {
	*tview.Flex

	dateText    *tview.TextView
	tempText    *tview.TextView
	detailsText *tview.TextView
}

func newWeatherForecastRecord() *weatherForecastRecord {
	wfc := &weatherForecastRecord{
		Flex:        tview.NewFlex(),
		dateText:    tview.NewTextView(),
		tempText:    tview.NewTextView(),
		detailsText: tview.NewTextView(),
	}

	wfc.dateText.SetTextAlign(tview.AlignLeft)
	wfc.detailsText.SetTextAlign(tview.AlignRight)
	wfc.tempText.SetTextAlign(tview.AlignLeft)

	wfc.SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(wfc.dateText, 0, 4, false).
			AddItem(wfc.detailsText, 0, 1, false), 1, 1, false).
		AddItem(wfc.tempText, 0, 1, false)

	return wfc
}

// WeatherForecast is a widget that displays the upcoming weather forecast.
type WeatherForecast struct {
	*tview.Flex

	app *tview.Application

	records []*weatherForecastRecord
}

// NewWeatherForecast creates a new WeatherForecast widget with the specified number of rows.
// It will not show any data until Refresh() is called to display the data.
func NewWeatherForecast(app *tview.Application, rowCount int) *WeatherForecast {
	wf := &WeatherForecast{
		Flex: tview.NewFlex(),
		app:  app,
	}

	wf.SetBorder(true).
		SetTitle("Weather Forecast").
		SetTitleAlign(tview.AlignLeft)

	wf.SetDirection(tview.FlexRow)
	for i := 0; i < rowCount; i++ {
		wf.records = append(wf.records, newWeatherForecastRecord())
		wf.AddItem(wf.records[i], 2, 1, false)
	}

	return wf
}

// Refresh causes the forecast data to be updated.
func (wf *WeatherForecast) Refresh(forecast *weather.GetForecastResponse) {
	wf.app.QueueUpdateDraw(func() {
		for i := 0; i < len(wf.records); i++ {
			if forecast == nil || i >= len(forecast.ForecastRecords) {
				wf.records[i].dateText.Clear()
				wf.records[i].tempText.Clear()
				wf.records[i].detailsText.Clear()
				continue
			}

			record := forecast.ForecastRecords[i]
			forecastedFor, _ := ptypes.Timestamp(record.ForecastedFor)

			wf.records[i].detailsText.SetText(weatherIconToEmoji(record.Conditions.SummaryIcon))

			// This gives us our 'low' temperature
			if forecastedFor.Hour() == 23 {
				wf.records[i].tempText.SetText(fmt.Sprintf(" Low %2.f C", record.Conditions.Temperature))
				wf.records[i].dateText.SetText(forecastedFor.Format("Monday") + " evening")
			} else {
				wf.records[i].tempText.SetText(fmt.Sprintf(" High %2.f C", record.Conditions.Temperature))
				wf.records[i].dateText.SetText(forecastedFor.Format("Monday"))
			}
		}
	})
}

func weatherIconToEmoji(icon weather.WeatherIcon) string {
	switch icon {
	case weather.WeatherIcon_SUNNY:
		return "☼"
	case weather.WeatherIcon_CLOUDY:
		return "☁"
	case weather.WeatherIcon_PARTIALLY_CLOUDY:
		return "🌤"
	case weather.WeatherIcon_MOSTLY_CLOUDY:
		return "🌥"
	case weather.WeatherIcon_RAIN, weather.WeatherIcon_SNOW_SHOWERS:
		return "🌧"
	case weather.WeatherIcon_CHANCE_OF_RAIN:
		return "🌦"
	case weather.WeatherIcon_SNOW, weather.WeatherIcon_CHANCE_OF_SNOW:
		return "🌨"
	case weather.WeatherIcon_THUNDERSTORMS:
		return "⛈"
	default:
		return icon.String()
	}
}
