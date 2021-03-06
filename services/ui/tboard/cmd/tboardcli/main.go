package main

import (
	"context"
	"net/url"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/rivo/tview"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/rmrobinson/nerves/services/news"
	"github.com/rmrobinson/nerves/services/transit"
	"github.com/rmrobinson/nerves/services/ui/tboard/widget"
	"github.com/rmrobinson/nerves/services/weather"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	envVarWeatherdEndpoint  = "WEATHERD_ENDPOINT"
	envVarDomoticsdEndpoint = "DOMOTICSD_ENDPOINT"
	envVarNewsdEndpoint     = "NEWSD_ENDPOINT"
	envVarTransitdEndpoint  = "TRANSITD_ENDPOINT"
	envVarLatitude          = "LATITUDE"
	envVarLongitude         = "LONGITUDE"
	envVarTransitdStopID    = "TRANSIT_STOP_ID"
)

func main() {
	viper.SetEnvPrefix("NVS")
	viper.BindEnv(envVarWeatherdEndpoint)
	viper.BindEnv(envVarDomoticsdEndpoint)
	viper.BindEnv(envVarNewsdEndpoint)
	viper.BindEnv(envVarTransitdEndpoint)
	viper.BindEnv(envVarLatitude)
	viper.BindEnv(envVarLongitude)
	viper.BindEnv(envVarTransitdStopID)

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

	weatherConn, err := grpc.Dial(viper.GetString(envVarWeatherdEndpoint), grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial weather server",
			zap.String("endpoint", viper.GetString(envVarWeatherdEndpoint)),
			zap.Error(err),
		)
	}
	defer weatherConn.Close()

	weatherClient := weather.NewWeatherServiceClient(weatherConn)

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

	weatherView := widget.NewWeatherCondition(app)
	go func() {
		for {
			report, err := weatherClient.GetCurrentReport(context.Background(), &weather.GetCurrentReportRequest{
				Latitude:  viper.GetFloat64(envVarLatitude),
				Longitude: viper.GetFloat64(envVarLongitude),
			})
			if err != nil {
				logger.Warn("unable to get weather")
			}

			if report == nil {
				report = &weather.GetCurrentReportResponse{}
			}
			weatherView.Refresh(report.Report)

			time.Sleep(time.Second * 3)
		}
	}()

	forecastView := widget.NewWeatherForecast(app, 6)
	go func() {
		for {
			forecast, err := weatherClient.GetForecast(context.Background(), &weather.GetForecastRequest{
				Latitude:  viper.GetFloat64(envVarLatitude),
				Longitude: viper.GetFloat64(envVarLongitude),
			})
			if err != nil {
				logger.Warn("unable to get weather")
			}

			forecastView.Refresh(forecast)

			time.Sleep(time.Second * 3)
		}
	}()

	domoticsConn, err := grpc.Dial(viper.GetString(envVarDomoticsdEndpoint), grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial domotics server",
			zap.String("endpoint", viper.GetString(envVarDomoticsdEndpoint)),
			zap.Error(err),
		)
	}
	defer domoticsConn.Close()

	devicesClient := bridge.NewBridgeServiceClient(domoticsConn)

	devicesView := widget.NewDevices(app, logger, devicesClient)

	newsConn, err := grpc.Dial(viper.GetString(envVarNewsdEndpoint), grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial news server",
			zap.String("endpoint", viper.GetString(envVarNewsdEndpoint)),
			zap.Error(err),
		)
	}
	defer newsConn.Close()

	newsClient := news.NewNewsServiceClient(newsConn)

	listArticlesResp, err := newsClient.ListArticles(context.Background(), &news.ListArticlesRequest{})
	if err != nil {
		logger.Warn("unable to retrieve articles",
			zap.Error(err),
		)
		listArticlesResp = &news.ListArticlesResponse{}
	}

	articlesView := widget.NewArticles(app, listArticlesResp.Articles)

	articlesView.SetNextWidget(devicesView)
	devicesView.SetNextWidget(articlesView)

	transitConn, err := grpc.Dial(viper.GetString(envVarTransitdEndpoint), grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial transit server",
			zap.String("endpoint", viper.GetString(envVarTransitdEndpoint)),
			zap.Error(err),
		)
	}
	defer transitConn.Close()

	transitClient := transit.NewTransitServiceClient(transitConn)

	getStopArrivalsResp, err := transitClient.GetStopArrivals(context.Background(), &transit.GetStopArrivalsRequest{
		StopCode:              viper.GetString(envVarTransitdStopID),
		ExcludeArrivalsBefore: ptypes.TimestampNow(),
	})
	if err != nil {
		logger.Warn("unable to retrieve transit arrivals",
			zap.Error(err),
		)
		getStopArrivalsResp = &transit.GetStopArrivalsResponse{}
	}

	transitView := widget.NewTransit(app, 3)
	go func() {
		for {
			transitView.Refresh(getStopArrivalsResp.Stop, getStopArrivalsResp.Arrivals)

			time.Sleep(time.Second * 30)
		}
	}()
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(torontoTime, 4, 1, false).
				AddItem(weatherView, 17, 1, false).
				AddItem(forecastView, 0, 1, false).
				AddItem(transitView, 8, 1, false), 29, 1, false).
			AddItem(articlesView, 50, 1, true).
			AddItem(devicesView, 0, 1, true), 0, 1, true).
		AddItem(debugView, 3, 1, false)

	if err := app.SetRoot(layout, true).SetFocus(layout).Run(); err != nil {
		panic(err)
	}
}
