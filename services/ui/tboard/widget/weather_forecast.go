package widget

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

type weatherForecastRecord struct {
	*tview.Flex

	dateText    *tview.TextView
	lowText     *tview.TextView
	highText    *tview.TextView
	detailsText *tview.TextView
}

func newWeatherForecastRecord() *weatherForecastRecord {
	wfc := &weatherForecastRecord{
		Flex:        tview.NewFlex(),
		dateText:    tview.NewTextView(),
		lowText:     tview.NewTextView(),
		highText:    tview.NewTextView(),
		detailsText: tview.NewTextView(),
	}

	wfc.dateText.SetTextAlign(tview.AlignLeft)
	wfc.detailsText.SetTextAlign(tview.AlignRight)
	wfc.lowText.SetTextAlign(tview.AlignLeft)
	wfc.highText.SetTextAlign(tview.AlignLeft)

	wfc.SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(wfc.dateText, 0, 1, false).
			AddItem(wfc.detailsText, 0, 1, false), 1, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(wfc.lowText, 0, 1, false).
			AddItem(wfc.highText, 0, 1, false), 1, 1, false)

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
func (wf *WeatherForecast) Refresh(forecast *WeatherForecastInfo) {
	wf.app.QueueUpdateDraw(func() {
		for i := 0; i < len(wf.records); i++ {
			if i >= len(forecast.Records) {
				wf.records[i].dateText.Clear()
				wf.records[i].lowText.Clear()
				wf.records[i].highText.Clear()
				wf.records[i].detailsText.Clear()
				continue
			}

			wf.records[i].dateText.SetText(forecast.Records[i].Date.Format("Monday"))
			wf.records[i].lowText.SetText(fmt.Sprintf("Low %2.f C", forecast.Records[i].LowCelsius))
			wf.records[i].highText.SetText(fmt.Sprintf("High %2.f C", forecast.Records[i].HighCelsius))
			wf.records[i].detailsText.SetText(textToConditionSymbol(forecast.Records[i].Description))
		}
	})
}

func textToConditionSymbol(origText string) string {
	text := strings.ToLower(origText)
	if strings.Contains(text, "snow") || strings.Contains(text, "flurries") {
		return "🌨"
	}

	if strings.Contains(text, "rain") {
		if strings.Contains(text, "chance") || strings.Contains(text,"partially") {
			return "🌦"
		} else if strings.Contains(text, "storm") || strings.Contains(text, "lightning") {
			return "⛈"
		}
		return "🌧"
	}

	if strings.Contains(text, "thunder") {
		return "🌩"
	}

	if strings.Contains(text, "cloud") {
		if strings.Contains(text, "partially") {
			return "🌥"
		} else if strings.Contains(text, "sun") {
			return "🌤"
		}
		return "☁"
	}

	if strings.Contains(text,"sunny") {
		if strings.Contains(text, "partially") {
			return "🌤"
		}
		return "☼"
	}
	return origText
}
