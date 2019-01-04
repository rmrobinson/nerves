package main

import (
	"context"
	"fmt"
	"net"

	"github.com/rmrobinson/nerves/services/weather"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	ecURL := "https://weather.gc.ca/rss/city/on-82_e.xml"

	logger, _ := zap.NewDevelopment()
	ecf := weather.NewEnvironmentCanadaFeed(logger, ecURL)
	go ecf.Run(context.Background())

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 10101))
	if err != nil {
		logger.Fatal("failed to listen",
			zap.Error(err),
		)
	}

	grpcServer := grpc.NewServer()
	weather.RegisterWeatherServer(grpcServer, weather.NewAPI(ecf))
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatal("failed to serve",
			zap.Error(err),
		)
	}
}