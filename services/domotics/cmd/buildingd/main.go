package main

import (
	"fmt"
	"net"

	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	portEnvVar = "PORT"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(portEnvVar)

	p := domotics.NewInMemoryPersister()
	s := domotics.NewService(logger, p)

	connStr := fmt.Sprintf("%s:%d", "", viper.GetInt(portEnvVar))
	lis, err := net.Listen("tcp", connStr)
	if err != nil {
		logger.Fatal("error initializing listener",
			zap.Error(err),
		)
	}
	defer lis.Close()
	logger.Info("listening",
		zap.String("local_addr", connStr),
	)

	api := domotics.NewAPI(logger, nil, s)

	grpcServer := grpc.NewServer()
	domotics.RegisterBuildingAdminServiceServer(grpcServer, api)
	grpcServer.Serve(lis)
}
