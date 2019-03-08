package main

import (
	"context"

	"github.com/rmrobinson/nerves/services/policy"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	envVarDomoticsdEndpoint = "DOMOTICSD_ENDPOINT"
)

func main() {
	viper.SetEnvPrefix("NVS")
	viper.BindEnv(envVarDomoticsdEndpoint)

	logger, _ := zap.NewDevelopment()

	var grpcOpts []grpc.DialOption
	grpcOpts = append(grpcOpts, grpc.WithInsecure())

	domoticsConn, err := grpc.Dial(viper.GetString(envVarDomoticsdEndpoint), grpcOpts...)
	if err != nil {
		logger.Warn("unable to dial domotics server",
			zap.String("endpoint", viper.GetString(envVarDomoticsdEndpoint)),
			zap.Error(err),
		)
	}
	defer domoticsConn.Close()

	state := policy.NewState(logger, domoticsConn)

	engine := policy.NewEngine(logger, state)

	go state.Monitor(context.Background())

	p := &policy.Policy{
		Name: "test policy 1",
		Condition: &policy.Condition{
			Name: "every minute",
			Cron: &policy.Condition_Cron{
				Tz:    "America/Los_Angeles",
				Entry: "0 * * * * *",
			},
		},
		Actions: []*policy.Action{
			{
				Name: "test policy action",
				Type: policy.Action_Log,
			},
		},
	}
	engine.AddPolicy(p)
	engine.Run(context.Background())
}
