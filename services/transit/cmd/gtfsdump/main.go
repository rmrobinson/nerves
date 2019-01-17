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
	feed, err := svc.GetRealtimeFeed(context.Background(), "http://192.237.29.212:8080/gtfsrealtime/VehiclePositions")
	if err != nil {
		logger.Warn("error getting feed")
		return
	}

	fmt.Printf("%v\n", feed.Header)
	for _, entity := range feed.Entity {
		fmt.Printf("%v\n", entity)
	}
}
