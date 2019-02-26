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

	svc := noaa.NewService(logger, "https://api.weather.gov/gridpoints/MTR/88,126")

	conditions, err := svc.GetCurrentReport(context.Background())
	if err != nil {
		logger.Info("error getting weather")
	}

	spew.Dump(conditions)
}
