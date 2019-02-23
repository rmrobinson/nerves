package mind

import (
	"context"

	"github.com/golang/protobuf/ptypes"
	"github.com/rmrobinson/nerves/services/users"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	mimeTypeText = "text/plain"
)

var (
	// ErrContentTypeNotSupported is returned if the specified content type isn't supported
	ErrContentTypeNotSupported = status.New(codes.InvalidArgument, "content type not supported")
	// ErrStatementNotHandled is returned if a particular handler fails to process the statement
	ErrStatementNotHandled = status.New(codes.FailedPrecondition, "statement not handled")
	// ErrStatementIgnored is returned if no registered handlers chose to process the statement
	ErrStatementIgnored = status.New(codes.InvalidArgument, "statement not handled")
	// ErrStatementDisallowed is returned if the requesting user cannot make this statement
	ErrStatementDisallowed = status.New(codes.PermissionDenied, "statement user cannot make this request")
)

// Handler describes an implementation to process statements and potentially take actions on them
type Handler interface {
	ProcessStatement(context.Context, *SendStatementRequest) (*Statement, error)
}

// Channel represents a public or private shared communication location.
type Channel interface {
	SendStatement(context.Context, *Statement) error
}

// Service is a messaging service.
type Service struct {
	logger *zap.Logger
	users  map[string]*users.User

	handlers []Handler

	channels []Channel
}

// NewService creates a new messaging service.
func NewService(logger *zap.Logger, users map[string]*users.User) *Service {
	return &Service{
		logger: logger,
		users:  users,
	}
}

// BroadcastUpdate sends a specified statement update to all registered handlers.
func (s *Service) BroadcastUpdate(ctx context.Context, statement *Statement) error {
	for _, channel := range s.channels {
		err := channel.SendStatement(ctx, statement)
		if err != nil {
			s.logger.Info("error sending statement to channel",
				zap.Error(err),
			)

			return err
		}
	}

	return nil
}

// RegisterChannel adds another implementation to the broadcast chain.
func (s *Service) RegisterChannel(c Channel) {
	s.channels = append(s.channels, c)
}

// RegisterHandler adds another implementation to the call chain.
func (s *Service) RegisterHandler(handler Handler) {
	s.handlers = append(s.handlers, handler)
}

// RegisterUser adds another user/endpoint mapping.
func (s *Service) RegisterUser(context.Context, *RegisterUserRequest) (*users.User, error) {
	return nil, nil
}

// SendStatement takes a supplied statement and passes it into the handler chain.
func (s *Service) SendStatement(ctx context.Context, req *SendStatementRequest) (*Statement, error) {
	for _, handler := range s.handlers {
		resp, err := handler.ProcessStatement(ctx, req)
		if err == ErrStatementNotHandled.Err() {
			continue
		} else if err != nil {
			return statementFromText(err.Error()), nil
		}

		return resp, nil
	}

	return statementFromText(ErrStatementIgnored.Message()), nil
}

// ReceiveStatements is used to broadcast info a receiver.
func (s *Service) ReceiveStatements(*ReceiveStatementsRequest, MessageService_ReceiveStatementsServer) error {
	return nil
}

func statementFromText(content string) *Statement {
	return &Statement{
		MimeType: mimeTypeText,
		Content:  []byte(content),
		CreateAt: ptypes.TimestampNow(),
	}
}
