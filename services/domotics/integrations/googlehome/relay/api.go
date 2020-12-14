package relay

import (
	"encoding/json"
	"fmt"
	"io"

	action "github.com/rmrobinson/google-smart-home-action-go"
	"github.com/rmrobinson/nerves/services/domotics/integrations/googlehome"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrNoAgentRegistered is returned when a message is received on a stream which has not registered an agent.
	ErrNoAgentRegistered = status.New(codes.InvalidArgument, "no agent registered")
)

// API holds the implementation of the Google Home Service gRPC API
type API struct {
	logger *zap.Logger

	providerSvc *ProviderService

	actionSvc *action.Service
}

// NewAPI creates a new instance of the API
func NewAPI(logger *zap.Logger, ps *ProviderService, as *action.Service) *API {
	return &API{
		logger:      logger,
		providerSvc: ps,
		actionSvc:   as,
	}
}

// StateSync contains the bidirectional stream of requests coming in from a connected client.
func (a *API) StateSync(stream googlehome.GoogleHomeService_StateSyncServer) error {
	var provider *RemoteProvider
	for {
		reqMsg, err := stream.Recv()
		if err == io.EOF {
			if provider != nil {
				a.providerSvc.UnregisterProvider(provider.agentID)
			}

			return nil
		}
		if err != nil {
			a.logger.Error("error receiving from stream",
				zap.Error(err),
			)
			return err
		}

		if req := reqMsg.GetRegisterAgent(); req != nil {
			provider = NewRemoteProvider(a.logger, req.AgentId, stream)
			a.providerSvc.RegisterProvider(req.AgentId, provider)
		}

		if provider == nil {
			a.logger.Info("received command from stream before registered, resetting")
			return ErrNoAgentRegistered.Err()
		}

		// Operations received here fit into one of two categories:
		// - requests that can be directly relayed on to Google
		// - requests that need to be passed to the provider for handling

		if msg := reqMsg.GetRequestSync(); msg != nil {
			err := a.actionSvc.RequestSync(stream.Context(), provider.agentID)
			if err != nil {
				a.logger.Error("unable to request sync",
					zap.String("agent_id", provider.agentID),
					zap.Error(err),
				)
			}
		} else if msg := reqMsg.GetReportState(); msg != nil {
			deviceStates := map[string]action.DeviceState{}
			err := json.Unmarshal(msg.GetPayload(), &deviceStates)
			if err != nil {
				a.logger.Error("unable to deserialize device state",
					zap.String("agent_id", provider.agentID),
					zap.Error(err),
				)
				continue
			}
			err = a.actionSvc.ReportState(stream.Context(), provider.agentID, deviceStates)
			if err != nil {
				a.logger.Error("unable to report state",
					zap.String("agent_id", provider.agentID),
					zap.Error(err),
				)
			}
		} else if msg := reqMsg.GetCommandResponse(); msg != nil {
			err := provider.receiveResponse(msg)
			if err != nil {
				a.logger.Error("unable to process command response",
					zap.String("agent_id", provider.agentID),
					zap.Error(err),
				)
			}
		} else {
			a.logger.Info("received unhandled type from stream",
				zap.String("agent_id", provider.agentID),
				zap.String("type", fmt.Sprintf("%T", reqMsg.GetField())),
			)
		}
	}
}
