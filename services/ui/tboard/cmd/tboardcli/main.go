package main

import (
	"context"
	"net/url"
	"time"

	"github.com/rivo/tview"
	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/news"
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

	weatherConn, err := grpc.Dial("127.0.0.1:10101", grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial weather server",
			zap.Error(err),
		)
	}
	defer weatherConn.Close()

	weatherClient := weather.NewWeatherClient(weatherConn)

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

	domoticsConn, err := grpc.Dial("127.0.0.1:10102", grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial domotics server",
			zap.Error(err),
		)
	}
	defer domoticsConn.Close()

	devicesClient := domotics.NewDeviceServiceClient(domoticsConn)

	listDevicesResp, err := devicesClient.ListDevices(context.Background(), &domotics.ListDevicesRequest{})
	if err != nil {
		logger.Warn("unable to retrieve devices",
			zap.Error(err),
		)
		listDevicesResp = &domotics.ListDevicesResponse{}
	}

	devicesView := widget.NewDevices(app, listDevicesResp.Devices)

	articlesView := widget.NewArticles(app, []*news.Article{
		{
			Title:       "Trump walks out of meeting with Democrats on government shutdown",
			Description: `Hours after U.S. President Donald Trump called a meeting with Democrat leaders a "total waste of time," the House passed a bill to reopen parts of the government — but it's unlikely to survive the Republican-controlled Senate.`,
			Link:        "https://www.cbc.ca/news/world/trump-walks-out-shutdown-meeting-1.4972128",
		},
		{
			Title:       "Canadian astronomers discover 2nd mysterious repeating fast radio burst",
			Description: `Out in the depths of space, there are radio signals that astronomers don't understand. Now a Canadian research team has found a repeating signal, only the second of its kind to be discovered.`,
			Link:        "https://www.cbc.ca/news/technology/fast-radio-bursts-1.4969863",
		},
	})

	articlesView.SetNextWidget(devicesView)
	devicesView.SetNextWidget(articlesView)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(torontoTime, 4, 1, false).
				AddItem(calgaryTime, 4, 1, false).
				AddItem(weatherView, 17, 1, false).
				AddItem(forecastView, 0, 1, false), 24, 1, false).
			AddItem(articlesView, 50, 1, true).
			AddItem(devicesView, 0, 1, true), 0, 1, true).
		AddItem(debugView, 3, 1, false)

	if err := app.SetRoot(layout, true).SetFocus(layout).Run(); err != nil {
		panic(err)
	}
}
