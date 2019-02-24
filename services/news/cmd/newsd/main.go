package main

import (
	"context"
	"fmt"
	"net"

	"github.com/rmrobinson/nerves/services/news"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cbcURL := "https://rss.cbc.ca/lineup/topstories.xml"
	bbcURL := "http://feeds.bbci.co.uk/news/rss.xml"

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	api := news.NewAPI(logger)

	cbcf := news.NewCBCFeed(logger, cbcURL, api)
	go cbcf.Run(context.Background())

	bbcf := news.NewBBCFeed(logger, bbcURL, api)
	go bbcf.Run(context.Background())

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 10103))
	if err != nil {
		logger.Fatal("failed to listen",
			zap.Error(err),
		)
	}

	grpcServer := grpc.NewServer()
	news.RegisterNewsServiceServer(grpcServer, api)
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatal("failed to serve",
			zap.Error(err),
		)
	}
}
