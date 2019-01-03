package main

import (
	"math/rand"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/rivo/tview"
	"github.com/rmrobinson/nerves/services/ui/tboard/widget"
	"github.com/rmrobinson/nerves/services/weather"
)

func main() {
	app := tview.NewApplication()

	// Toronto, Calgary & SF
	locations := []string{
		"America/Toronto",
		"America/Edmonton",
	}

	var tzLocations []*time.Location

	for _, location := range locations {
		loc, err := time.LoadLocation(location)
		if err != nil {
			panic(err)
		}
		tzLocations = append(tzLocations, loc)
	}

	torontoTime := widget.NewTime(app, tzLocations[0])
	go torontoTime.Run()
	calgaryTime := widget.NewTime(app, tzLocations[1])
	go calgaryTime.Run()

	weatherView := widget.NewWeatherCondition(app)
	go func() {
		for {
			conds := &widget.WeatherConditionInfo{
				Description:        "Partially cloudy",
				TemperatureCelsius: -50 + rand.Float32()*100,
				WindChillCelsius:   -50 + rand.Float32()*100,
				HumidityPercentage: uint8(rand.Intn(100)),
				PressureKPa:        90 + rand.Float32()*20,
				WindSpeedKmPerHr:   uint32(rand.Intn(150)),
				VisibilityKm:       uint32(rand.Intn(100)),
				DewPointCelsius:    -10 + rand.Float32()*20,
				UVIndex:            uint8(rand.Intn(10)),
			}
			weatherView.Refresh(conds)

			time.Sleep(time.Second * 3)
		}
	}()

	forecastView := widget.NewWeatherForecast(app, 6)
	go func() {
		for {
			forecast := &weather.GetForecastResponse{}
			for i := 0; i < rand.Intn(9); i++ {
				forecastedFor, _ := ptypes.TimestampProto(time.Now().AddDate(0, 0, i+1))
				forecastRecord := &weather.WeatherForecast{
					ForecastedFor:       forecastedFor,
					Conditions: &weather.WeatherCondition{
						Temperature: -50 + rand.Float32()*100,
						Summary: "Cloudy with 30 percent chance of flurries.",
					},
				}

				forecast.ForecastRecords = append(forecast.ForecastRecords, forecastRecord)
			}
			forecastView.Refresh(forecast)
			time.Sleep(time.Second * 10)
		}
	}()

	devicesView := widget.NewDevices(app, []*widget.DeviceInfo{
		{
			"deviceInfo 1",
			"first deviceInfo with some deets",
			false,
			100,
			0,
			0,
			0,
		},
		{
			"deviceInfo 2",
			"second deviceInfo with some more details",
			true,
			100,
			64,
			128,
			192,
		},
		{
			"deviceInfo 3",
			"third deviceInfo",
			true,
			50,
			0,
			0,
			0,
		},
	},
	)

	layout := tview.NewFlex().
		AddItem(devicesView, 0, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(torontoTime, 4, 1, false).
			AddItem(calgaryTime, 4, 1, false).
			AddItem(weatherView, 0, 1, false).
			AddItem(forecastView, 0, 2, false), 24, 1, false)
	if err := app.SetRoot(layout, true).SetFocus(layout).Run(); err != nil {
		panic(err)
	}
}
