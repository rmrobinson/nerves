package policy

import (
	"testing"

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
					Comparison:          Comparison_GREATER_THAN,
					TemperatureCelsisus: 10,
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
					Comparison:          Comparison_GREATER_THAN,
					TemperatureCelsisus: -1,
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
					Comparison:          Comparison_GREATER_THAN,
					TemperatureCelsisus: 10,
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
								Comparison:          Comparison_LESS_THAN,
								TemperatureCelsisus: 10,
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
								Comparison:          Comparison_GREATER_THAN,
								TemperatureCelsisus: 10,
							},
						},
					},
				},
			},
		},
		validate: true,
		trigger:  false,
	},
}

func TestCondition(t *testing.T) {
	s := NewState(zaptest.NewLogger(t), nil)
	s.weatherState["test loc"] = &weather.WeatherReport{
		Conditions: &weather.WeatherCondition{Temperature: 0},
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
