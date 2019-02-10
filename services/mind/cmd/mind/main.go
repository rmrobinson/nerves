package main

import (
	"fmt"
	"net"

	"github.com/nlopes/slack"
	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/mind"
	"github.com/rmrobinson/nerves/services/news"
	"github.com/rmrobinson/nerves/services/transit"
	"github.com/rmrobinson/nerves/services/weather"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	envVarSlackKey          = "SLACK_KEY"
	envVarSlackChannelID    = "SLACK_CHANNEL_ID"
	envVarLatitude          = "LATITUDE"
	envVarLongitude         = "LONGITUDE"
	envVarWeatherdEndpoint  = "WEATHERD_ENDPOINT"
	envVarNewsdEndpoint     = "NEWSD_ENDPOINT"
	envVarTransitdEndpoint  = "TRANSITD_ENDPOINT"
	envVarDomoticsdEndpoint = "DOMOTICSD_ENDPOINT"
)

func main() {
	viper.SetEnvPrefix("NVS")
	viper.BindEnv(envVarSlackKey)
	viper.BindEnv(envVarSlackChannelID)
	viper.BindEnv(envVarLatitude)
	viper.BindEnv(envVarLongitude)
	viper.BindEnv(envVarWeatherdEndpoint)
	viper.BindEnv(envVarNewsdEndpoint)
	viper.BindEnv(envVarTransitdEndpoint)
	viper.BindEnv(envVarDomoticsdEndpoint)

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

	newsConn, err := grpc.Dial(viper.GetString(envVarNewsdEndpoint), grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial news server",
			zap.String("endpoint", viper.GetString(envVarNewsdEndpoint)),
			zap.Error(err),
		)
	}
	defer newsConn.Close()

	transitConn, err := grpc.Dial(viper.GetString(envVarTransitdEndpoint), grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial transit server",
			zap.String("endpoint", viper.GetString(envVarTransitdEndpoint)),
			zap.Error(err),
		)
	}
	defer transitConn.Close()

	domoticsConn, err := grpc.Dial(viper.GetString(envVarDomoticsdEndpoint), grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial domotics server",
			zap.String("endpoint", viper.GetString(envVarDomoticsdEndpoint)),
			zap.Error(err),
		)
	}
	defer domoticsConn.Close()

	svc := mind.NewService(logger)
	svc.RegisterHandler(mind.NewEcho(logger))
	svc.RegisterHandler(mind.NewWeather(logger,
		weather.NewWeatherServiceClient(weatherConn),
		viper.GetFloat64(envVarLatitude),
		viper.GetFloat64(envVarLongitude)))
	svc.RegisterHandler(mind.NewNews(logger,
		news.NewNewsServiceClient(newsConn)))
	svc.RegisterHandler(mind.NewTransit(logger,
		transit.NewTransitServiceClient(transitConn)))
	svc.RegisterHandler(mind.NewDomotics(logger,
		domotics.NewBridgeServiceClient(domoticsConn),
		domotics.NewDeviceServiceClient(domoticsConn)))

	slackClient := slack.New(viper.GetString(envVarSlackKey))
	slackbot := mind.NewSlackBot(logger, svc, slackClient)

	go slackbot.Run(viper.GetString(envVarSlackChannelID))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 10108))
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
