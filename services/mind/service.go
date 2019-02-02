package mind

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"go.uber.org/zap"
)

// Service is a messaging service.
type Service struct {
	logger *zap.Logger
	users map[string]*User
}

// NewService creates a new messaging service.
func NewService(logger *zap.Logger) *Service{
	return &Service{
		logger: logger,
		users: map[string]*User{},
	}
}

func (s *Service) RegisterUser(context.Context, *RegisterUserRequest) (*User, error) {
	return nil, nil
}

func (s *Service) SendStatement(ctx context.Context, req *SendStatementRequest) (*empty.Empty, error) {
	s.logger.Debug("Received statement",
		zap.String("name", req.Name),
		zap.String("message", string(req.Statement.Content)),
	)

	return nil, nil
}

func (s *Service) ReceiveStatements(*ReceiveStatementsRequest, MessageService_ReceiveStatementsServer) error {
	return nil
}
