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

	logger, _ := zap.NewDevelopment()
	cbcf := news.NewCBCFeed(logger, cbcURL)
	go cbcf.Run(context.Background())

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 10103))
	if err != nil {
		logger.Fatal("failed to listen",
			zap.Error(err),
		)
	}

	grpcServer := grpc.NewServer()
	news.RegisterNewsServiceServer(grpcServer, news.NewAPI(logger, cbcf))
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatal("failed to serve",
			zap.Error(err),
		)
	}
}
