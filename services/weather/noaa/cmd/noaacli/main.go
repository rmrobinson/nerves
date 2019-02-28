package main

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/rmrobinson/nerves/services/weather/noaa"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	station := noaa.NewStation(logger, "https://api.weather.gov/gridpoints/MTR/88,126", "San Francisco", 37.7749, -122.4194)
	svc := noaa.NewService(logger)
	svc.AddStation(station)

	conditions, err := svc.GetReport(context.Background(), 37.808673, -122.4120097)
	if err != nil {
		logger.Info("error getting weather")
	}

	spew.Dump(conditions)
}
