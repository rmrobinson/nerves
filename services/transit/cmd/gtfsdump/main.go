package main

import (
	"context"
	"fmt"

	"github.com/rmrobinson/nerves/services/transit"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	svc := transit.NewService(logger)

	rtFeed, err := svc.GetRealtimeFeed(context.Background(), "http://192.237.29.212:8080/gtfsrealtime/VehiclePositions")
	if err != nil {
		logger.Warn("error getting rtFeed")
		return
	}

	fmt.Printf("%v\n", rtFeed.Header)
	for _, entity := range rtFeed.Entity {
		fmt.Printf("%v\n", entity)
	}

	err = svc.GetFeed(context.Background(), "https://www.regionofwaterloo.ca/opendatadownloads/GRT_GTFS.zip")
	if err != nil {
		logger.Warn("error getting feed")
	}

	// University of Waterloo
	stop := svc.GetClosestStop(43.4722854,-80.5470516)
	fmt.Printf("%+v\n", stop)
}
