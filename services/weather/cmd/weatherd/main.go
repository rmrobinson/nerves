package main

import (
	"fmt"
	"net"

	"github.com/rmrobinson/nerves/services/weather"
	"github.com/rmrobinson/nerves/services/weather/envcan"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	viper.SetEnvPrefix("NVS")
	viper.BindEnv("ENVCAN_MAP")

	logger, _ := zap.NewDevelopment()
	ecsvc, err := envcan.NewService(logger, viper.GetString("ENVCAN_MAP"))
	if err != nil {
		logger.Fatal("error creating feed",
			zap.Error(err),
		)
	}

	//go ecsvc.Run(context.Background())

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 10101))
	if err != nil {
		logger.Fatal("failed to listen",
			zap.Error(err),
		)
	}

	grpcServer := grpc.NewServer()
	weather.RegisterWeatherServiceServer(grpcServer, weather.NewAPI(ecsvc))
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Fatal("failed to serve",
			zap.Error(err),
		)
	}
}
