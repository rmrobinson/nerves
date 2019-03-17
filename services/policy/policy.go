package policy

import (
	"context"

	"go.uber.org/zap"
)

func (p *Policy) execute(ctx context.Context, logger *zap.Logger, state *State) {
	if !p.Condition.triggered(state) {
		logger.Debug("policy conditions not met",
			zap.String("name", p.Name),
		)
		return
	}

	logger.Debug("policy conditions met, executing actions")
	for _, action := range p.Actions {
		action.execute(ctx, logger, state)
	}
}
