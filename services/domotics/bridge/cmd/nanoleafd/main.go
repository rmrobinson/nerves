package main

import (
	"context"

	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	apiKeyEnvVar = "API_KEY"
	idEnvVar     = "ID"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(idEnvVar)
	viper.BindEnv(apiKeyEnvVar)

	s := NewService(logger, viper.GetString(idEnvVar), viper.GetString(apiKeyEnvVar))

	m := bridge.NewMonitor(logger, s, []string{"nanoleaf_aurora:light"})

	logger.Info("listening for advertisements")
	m.Run(context.Background())
}
