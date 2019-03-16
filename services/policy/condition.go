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
	} else if c.Device != nil {
		if c.Device.Binary == nil {
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
			triggered = intComparison(c.Weather.Temperature.Comparison, int32(report.Conditions.Temperature), c.Weather.Temperature.TemperatureCelsius)
		}
	} else if c.Device != nil {
		if device, ok := state.deviceState[c.Device.DeviceId]; ok {
			if c.Device.Binary != nil && device.State.Binary != nil {
				triggered = c.Device.Binary.IsOn == device.State.Binary.IsOn
			}
			if c.Device.Rgb != nil && device.State.ColorRgb != nil {
				triggered = intComparison(c.Device.Rgb.RedCheck, device.State.ColorRgb.Red, c.Device.Rgb.Red) &&
					intComparison(c.Device.Rgb.GreenCheck, device.State.ColorRgb.Green, c.Device.Rgb.Green) &&
					intComparison(c.Device.Rgb.BlueCheck, device.State.ColorRgb.Blue, c.Device.Rgb.Blue)
			}
			if c.Device.Presence != nil && device.State.Presence != nil {
				triggered = c.Device.Presence.IsPresent == device.State.Presence.IsPresent
			}
		}
	}

	// TODO: add other conditions

	if c.Negate {
		triggered = !triggered
	}

	return triggered
}

func intComparison(comparison Comparison, value int32, threshold int32) bool {
	switch comparison {
	case Comparison_EQUAL:
		return value == threshold
	case Comparison_GREATER_THAN:
		return value > threshold
	case Comparison_GREATER_THAN_EQUAL_TO:
		return value >= threshold
	case Comparison_LESS_THAN:
		return value < threshold
	case Comparison_LESS_THAN_EQUAL_TO:
		return value <= threshold
	}

	return false
}
