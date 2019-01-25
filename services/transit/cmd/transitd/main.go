package main

import (
	"context"
	"fmt"
	"net"

	"github.com/rmrobinson/nerves/services/transit"
	"github.com/rmrobinson/nerves/services/transit/gtfs"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	dataset := gtfs.NewDataset(logger)
	err = dataset.LoadFromFSPath(context.Background(), "/Users/robert.robinson/Downloads/GRT_GTFS")
	if err != nil {
		logger.Fatal("error loading dataset",
			zap.Error(err),
		)
	}

	feed := transit.NewFeed(logger, dataset, "")

	svc := transit.NewService(logger)
	svc.AddFeed(feed)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 10104))
	if err != nil {
		logger.Fatal("failed to listen",
			zap.Error(err),
		)
	}

	grpcServer := grpc.NewServer()
	transit.RegisterTransitServiceServer(grpcServer, svc)
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatal("failed to serve",
			zap.Error(err),
		)
	}
}
