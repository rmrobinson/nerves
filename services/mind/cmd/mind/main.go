package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/gocarina/gocsv"
	"github.com/nlopes/slack"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/rmrobinson/nerves/services/mind"
	"github.com/rmrobinson/nerves/services/news"
	"github.com/rmrobinson/nerves/services/transit"
	"github.com/rmrobinson/nerves/services/users"
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
	envVarUsersDBPath       = "USERS_DB_PATH"
)

type csvUser struct {
	Name        string `csv:"name"`
	DisplayName string `csv:"display_name"`
	FirstName   string `csv:"first_name"`
	LastName    string `csv:"last_name"`
}

func getUsers() (map[string]*users.User, error) {
	path := viper.GetString(envVarUsersDBPath)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var csvUsers []*csvUser
	err = gocsv.Unmarshal(f, &csvUsers)
	if err != nil {
		return nil, err
	}

	u := map[string]*users.User{}
	for _, user := range csvUsers {
		u[user.Name] = &users.User{
			Name:        user.Name,
			DisplayName: user.DisplayName,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
		}
	}

	return u, nil
}

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
	viper.BindEnv(envVarUsersDBPath)

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

	users, err := getUsers()
	if err != nil {
		logger.Fatal("unable to load users map",
			zap.Error(err),
		)
	}

	svc := mind.NewService(logger, users)
	svc.RegisterHandler(mind.NewEcho(logger))
	svc.RegisterHandler(mind.NewWeather(logger,
		weather.NewWeatherServiceClient(weatherConn),
		viper.GetFloat64(envVarLatitude),
		viper.GetFloat64(envVarLongitude)))
	svc.RegisterHandler(mind.NewNews(logger,
		news.NewNewsServiceClient(newsConn)))
	svc.RegisterHandler(mind.NewTransit(logger,
		transit.NewTransitServiceClient(transitConn)))

	domoticsHandler := mind.NewDomotics(logger,
		svc,
		bridge.NewBridgeServiceClient(domoticsConn))

	go domoticsHandler.Monitor(context.Background())

	svc.RegisterHandler(domoticsHandler)

	slackClient := slack.New(viper.GetString(envVarSlackKey))
	slackbot := mind.NewSlackBot(logger, svc, slackClient, viper.GetString(envVarSlackChannelID))

	svc.RegisterChannel(slackbot)

	go slackbot.Run()

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
