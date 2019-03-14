package policy

func (c *Condition) validate() bool {
	if c.Set != nil {
		if len(c.Set.Conditions) < 1 {
			return false
		} else if c.Set.Operator == Condition_Set_NO_OPERATOR {
			return false
		}
	} else if c.Cron != nil {
		if len(c.Cron.Entry) < 1 {
			return false
		}
	} else if c.Weather != nil {
		if len(c.Weather.Location) < 1 {
			return false
		} else if c.Weather.Temperature == nil {
			return false
		}
	} else {
		return false
	}

	return true
}

func (c *Condition) triggered(state *State) bool {
	triggered := false

	if c.Set != nil {
		anyTriggered := false
		allTriggered := true
		for _, condition := range c.Set.Conditions {
			triggered := condition.triggered(state)

			if triggered && c.Set.Operator == Condition_Set_OR {
				anyTriggered = true
				break
			} else if !triggered && c.Set.Operator == Condition_Set_AND {
				allTriggered = false
				break
			}
		}

		if c.Set.Operator == Condition_Set_OR && anyTriggered {
			triggered = true
		} else if c.Set.Operator == Condition_Set_AND && allTriggered {
			triggered = true
		}
	} else if c.Cron != nil {
		if cron, ok := state.cronsByCond[c]; ok {
			if cron.active {
				triggered = true
			}
		}
	} else if c.Weather != nil {
		if report, ok := state.weatherState[c.Weather.Location]; ok {
			switch c.Weather.Temperature.Comparison {
			case Comparison_EQUAL:
				triggered = c.Weather.Temperature.TemperatureCelsisus == int32(report.Conditions.Temperature)
			case Comparison_GREATER_THAN:
				triggered = int32(report.Conditions.Temperature) > c.Weather.Temperature.TemperatureCelsisus
			case Comparison_GREATER_THAN_EQUAL_TO:
				triggered = int32(report.Conditions.Temperature) >= c.Weather.Temperature.TemperatureCelsisus
			case Comparison_LESS_THAN:
				triggered = int32(report.Conditions.Temperature) < c.Weather.Temperature.TemperatureCelsisus
			case Comparison_LESS_THAN_EQUAL_TO:
				triggered = int32(report.Conditions.Temperature) <= c.Weather.Temperature.TemperatureCelsisus
			}
		}
	}

	// TODO: add other conditions

	if c.Negate {
		triggered = !triggered
	}

	return triggered
}
