package main

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/rmrobinson/nerves/services/weather"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	envVarWeatherdEndpoint = "WEATHERD_ENDPOINT"
	envVarLatitude         = "LATITUDE"
	envVarLongitude        = "LONGITUDE"
)

func main() {
	viper.SetEnvPrefix("NVS")
	viper.BindEnv(envVarWeatherdEndpoint)
	viper.BindEnv(envVarLatitude)
	viper.BindEnv(envVarLongitude)

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
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
	report, err := weatherClient.GetCurrentReport(context.Background(), &weather.GetCurrentReportRequest{
		Latitude:  viper.GetFloat64(envVarLatitude),
		Longitude: viper.GetFloat64(envVarLongitude),
	})
	if err != nil {
		logger.Warn("unable to get weather report")
	}

	spew.Dump(report)

	forecast, err := weatherClient.GetForecast(context.Background(), &weather.GetForecastRequest{
		Latitude:  viper.GetFloat64(envVarLatitude),
		Longitude: viper.GetFloat64(envVarLongitude),
	})
	if err != nil {
		logger.Warn("unable to get weather forecast")
	}

	spew.Dump(forecast)
}
