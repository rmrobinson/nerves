package main

import (
	"context"
	"fmt"

	"github.com/rmrobinson/nerves/services/transit"
	"github.com/rmrobinson/nerves/services/transit/gtfs"
	"go.uber.org/zap"
)

func getExampleFeed(logger *zap.Logger) (*transit.Feed, error) {
	exampleDataset := gtfs.NewDataset(logger)
	err := exampleDataset.LoadFromFSPath(context.Background(), "/tmp/sample-feed")
	//err := exampleDataset.LoadFromURL(context.Background(), "https://developers.google.com/transit/gtfs/examples/sample-feed.zip")
	if err != nil {
		logger.Warn("error loading dataset",
			zap.Error(err),
		)
		return nil, err
	}

	exampleFeed := transit.NewFeed(logger, exampleDataset, "")

	for routeID, route := range exampleFeed.Routes() {
		fmt.Printf("Route %s:\n", routeID)
		for _, trip := range route.Trips() {
			fmt.Printf(" Trip %s (%s):\n", trip.ID, trip.Headsign)
			for _, stop := range trip.Plan() {
				fmt.Printf("  %s at %s (%f,%f)\n", stop.ArrivalTime, stop.Stop().Name, stop.Stop().Latitude, stop.Stop().Longitude)
			}
		}
	}

	for stopID, stop := range exampleFeed.Stops() {
		fmt.Printf("Stop %s (%s):\n", stopID, stop.Name)
		for _, arrival := range stop.Arrivals() {
			fmt.Printf(" Trip %s (%s) arriving at %s\n", arrival.TripID, arrival.VehicleHeadsign(), arrival.ArrivalTime)
		}
	}

	return exampleFeed, nil
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	svc := transit.NewService(logger)

	exampleFeed, err := getExampleFeed(logger)
	if err != nil {
		logger.Fatal("cannot run without feed",
			zap.Error(err),
		)
	}

	svc.AddFeed(exampleFeed)
}
