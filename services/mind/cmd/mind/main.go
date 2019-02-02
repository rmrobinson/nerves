package main

import (
	"fmt"
	"net"

	"github.com/nlopes/slack"
	"github.com/rmrobinson/nerves/services/mind"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	envVarSlackKey = "SLACK_KEY"
	envVarSlackChannelID = "SLACK_CHANNEL_ID"
)

func main() {
	viper.SetEnvPrefix("NVS")
	viper.BindEnv(envVarSlackKey)
	viper.BindEnv(envVarSlackChannelID)

	logger, _ := zap.NewDevelopment()

	svc := mind.NewService(logger)

	slackClient := slack.New(viper.GetString(envVarSlackKey))
	slackbot := mind.NewSlackBot(logger, svc, slackClient)

	go slackbot.Run(viper.GetString(envVarSlackChannelID))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 10103))
	if err != nil {
		logger.Fatal("failed to listen",
			zap.Error(err),
		)
	}

	grpcServer := grpc.NewServer()
	mind.RegisterMessageServiceServer(grpcServer, svc)
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatal("failed to serve",
			zap.Error(err),
		)
	}
}
