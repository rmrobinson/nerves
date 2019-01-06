package main

import (
	"context"
	"net/url"
	"time"

	"github.com/rivo/tview"
	"github.com/rmrobinson/nerves/services/ui/tboard/widget"
	"github.com/rmrobinson/nerves/services/weather"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	app := tview.NewApplication()

	debugView := widget.NewDebug(app)

	zap.RegisterSink("widget", func(*url.URL) (zap.Sink, error) {
		return NewWidgetSink(debugView), nil
	})

	conf := zap.NewDevelopmentConfig()
	// Redirect all messages to the WidgetSink.
	conf.OutputPaths = []string{"widget://"}

	logger, err := conf.Build()
	if err != nil {
		return
	}

	var grpcOpts []grpc.DialOption
	grpcOpts = append(grpcOpts, grpc.WithInsecure())

	conn, err := grpc.Dial("127.0.0.1:10101", grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial",
			zap.Error(err),
		)
	}
	defer conn.Close()

	weatherClient := weather.NewWeatherClient(conn)

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
			report, err := weatherClient.GetCurrentReport(context.Background(), &weather.GetCurrentReportRequest{})
			if err != nil {
				logger.Warn("unable to get weather")
			}

			weatherView.Refresh(report)

			time.Sleep(time.Second * 3)
		}
	}()

	forecastView := widget.NewWeatherForecast(app, 6)
	go func() {
		for {
			forecast, err := weatherClient.GetForecast(context.Background(), &weather.GetForecastRequest{})
			if err != nil {
				logger.Warn("unable to get weather")
			}

			forecastView.Refresh(forecast)

			time.Sleep(time.Second * 3)
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

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(devicesView, 0, 1, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(torontoTime, 4, 1, false).
				AddItem(calgaryTime, 4, 1, false).
				AddItem(weatherView, 17, 1, false).
				AddItem(forecastView, 0, 1, false), 24, 1, false), 0, 1, true).
		AddItem(debugView, 3, 1, false)

	if err := app.SetRoot(layout, true).SetFocus(layout).Run(); err != nil {
		panic(err)
	}
}
