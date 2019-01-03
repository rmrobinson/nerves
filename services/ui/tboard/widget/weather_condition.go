package widget

import (
	"fmt"

	"github.com/rivo/tview"
)

// WeatherCondition is a widget to display the current weather conditions.
// It is a fixed-size widget of width 24.
type WeatherCondition struct {
	*tview.Flex

	app *tview.Application

	conditionsView  *tview.TextView
	temperatureView *tview.TextView
	windChillView   *tview.TextView
	humidityView    *tview.TextView
	pressureView    *tview.TextView
	windSpeedView   *tview.TextView
	visibilityView  *tview.TextView
	dewPointView    *tview.TextView
	uvIndexView     *tview.TextView

	condition *WeatherConditionInfo
}

// NewWeatherCondition creates a new weather condition widget.
// Nothing will be displayed until a WeatherConditionInfo is set on this view using Refresh()
func NewWeatherCondition(app *tview.Application) *WeatherCondition {
	wc := &WeatherCondition{
		Flex: tview.NewFlex(),
		app:  app,

		conditionsView:  tview.NewTextView(),
		temperatureView: tview.NewTextView(),
		windChillView:   tview.NewTextView(),
		humidityView:    tview.NewTextView(),
		pressureView:    tview.NewTextView(),
		windSpeedView:   tview.NewTextView(),
		visibilityView:  tview.NewTextView(),
		dewPointView:    tview.NewTextView(),
		uvIndexView:     tview.NewTextView(),
	}

	wc.conditionsView.SetTextAlign(tview.AlignCenter).
		SetTitle("Cond").
		SetBorder(true)
	wc.temperatureView.SetTextAlign(tview.AlignCenter).
		SetTitle("Temp").
		SetBorder(true)
	wc.windChillView.SetTextAlign(tview.AlignCenter).
		SetTitle("WChl").
		SetBorder(true)
	wc.windSpeedView.SetTextAlign(tview.AlignCenter).
		SetTitle("WSpd").
		SetBorder(true)
	wc.humidityView.SetTextAlign(tview.AlignCenter).
		SetTitle("Humd").
		SetBorder(true)
	wc.pressureView.SetTextAlign(tview.AlignCenter).
		SetTitle("Prsre").
		SetBorder(true)
	wc.visibilityView.SetTextAlign(tview.AlignCenter).
		SetTitle("Vis").
		SetBorder(true)
	wc.dewPointView.SetTextAlign(tview.AlignCenter).
		SetTitle("Dew").
		SetBorder(true)
	wc.uvIndexView.SetTextAlign(tview.AlignCenter).
		SetTitle("UV Idx").
		SetBorder(true)

	leftCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(wc.temperatureView, 3, 1, false).
		AddItem(wc.humidityView, 3, 1, false).
		AddItem(wc.visibilityView, 3, 1, false).
		AddItem(wc.dewPointView, 3, 1, false)

	rightCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(wc.windChillView, 3, 1, false).
		AddItem(wc.windSpeedView, 3, 1, false).
		AddItem(wc.pressureView, 3, 1, false).
		AddItem(wc.uvIndexView, 3, 1, false)

	wc.SetBorder(true).
		SetTitle("Weather Conditions").
		SetTitleAlign(tview.AlignLeft)

	wc.SetDirection(tview.FlexRow).
		AddItem(wc.conditionsView, 3, 1, false).
		AddItem(tview.NewFlex().
			AddItem(leftCol, 11, 1, false).
			AddItem(rightCol, 11, 1, false), 0, 1, false)

	return wc
}

// Refresh takes the supplied information and updates the widget with the supplied values.
func (wc *WeatherCondition) Refresh(conditions *WeatherConditionInfo) {
	wc.app.QueueUpdateDraw(func() {
		wc.condition = conditions

		wc.conditionsView.SetText(wc.condition.Description)
		wc.temperatureView.SetText(fmt.Sprintf("%2.1f C", wc.condition.TemperatureCelsius))
		wc.windChillView.SetText(fmt.Sprintf("%2.1f C", wc.condition.WindChillCelsius))
		wc.humidityView.SetText(fmt.Sprintf("%3d %%", wc.condition.HumidityPercentage))
		wc.pressureView.SetText(fmt.Sprintf("%3.1f kPa", wc.condition.PressureKPa))
		wc.windSpeedView.SetText(fmt.Sprintf("%2d km/h", wc.condition.WindSpeedKmPerHr))
		wc.visibilityView.SetText(fmt.Sprintf("%3d km", wc.condition.VisibilityKm))
		wc.dewPointView.SetText(fmt.Sprintf("%2.1f C", wc.condition.DewPointCelsius))
		wc.uvIndexView.SetText(fmt.Sprintf("%1d", wc.condition.UVIndex))
	})
}
