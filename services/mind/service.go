package mind

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrStatementNotHandled = status.New(codes.FailedPrecondition, "statement not handled")
	ErrStatementIgnored = status.New(codes.InvalidArgument, "statement not handled")
)

type Handler interface {
	ProcessStatement(context.Context, *Statement) (*Statement, error)
}

// Service is a messaging service.
type Service struct {
	logger *zap.Logger
	users map[string]*User

	handlers []Handler
}

// NewService creates a new messaging service.
func NewService(logger *zap.Logger) *Service{
	return &Service{
		logger: logger,
		users: map[string]*User{},
	}
}

// RegisterHandler adds another implementation to the call chain.
func (s *Service) RegisterHandler(handler Handler) {
	s.handlers = append(s.handlers, handler)
}

// RegisterUser adds another user/endpoint mapping.
func (s *Service) RegisterUser(context.Context, *RegisterUserRequest) (*User, error) {
	return nil, nil
}

// SendStatement takes a supplied statement and passes it into the handler chain.
func (s *Service) SendStatement(ctx context.Context, req *SendStatementRequest) (*empty.Empty, error) {
	for _, handler := range s.handlers {
		_, err := handler.ProcessStatement(ctx, req.Statement)
		if err == ErrStatementNotHandled.Err() {
			continue
		}

		return nil, nil
	}

	return nil, ErrStatementNotHandled.Err()
}

// ReceiveStatements is used to broadcast info a receiver.
func (s *Service) ReceiveStatements(*ReceiveStatementsRequest, MessageService_ReceiveStatementsServer) error {
	return nil
}
