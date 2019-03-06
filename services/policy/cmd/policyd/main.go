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

	stateRefresh := make(chan bool)
	state := policy.NewState(logger, domoticsConn, stateRefresh)

	go state.Monitor(context.Background())

	engine := policy.NewEngine(logger, state)

	p := &policy.Policy{
		Condition: &policy.Condition{
			Device: &policy.DeviceCondition{
				DeviceId: "Asdf",
				Binary: &policy.DeviceCondition_Binary{
					IsOn: true,
				},
			},
		},
		Actions: []*policy.Action{
			{
				Name: "test policy",
				Type: policy.Action_Log,
			},
		},
	}
	engine.AddPolicy(p)
	engine.Run(context.Background())
}
