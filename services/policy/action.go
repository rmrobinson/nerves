package policy

import (
	"go.uber.org/zap"
)

func (a *Action) execute(logger *zap.Logger, state *State) {
	switch a.Type {
	case Action_Log:
		logger.Info("executing action",
			zap.String("name", a.Name),
		)
	}
}
