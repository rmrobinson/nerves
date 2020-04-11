package main

import (
	"context"
	"fmt"
	"net"

	"github.com/rmrobinson/nerves/services/feed"
	"github.com/rmrobinson/nerves/services/feed/aggregator"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	mockDB := aggregator.NewMockDB(logger)
	state := aggregator.NewState(logger, mockDB)
	api := aggregator.NewAPI(logger, mockDB)

	logger.Debug("starting to initialize")
	err = state.Initialize(context.Background())
	if err != nil {
		panic(err)
	}

	req := &feed.ListFeedsRequest{}
	resp, err := state.ListFeeds(context.Background(), req)
	if err != nil {
		panic(err)
	}
	for _, feed := range resp.Feeds {
		logger.Debug("feed watching",
			zap.String("name", feed.Name),
			zap.String("url", feed.Url),
		)
	}

	close := make(chan bool)

	go func() {
		logger.Debug("starting to run refresher")
		state.Run(close)
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 10103))
	if err != nil {
		logger.Fatal("failed to listen",
			zap.Error(err),
		)
	}

	grpcServer := grpc.NewServer()
	feed.RegisterFeedServiceServer(grpcServer, api)
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatal("failed to serve",
			zap.Error(err),
		)
	}

}
