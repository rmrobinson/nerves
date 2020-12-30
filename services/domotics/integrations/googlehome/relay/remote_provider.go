package relay

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	action "github.com/rmrobinson/google-smart-home-action-go"
	"github.com/rmrobinson/nerves/services/domotics/integrations/googlehome"
	"go.uber.org/zap"
)

const (
	requestTimeout = time.Second * 5
)

var (
	// ErrUnsupportedCommandResponseReceived is returned if the matched up response isn't a command response type
	ErrUnsupportedCommandResponseReceived = errors.New("unsupported command response received")
	// ErrCommandTimeout is returned if the command isn't received in time
	ErrCommandTimeout = errors.New("command request timed out")
	// ErrRequestMissing is returned if the targetted request isn't pending
	ErrRequestMissing = errors.New("request missing")
	// ErrContextTimeout is returned if the command context times out
	ErrContextTimeout = errors.New("context timed out")
	// ErrCommandFailed is returned if we received a proper response but the far end couldn't process it
	ErrCommandFailed = errors.New("command failed")
)

// RemoteProvider implements a Google Assistent provider that relays requests over a GoogleHomeService streaming RPC.
type RemoteProvider struct {
	logger  *zap.Logger
	agentID string

	stream googlehome.GoogleHomeService_StateSyncServer

	outstandingReqs      map[string](chan *googlehome.ClientRequest)
	outstandingReqsMutex sync.Mutex
}

// NewRemoteProvider creates a new provider backed by a GoogleHomeService streaming RPC.
func NewRemoteProvider(logger *zap.Logger, agentID string, stream googlehome.GoogleHomeService_StateSyncServer) *RemoteProvider {
	return &RemoteProvider{
		logger:          logger,
		agentID:         agentID,
		stream:          stream,
		outstandingReqs: map[string](chan *googlehome.ClientRequest){},
	}
}

func (m *RemoteProvider) receiveResponse(commandResp *googlehome.ClientRequest) error {
	m.outstandingReqsMutex.Lock()
	defer m.outstandingReqsMutex.Unlock()

	if req, ok := m.outstandingReqs[commandResp.RequestId]; ok {
		req <- commandResp
		return nil
	}

	m.logger.Info("received response for request ID no longer present",
		zap.String("agent_id", m.agentID),
		zap.String("request_id", commandResp.RequestId),
	)
	return ErrRequestMissing
}

func (m *RemoteProvider) doRequest(ctx context.Context, serverReq *googlehome.ServerRequest) (*googlehome.ClientRequest, error) {
	reqID := uuid.New().String()
	serverReq.RequestId = reqID

	if err := m.stream.Send(serverReq); err != nil {
		m.logger.Error("failed to send command request",
			zap.String("agent_id", m.agentID),
			zap.String("request_id", reqID),
			zap.Error(err),
		)

		return nil, err
	}

	// Serialize the request to a proto message
	// Send the request over the stream
	// Create a response channel
	respData := make(chan *googlehome.ClientRequest)
	m.outstandingReqsMutex.Lock()
	m.outstandingReqs[reqID] = respData
	m.outstandingReqsMutex.Unlock()

	defer func(reqID string) {
		m.outstandingReqsMutex.Lock()
		delete(m.outstandingReqs, reqID)
		close(respData)
		m.outstandingReqsMutex.Unlock()
	}(reqID)

	select {
	case <-ctx.Done():
		m.logger.Error("context for remote request timed out",
			zap.String("agent_id", m.agentID),
		)
		return nil, ErrContextTimeout
	case resp := <-respData:
		// Do stuff
		if resp.GetRequestId() != reqID {
			m.logger.Error("mismatched request id",
				zap.String("agent_id", m.agentID),
			)
		} else if resp.GetField() == nil {
			m.logger.Error("command response missing a field entry",
				zap.String("agent_id", m.agentID),
			)
			return nil, ErrUnsupportedCommandResponseReceived
		}
		return resp, nil
	case <-time.After(requestTimeout):
		m.logger.Error("command timed out",
			zap.String("agent_id", m.agentID),
		)
		return nil, ErrCommandTimeout
	}
}

// Sync returns the set of known devices.
func (m *RemoteProvider) Sync(ctx context.Context, _ string) (*action.SyncResponse, error) {
	m.logger.Debug("sync",
		zap.String("agent_id", m.agentID),
	)

	resp, err := m.doRequest(ctx, &googlehome.ServerRequest{
		Field: &googlehome.ServerRequest_SyncRequest{},
	})
	if err != nil {
		m.logger.Info("error processing sync request",
			zap.String("agent_id", m.agentID),
			zap.Error(err),
		)
		return nil, err
	}

	var syncResp *googlehome.ClientRequest_SyncResponse
	var ok bool
	if syncResp, ok = resp.GetField().(*googlehome.ClientRequest_SyncResponse); !ok {
		m.logger.Info("received wrong type for sync response",
			zap.String("agent_id", m.agentID),
		)
		return nil, ErrUnsupportedCommandResponseReceived
	}

	if len(syncResp.SyncResponse.ErrorDetails) > 0 {
		m.logger.Info("far side encountered error processing sync request",
			zap.String("agent_id", m.agentID),
			zap.String("error_details", syncResp.SyncResponse.ErrorDetails),
		)
		return nil, ErrCommandFailed
	}

	actionResp := &action.SyncResponse{}
	err = json.Unmarshal(syncResp.SyncResponse.Payload, &actionResp.Devices)
	if err != nil {
		m.logger.Info("unable to unmarshal sync response",
			zap.String("agent_id", m.agentID),
			zap.Error(err),
		)
		return nil, ErrUnsupportedCommandResponseReceived
	}

	return actionResp, nil
}

// Disconnect removes this agent ID provider.
func (m *RemoteProvider) Disconnect(context.Context, string) error {
	m.logger.Debug("disconnect",
		zap.String("agent_id", m.agentID),
	)

	if err := m.stream.Send(&googlehome.ServerRequest{
		Field: &googlehome.ServerRequest_DisconnectRequest{},
	}); err != nil {
		m.logger.Error("failed to send command request",
			zap.String("agent_id", m.agentID),
			zap.Error(err),
		)

		return err
	}

	return nil
}

// Query retrieves the requested device data
func (m *RemoteProvider) Query(ctx context.Context, req *action.QueryRequest) (*action.QueryResponse, error) {
	m.logger.Debug("query",
		zap.String("agent_id", m.agentID),
	)

	deviceIDs := []string{}
	for _, deviceArg := range req.Devices {
		deviceIDs = append(deviceIDs, deviceArg.ID)
	}
	resp, err := m.doRequest(ctx, &googlehome.ServerRequest{
		Field: &googlehome.ServerRequest_QueryRequest{
			QueryRequest: &googlehome.QueryRequest{
				DeviceIds: deviceIDs,
			},
		},
	})
	if err != nil {
		m.logger.Info("error processing query request",
			zap.String("agent_id", m.agentID),
			zap.Error(err),
		)
		return nil, err
	}

	var queryResp *googlehome.ClientRequest_QueryResponse
	var ok bool
	if queryResp, ok = resp.GetField().(*googlehome.ClientRequest_QueryResponse); !ok {
		m.logger.Info("received wrong type for query response",
			zap.String("agent_id", m.agentID),
		)
		return nil, ErrUnsupportedCommandResponseReceived
	}

	if len(queryResp.QueryResponse.ErrorDetails) > 0 {
		m.logger.Info("far side encountered error processing query request",
			zap.String("agent_id", m.agentID),
			zap.String("error_details", queryResp.QueryResponse.ErrorDetails),
		)
		return nil, ErrCommandFailed
	}

	actionResp := &action.QueryResponse{
		States: map[string]action.DeviceState{},
	}
	err = json.Unmarshal(queryResp.QueryResponse.Payload, &actionResp.States)
	if err != nil {
		m.logger.Info("unable to unmarshal query response",
			zap.String("agent_id", m.agentID),
			zap.Error(err),
		)
		return nil, ErrUnsupportedCommandResponseReceived
	}

	return actionResp, nil
}

// Execute makes the specified devices change state.
func (m *RemoteProvider) Execute(ctx context.Context, req *action.ExecuteRequest) (*action.ExecuteResponse, error) {
	m.logger.Debug("execute",
		zap.String("agent_id", m.agentID),
	)

	execCommands := []*googlehome.ExecuteRequest_Command{}
	for _, commandArg := range req.Commands {
		execCommand := &googlehome.ExecuteRequest_Command{}
		for _, deviceArg := range commandArg.TargetDevices {
			execCommand.DeviceIds = append(execCommand.DeviceIds, deviceArg.ID)
		}
		for _, cmd := range commandArg.Commands {
			bytes, err := json.Marshal(cmd)
			if err != nil {
				m.logger.Info("error marshaling command",
					zap.String("agent_id", m.agentID),
					zap.Error(err),
				)
				return nil, err
			}

			execCommand.ExecutionContext = append(execCommand.ExecutionContext, &googlehome.ExecuteRequest_Command_ExecutionContext{
				Payload: bytes,
			})
		}

		execCommands = append(execCommands, execCommand)
	}
	resp, err := m.doRequest(ctx, &googlehome.ServerRequest{
		Field: &googlehome.ServerRequest_ExecuteRequest{
			ExecuteRequest: &googlehome.ExecuteRequest{
				Commands: execCommands,
			},
		},
	})
	if err != nil {
		m.logger.Info("error processing execute request",
			zap.String("agent_id", m.agentID),
			zap.Error(err),
		)
		return nil, err
	}

	var execResp *googlehome.ClientRequest_ExecuteResponse
	var ok bool
	if execResp, ok = resp.GetField().(*googlehome.ClientRequest_ExecuteResponse); !ok {
		m.logger.Info("received wrong type for execute response",
			zap.String("agent_id", m.agentID),
		)
		return nil, ErrUnsupportedCommandResponseReceived
	}

	if len(execResp.ExecuteResponse.ErrorDetails) > 0 {
		m.logger.Info("far side encountered error processing execute request",
			zap.String("agent_id", m.agentID),
			zap.String("error_details", execResp.ExecuteResponse.ErrorDetails),
		)
		return nil, ErrCommandFailed
	}

	actionResp := &action.ExecuteResponse{
		FailedDevices: map[string]struct {
			Devices []string
		}{},
	}
	for _, result := range execResp.ExecuteResponse.Results {
		switch result.Status {
		case googlehome.ExecuteResponse_SUCCESS:
			err = json.Unmarshal(result.States, &actionResp.UpdatedState)
			if err != nil {
				m.logger.Info("unable to unmarshal exec response",
					zap.String("agent_id", m.agentID),
					zap.Error(err),
				)
				return nil, ErrUnsupportedCommandResponseReceived
			}

			actionResp.UpdatedDevices = result.DeviceIds
		case googlehome.ExecuteResponse_OFFLINE:
			actionResp.OfflineDevices = append(actionResp.OfflineDevices, result.DeviceIds...)
		case googlehome.ExecuteResponse_ERROR:
			actionResp.FailedDevices[result.ErrorCode] = struct {
				Devices []string
			}{
				Devices: result.DeviceIds,
			}
		}
	}

	return actionResp, nil
}
