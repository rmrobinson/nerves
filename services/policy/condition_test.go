package policy

import (
	"testing"

	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/weather"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

type conditionTest struct {
	name     string
	cond     Condition
	validate bool
	trigger  bool
}

var conditionTests = []conditionTest{
	{
		name: "invalid condition fails validation",
		cond: Condition{
			Name: "Test",
			Set:  &Condition_Set{},
		},
		validate: false,
		trigger:  false,
	},
	{
		name: "valid condition passes validation but doesn't execute",
		cond: Condition{
			Name: "Test",
			Weather: &WeatherCondition{
				Location: "test loc",
				Temperature: &WeatherCondition_Temperature{
					Comparison:         Comparison_GREATER_THAN,
					TemperatureCelsius: 10,
				},
			},
		},
		validate: true,
		trigger:  false,
	},
	{
		name: "valid condition passes validation and executes",
		cond: Condition{
			Name: "Test",
			Weather: &WeatherCondition{
				Location: "test loc",
				Temperature: &WeatherCondition_Temperature{
					Comparison:         Comparison_GREATER_THAN,
					TemperatureCelsius: -1,
				},
			},
		},
		validate: true,
		trigger:  true,
	},
	{
		name: "negated condition passes validation and executes",
		cond: Condition{
			Name: "Test",
			Weather: &WeatherCondition{
				Location: "test loc",
				Temperature: &WeatherCondition_Temperature{
					Comparison:         Comparison_GREATER_THAN,
					TemperatureCelsius: 10,
				},
			},
			Negate: true,
		},
		validate: true,
		trigger:  true,
	},
	{
		name: "set condition with or passes validation and executes",
		cond: Condition{
			Name: "Test",
			Set: &Condition_Set{
				Operator: Condition_Set_OR,
				Conditions: []*Condition{
					{
						Name: "every minute",
						Cron: &Condition_Cron{
							Tz:    "America/Los_Angeles",
							Entry: "0 0 * * * *",
						},
					},
					{
						Name: "test temp > 10",
						Weather: &WeatherCondition{
							Location: "test loc",
							Temperature: &WeatherCondition_Temperature{
								Comparison:         Comparison_LESS_THAN,
								TemperatureCelsius: 10,
							},
						},
					},
				},
			},
		},
		validate: true,
		trigger:  true,
	},
	{
		name: "set condition with and passes validation and does not execute",
		cond: Condition{
			Name: "Test",
			Set: &Condition_Set{
				Operator: Condition_Set_AND,
				Conditions: []*Condition{
					{
						Name: "every minute",
						Cron: &Condition_Cron{
							Tz:    "America/Los_Angeles",
							Entry: "0 0 * * * *",
						},
					},
					{
						Name: "test temp > 10",
						Weather: &WeatherCondition{
							Location: "test loc",
							Temperature: &WeatherCondition_Temperature{
								Comparison:         Comparison_GREATER_THAN,
								TemperatureCelsius: 10,
							},
						},
					},
				},
			},
		},
		validate: true,
		trigger:  false,
	},
	{
		name: "valid device rgb condition passes validation and executes",
		cond: Condition{
			Name: "Test",
			Device: &DeviceCondition{
				DeviceId: "test device",
				Binary: &DeviceCondition_Binary{
					IsOn: true,
				},
				Rgb: &DeviceCondition_RGB{
					Red:        100,
					RedCheck:   Comparison_GREATER_THAN,
					Green:      100,
					GreenCheck: Comparison_GREATER_THAN,
					Blue:       100,
					BlueCheck:  Comparison_GREATER_THAN,
				},
			},
		},
		validate: true,
		trigger:  true,
	},
}

func TestCondition(t *testing.T) {
	s := NewState(zaptest.NewLogger(t), nil)
	s.weatherState["test loc"] = &weather.WeatherReport{
		Conditions: &weather.WeatherCondition{Temperature: 0},
	}
	s.deviceState["test device"] = &domotics.Device{
		State: &domotics.DeviceState{
			Binary: &domotics.DeviceState_BinaryState{
				IsOn: true,
			},
			ColorRgb: &domotics.DeviceState_RGBState{
				Red:   150,
				Green: 110,
				Blue:  120,
			},
		},
	}

	for _, tt := range conditionTests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.cond.validate()
			triggered := tt.cond.triggered(s)
			assert.Equal(t, tt.validate, valid)
			assert.Equal(t, tt.trigger, triggered)
		})
	}
}
