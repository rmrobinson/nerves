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
			continue
		}

		if provider == nil {
			a.logger.Info("received command from stream before registered, resetting")
			return ErrNoAgentRegistered.Err()
		}

		// Operations received here fit into one of two categories:
		// - requests that can be directly relayed on to Google
		// - requests that need to be passed to the provider for handling

		if msg := reqMsg.GetRequestSync(); msg != nil {
			// Sync executes synchronously, which means that we will never be able to process the stream message
			// with the results for the SYNC call made to the HTTP API.
			// Therefore we run this one call on a goroutine so we don't block the stream when the response comes in.
			// This isn't really an issue since we don't report the results back to the caller anyways.
			go func() {
				err := a.actionSvc.RequestSync(stream.Context(), provider.agentID)
				if err != nil {
					a.logger.Error("unable to request sync",
						zap.String("agent_id", provider.agentID),
						zap.Error(err),
					)
					return
				}
				a.logger.Debug("requested sync from google")
			}()
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
			var deviceIDs []string
			for deviceID := range deviceStates {
				deviceIDs = append(deviceIDs, deviceID)
			}
			err = a.actionSvc.ReportState(stream.Context(), provider.agentID, deviceStates)
			if err != nil {
				a.logger.Error("unable to report state",
					zap.String("agent_id", provider.agentID),
					zap.Strings("device_ids", deviceIDs),
					zap.Error(err),
				)
				continue
			}
			a.logger.Debug("reported state update to google")
		} else if reqMsg.GetField() != nil {
			// If we have handled all the client-specific cases we can assume any remaining messages with a field set
			// are response messages being sent from the client. Pass them to the provider for handling.
			err := provider.receiveResponse(reqMsg)
			if err != nil {
				a.logger.Error("unable to process generic response",
					zap.String("agent_id", provider.agentID),
					zap.Error(err),
				)
				continue
			}
			a.logger.Debug("processed message with field")
		} else {
			a.logger.Info("received unhandled type from stream",
				zap.String("agent_id", provider.agentID),
				zap.String("type", fmt.Sprintf("%T", reqMsg.GetField())),
			)
			continue
		}
	}
}
