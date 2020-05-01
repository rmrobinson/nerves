package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	_ "github.com/mattn/go-sqlite3" // Blank import for sql drivers is "standard"
	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	portEnvVar   = "PORT"
	dbPathEnvVar = "DB_PATH"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(portEnvVar)
	viper.BindEnv(dbPathEnvVar)

	sqldb, err := sql.Open("sqlite3", viper.GetString(dbPathEnvVar))
	if err != nil {
		logger.Fatal("unable to open db",
			zap.Error(err),
		)
	}

	p := domotics.NewSQLPersister(logger, sqldb)
	s := domotics.NewService(logger, p)

	if err := s.Setup(context.Background()); err != nil {
		logger.Fatal("unable to setup service",
			zap.Error(err),
		)
	}

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
	domotics.RegisterBuildingServiceServer(grpcServer, api)
	grpcServer.Serve(lis)
}
