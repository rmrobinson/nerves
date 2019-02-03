package main

import (
	"fmt"
	"net"

	"github.com/nlopes/slack"
	"github.com/rmrobinson/nerves/services/mind"
	"github.com/rmrobinson/nerves/services/weather"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	envVarSlackKey = "SLACK_KEY"
	envVarSlackChannelID = "SLACK_CHANNEL_ID"
	envVarLatitude          = "LATITUDE"
	envVarLongitude         = "LONGITUDE"
	envVarWeatherdEndpoint  = "WEATHERD_ENDPOINT"
)

func main() {
	viper.SetEnvPrefix("NVS")
	viper.BindEnv(envVarSlackKey)
	viper.BindEnv(envVarSlackChannelID)
	viper.BindEnv(envVarLatitude)
	viper.BindEnv(envVarLongitude)
	viper.BindEnv(envVarWeatherdEndpoint)

	logger, _ := zap.NewDevelopment()

	var grpcOpts []grpc.DialOption
	grpcOpts = append(grpcOpts, grpc.WithInsecure())

	weatherConn, err := grpc.Dial(viper.GetString(envVarWeatherdEndpoint), grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial weather server",
			zap.String("endpoint", viper.GetString(envVarWeatherdEndpoint)),
			zap.Error(err),
		)
	}
	defer weatherConn.Close()

	svc := mind.NewService(logger)
	svc.RegisterHandler(mind.NewEcho(logger))
	svc.RegisterHandler(mind.NewWeather(logger,
		weather.NewWeatherServiceClient(weatherConn),
		viper.GetFloat64(envVarLatitude),
		viper.GetFloat64(envVarLongitude)))

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
