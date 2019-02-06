package mind

import (
	"context"

	"github.com/golang/protobuf/ptypes"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrStatementNotHandled is returned if a particular handler fails to process the statement
	ErrStatementNotHandled = status.New(codes.FailedPrecondition, "statement not handled")
	// ErrStatementIgnored is returned if no registered handlers chose to process the statement
	ErrStatementIgnored = status.New(codes.InvalidArgument, "statement not handled")
)

// Handler describes an implementation to process statements and potentially take actions on th
type Handler interface {
	ProcessStatement(context.Context, *Statement) (*Statement, error)
}

// Service is a messaging service.
type Service struct {
	logger *zap.Logger
	users  map[string]*User

	handlers []Handler
}

// NewService creates a new messaging service.
func NewService(logger *zap.Logger) *Service {
	return &Service{
		logger: logger,
		users:  map[string]*User{},
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
func (s *Service) SendStatement(ctx context.Context, req *SendStatementRequest) (*Statement, error) {
	for _, handler := range s.handlers {
		resp, err := handler.ProcessStatement(ctx, req.Statement)
		if err == ErrStatementNotHandled.Err() {
			continue
		}

		return resp, nil
	}

	return nil, ErrStatementIgnored.Err()
}

// ReceiveStatements is used to broadcast info a receiver.
func (s *Service) ReceiveStatements(*ReceiveStatementsRequest, MessageService_ReceiveStatementsServer) error {
	return nil
}

func statementFromText(content string) *Statement {
	return &Statement{
		MimeType: "text/plain",
		Content:  []byte(content),
		CreateAt: ptypes.TimestampNow(),
	}
}
