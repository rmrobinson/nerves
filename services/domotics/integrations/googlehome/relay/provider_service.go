package relay

import (
	"context"
	"errors"
	"sync"

	action "github.com/rmrobinson/google-smart-home-action-go"
	"go.uber.org/zap"
)

var (
	// ErrAgentNotFound is returned if the Google Assistant requested an action for a valid user who is not
	// registered with the system at the moment.
	ErrAgentNotFound = errors.New("agent not found")
)

// ProviderService satisfies the Google Smart Home Action Provider contract.
// This is called by the Google Assistant when users trigger requests against the HomeGraph.
// This maintains a map of the supported agent IDs and their registered handlers.
// Any request received that does not have a registered agent ID will fail.
type ProviderService struct {
	logger           *zap.Logger
	agentIDProviders map[string]action.Provider

	providersMutex sync.Mutex
}

// NewProviderService creates a new intent-handling ProviderService to register with the Google Assistant handling framework.
func NewProviderService(logger *zap.Logger) *ProviderService {
	return &ProviderService{
		logger:           logger,
		agentIDProviders: map[string]action.Provider{},
	}
}

// RegisterProvider adds a new agent ID to the set being handled.
func (s *ProviderService) RegisterProvider(agentID string, provider action.Provider) {
	s.providersMutex.Lock()
	defer s.providersMutex.Unlock()

	s.logger.Info("registering agent provider",
		zap.String("agent_id", agentID),
	)
	s.agentIDProviders[agentID] = provider
}

// UnregisterProvider removes the registered agent ID from the set of handled providers.
func (s *ProviderService) UnregisterProvider(agentID string) {
	s.providersMutex.Lock()
	defer s.providersMutex.Unlock()

	s.logger.Info("unregistering agent provider",
		zap.String("agent_id", agentID),
	)
	delete(s.agentIDProviders, agentID)
}

// Sync processes the Google Assistent request to retrieve the set of devices for the specified agent ID.
func (s *ProviderService) Sync(ctx context.Context, agentID string) (*action.SyncResponse, error) {
	if p, found := s.agentIDProviders[agentID]; found {
		return p.Sync(ctx, agentID)
	}

	return nil, ErrAgentNotFound
}

// Disconnect processes the Google Assistant request to remove updates from the specified agent ID.
func (s *ProviderService) Disconnect(ctx context.Context, agentID string) error {
	if p, found := s.agentIDProviders[agentID]; found {
		err := p.Disconnect(ctx, agentID)
		if err != nil {
			s.logger.Info("error unregistering agent ID",
				zap.String("agent_id", agentID),
				zap.Error(err),
			)
		}

		s.providersMutex.Lock()
		delete(s.agentIDProviders, agentID)
		s.providersMutex.Unlock()

		return nil
	}

	return ErrAgentNotFound
}

// Query processes the Google Assistent request for a state refresh for the specified devices for the specified agent.
func (s *ProviderService) Query(ctx context.Context, req *action.QueryRequest) (*action.QueryResponse, error) {
	if p, found := s.agentIDProviders[req.AgentID]; found {
		return p.Query(ctx, req)
	}

	return nil, ErrAgentNotFound
}

// Execute processes the Google Assistant request to change the state of the specified devices for the specified agent.
func (s *ProviderService) Execute(ctx context.Context, req *action.ExecuteRequest) (*action.ExecuteResponse, error) {
	if p, found := s.agentIDProviders[req.AgentID]; found {
		return p.Execute(ctx, req)
	}

	return nil, ErrAgentNotFound
}
