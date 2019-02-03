package mind

import (
	"context"
	"strings"

	"go.uber.org/zap"
)

// Echo is an echo handler
type Echo struct {
	logger *zap.Logger
}

// NewEcho creates a new echo handler
func NewEcho(logger *zap.Logger) *Echo {
	return &Echo{
		logger: logger,
	}
}

// ProcessStatement implements the handler interface. Logs and returns the statement.
func (e *Echo) ProcessStatement(ctx context.Context, stmt *Statement) (*Statement, error) {
	if stmt.MimeType != "text/plain" {
		return nil, ErrStatementNotHandled.Err()
	}

	content := string(stmt.Content)
	if !strings.Contains(content, "echo") {
		return nil, ErrStatementNotHandled.Err()
	}
	
	e.logger.Debug("processed message",
		zap.String("content", content),
	)

	return stmt, nil
}
