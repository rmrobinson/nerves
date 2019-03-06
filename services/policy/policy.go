package policy

import (
	"go.uber.org/zap"
)

func (p *Policy) execute(logger *zap.Logger, state *State) {
	if !p.Condition.triggered(state) {
		logger.Debug("policy conditions not met",
			zap.String("name", p.Name),
		)
		return
	}

	logger.Debug("policy conditions met, executing actions")
	for _, action := range p.Actions {
		action.execute(logger, state)
	}
}
