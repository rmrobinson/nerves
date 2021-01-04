package main

import (
	"context"

	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	apiKeyEnvVar  = "API_KEY"
	idEnvVar      = "ID"
	connStrEnvVar = "CONN_STR"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("NVS")
	viper.BindEnv(idEnvVar)
	viper.BindEnv(apiKeyEnvVar)
	viper.BindEnv(connStrEnvVar)

	s := NewService(logger, viper.GetString(idEnvVar), viper.GetString(apiKeyEnvVar))

	m := bridge.NewMonitor(logger, s, []string{"nanoleaf_aurora:light"})

	// Allow us to connect even if we can't receive an SSDP advertisement (different network possibly)
	connStr := viper.GetString(connStrEnvVar)
	if len(connStr) > 0 {
		s.Alive("nanoleaf_aurora:light", viper.GetString(idEnvVar), connStr)
	}

	logger.Info("listening for advertisements")
	m.Run(context.Background())
}
