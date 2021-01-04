package main

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/rmrobinson/deconz-go"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	apiKeyEnvVar = "API_KEY"
	hostEnvVar   = "HOST"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(hostEnvVar)
	viper.BindEnv(apiKeyEnvVar)

	host, portStr, err := net.SplitHostPort(viper.GetString(hostEnvVar))
	if err != nil {
		logger.Fatal("unable to parse supplied host env var",
			zap.Error(err),
		)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		logger.Fatal("unable to parse supplied host port as int",
			zap.Error(err),
		)
	}

	d := deconz.NewClient(&http.Client{}, host, port, viper.GetString(apiKeyEnvVar))
	ds := NewService(logger, d)

	err = ds.Setup(context.Background())
	if err != nil {
		logger.Fatal("error setting up deconz",
			zap.Error(err),
		)
	}

	go func() {
		err = ds.Run()
		if err != nil {
			logger.Fatal("error running deconz service",
				zap.Error(err),
			)
		}
	}()

	lis, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		logger.Fatal("error initializing listener",
			zap.Error(err),
		)
	}
	defer lis.Close()
	logger.Info("listening",
		zap.String("local_addr", lis.Addr().String()),
	)

	ad := bridge.NewAdvertiser(logger, ds.brInfo.Id, lis.Addr().String())
	go ad.Run()
	defer ad.Shutdown()

	grpcServer := grpc.NewServer()
	bridge.RegisterBridgeServiceServer(grpcServer, ds)
	bridge.RegisterPingServiceServer(grpcServer, ad)
	grpcServer.Serve(lis)
}
