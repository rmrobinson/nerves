package main

import (
	"context"

	"github.com/golang/protobuf/ptypes"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
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

	device := &policy.DeviceAction{
		Id: "test-device-id",
		State: &bridge.DeviceState{
			Binary: &bridge.DeviceState_Binary{
				IsOn: true,
			},
		},
	}
	deviceAction, err := ptypes.MarshalAny(device)
	if err != nil {
		panic(err)
	}

	timer := &policy.TimerAction{
		Id: "test-timer-id",
		Timer: &policy.TimerAction_Timer{
			IntervalMs: 5000,
		},
	}
	timerAction, err := ptypes.MarshalAny(timer)
	if err != nil {
		panic(err)
	}

	p := &policy.Policy{
		Name: "test policy 1 (cron or weather)",
		Condition: &policy.Condition{
			Name: "cron or weather condition",
			Set: &policy.Condition_Set{
				Operator: policy.Condition_Set_OR,
				Conditions: []*policy.Condition{
					{
						Name: "every minute",
						Cron: &policy.Condition_Cron{
							Tz:    "America/Los_Angeles",
							Entry: "0 * * * * *",
						},
					},
					{
						Name: "kitchener temp > 10",
						Weather: &policy.WeatherCondition{
							Location: "YKF",
							Temperature: &policy.WeatherCondition_Temperature{
								Comparison:         policy.Comparison_GREATER_THAN,
								TemperatureCelsius: 10,
							},
						},
					},
				},
			},
		},
		Actions: []*policy.Action{
			{
				Name: "test log action",
				Type: policy.Action_LOG,
			},
			{
				Name:    "test device action",
				Type:    policy.Action_DEVICE,
				Details: deviceAction,
			},
			{
				Name:    "test timer action",
				Type:    policy.Action_TIMER,
				Details: timerAction,
			},
		},
	}
	engine.AddPolicy(p)

	p2 := &policy.Policy{
		Name: "test policy 2 (timer)",
		Condition: &policy.Condition{
			Name: "timer condition",
			Timer: &policy.Condition_Timer{
				Id: "test-timer-id",
			},
		},
		Actions: []*policy.Action{
			{
				Name: "timer action triggered",
				Type: policy.Action_LOG,
			},
		},
	}
	engine.AddPolicy(p2)
	engine.Run(context.Background())
}
