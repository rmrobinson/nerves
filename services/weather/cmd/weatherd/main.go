package main

import (
	"fmt"
	"net"

	"github.com/rmrobinson/nerves/services/weather"
	"github.com/rmrobinson/nerves/services/weather/envcan"
	"github.com/rmrobinson/nerves/services/weather/noaa"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	viper.SetEnvPrefix("NVS")
	viper.BindEnv("ENVCAN_MAP")

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	api := weather.NewAPI(logger)

	_, err = envcan.NewService(logger, api, viper.GetString("ENVCAN_MAP"))
	if err != nil {
		logger.Fatal("error creating envcan service",
			zap.Error(err),
		)
	}

	sfStation := noaa.NewStation(logger, "https://api.weather.gov/gridpoints/MTR/88,126", "San Francisco", 37.7749, -122.4194)
	api.RegisterStation(sfStation, 37.7749, -122.4194)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 10101))
	if err != nil {
		logger.Fatal("failed to listen",
			zap.Error(err),
		)
	}

	grpcServer := grpc.NewServer()
	weather.RegisterWeatherServiceServer(grpcServer, api)
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatal("failed to serve",
			zap.Error(err),
		)
	}
}
