package relay

import (
	"context"
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
)

// RemoteProvider implements a Google Assistent provider that relays requests over a GoogleHomeService streaming RPC.
type RemoteProvider struct {
	logger  *zap.Logger
	agentID string

	stream googlehome.GoogleHomeService_StateSyncServer

	outstandingReqs      map[string](chan interface{})
	outstandingReqsMutex sync.Mutex
}

// NewRemoteProvider creates a new provider backed by a GoogleHomeService streaming RPC.
func NewRemoteProvider(logger *zap.Logger, agentID string, stream googlehome.GoogleHomeService_StateSyncServer) *RemoteProvider {
	return &RemoteProvider{
		logger:  logger,
		agentID: agentID,
		stream:  stream,
	}
}

func (m *RemoteProvider) receiveResponse(commandResp *googlehome.CommandResponse) error {
	m.outstandingReqsMutex.Lock()
	defer m.outstandingReqsMutex.Unlock()

	if req, ok := m.outstandingReqs[commandResp.RequestId]; ok {
		req <- commandResp
		return nil
	}

	m.logger.Info("received response for request ID no longer present",
		zap.String("request_id", commandResp.RequestId),
	)
	return ErrRequestMissing
}

func (m *RemoteProvider) doRequest(commandReq *googlehome.CommandRequest) ([]byte, error) {
	reqID := uuid.New().String()
	commandReq.RequestId = reqID

	serverReq := &googlehome.ServerRequest{
		Field: &googlehome.ServerRequest_CommandRequest{
			CommandRequest: commandReq,
		},
	}
	if err := m.stream.Send(serverReq); err != nil {
		m.logger.Error("failed to send command request",
			zap.String("request_id", reqID),
			zap.Error(err),
		)

		return nil, err
	}

	// Serialize the request to a proto message
	// Send the request over the stream
	// Create a response channel
	respData := make(chan interface{})
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
	case resp := <-respData:
		// Do stuff
		commandResp, ok := resp.(*googlehome.CommandResponse)
		if !ok {
			return nil, ErrUnsupportedCommandResponseReceived
		} else if commandResp.GetRequestId() != reqID {
			m.logger.Fatal("mismatched request id")
		}
		return commandResp.GetPayload(), nil
	case <-time.After(requestTimeout):
		return nil, ErrCommandTimeout
	}
}

// Sync returns the set of known devices.
func (m *RemoteProvider) Sync(context.Context, string) (*action.SyncResponse, error) {
	m.logger.Debug("sync")

	resp := &action.SyncResponse{}

	return resp, nil
}

// Disconnect removes this agent ID provider.
func (m *RemoteProvider) Disconnect(context.Context, string) error {
	m.logger.Debug("disconnect")
	return nil
}

// Query retrieves the requested device data
func (m *RemoteProvider) Query(_ context.Context, req *action.QueryRequest) (*action.QueryResponse, error) {
	m.logger.Debug("query")

	resp := &action.QueryResponse{
		States: map[string]action.DeviceState{},
	}

	return resp, nil
}

// Execute makes the specified devices change state.
func (m *RemoteProvider) Execute(_ context.Context, req *action.ExecuteRequest) (*action.ExecuteResponse, error) {
	m.logger.Debug("execute")

	resp := &action.ExecuteResponse{
		UpdatedState: action.NewDeviceState(true),
	}

	return resp, nil
}
