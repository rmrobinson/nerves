package policy

func (c *Condition) triggered(state *State) bool {
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
			return true
		} else if c.Set.Operator == Condition_Set_AND && allTriggered {
			return true
		}

		return false
	}

	// TODO: add other conditions

	return false
}
