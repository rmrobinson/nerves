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
	err := exampleDataset.LoadFromFSPath(context.Background(), "/Users/robert.robinson/Downloads/GRT_GTFS")
	if err != nil {
		logger.Warn("error loading dataset",
			zap.Error(err),
		)
		return nil, err
	}

	exampleFeed := transit.NewFeed(logger, exampleDataset, "")

	return exampleFeed, nil
}

func printFeed(f *transit.Feed) {
	for _, route := range f.Routes() {
		fmt.Printf("Route %s (%s):\n", route.ID, route.ShortName)
		for _, trip := range route.Trips() {
			fmt.Printf(" Trip %s (%s) starts at %s at %s:\n", trip.ID, trip.Headsign, trip.Plan()[0].ArrivalTime, trip.Plan()[0].Stop().Name)
		}

	}

	for stopID, stop := range f.Stops() {
		fmt.Printf("Stop %s (%s):\n", stopID, stop.Name)
		arrivals := stop.ArrivalsToday()
		for _, arrival := range arrivals {
			fmt.Printf(" Trip %s (%s) arriving at %s\n", arrival.TripID, arrival.VehicleHeadsign(), arrival.ArrivalTime)
		}
	}
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

	// UW DC station
	s := svc.StopClosestTo(43.4728998, -80.5420669)
	fmt.Printf("Stop %s (%s):\n", s.ID, s.Name)
	arrivals := s.RemainingArrivalsToday()
	for _, arrival := range arrivals {
		fmt.Printf(" Route %s (%s) arriving at %s\n", arrival.RouteID(), arrival.VehicleHeadsign(), arrival.ArrivalTime.String())
	}

	s = svc.StopByID("3629")
	fmt.Printf("Stop %s (%s):\n", s.ID, s.Name)
	arrivals = s.RemainingArrivalsToday()
	for _, arrival := range arrivals {
		fmt.Printf(" Route %s (%s) arriving at %s\n", arrival.RouteID(), arrival.VehicleHeadsign(), arrival.ArrivalTime.String())
	}
}
