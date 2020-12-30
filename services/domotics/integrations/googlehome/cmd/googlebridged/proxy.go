package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	action "github.com/rmrobinson/google-smart-home-action-go"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/rmrobinson/nerves/services/domotics/integrations/googlehome"
	"go.uber.org/zap"
)

// Proxy contains the logic to translate between the bridge & relay APIs
type Proxy struct {
	logger      *zap.Logger
	h           *bridge.Hub
	relayClient googlehome.GoogleHomeServiceClient
	agentID     string

	wait chan struct{}
}

// NewProxy creates a new proxy
func NewProxy(logger *zap.Logger, h *bridge.Hub, relayClient googlehome.GoogleHomeServiceClient, agentID string) *Proxy {
	return &Proxy{
		logger:      logger,
		h:           h,
		relayClient: relayClient,
		agentID:     agentID,
		wait:        make(chan struct{}),
	}
}

// Run connects to the hub, relay and begins translating between the two
func (p *Proxy) Run() error {
	relayStream, err := p.relayClient.StateSync(context.Background())
	if err != nil {
		p.logger.Error("unable to create state sync stream",
			zap.Error(err),
		)
		return err
	}

	p.logger.Info("waiting for hub to sync before registering with relay")
	time.Sleep(time.Second * 2)

	// We need to start by writing the agent ID request
	reqID := uuid.New().String()
	registerReq := &googlehome.ClientRequest{
		Field: &googlehome.ClientRequest_RegisterAgent{
			RegisterAgent: &googlehome.RegisterAgentRequest{
				AgentId: p.agentID,
			},
		},
		RequestId: reqID,
	}
	err = relayStream.Send(registerReq)
	if err != nil {
		p.logger.Error("unable to register agent id",
			zap.String("agent_id", p.agentID),
			zap.String("request_id", reqID),
			zap.Error(err),
		)
		return err
	}

	go p.processHubUpdates(relayStream)
	go p.processRelayUpdates(relayStream)

	<-p.wait

	return nil
}

func (p *Proxy) processHubUpdates(relayStream googlehome.GoogleHomeService_StateSyncClient) {
	// Listen to updates from the bridge and forward the relevant commands (RequestSync, ReportState) to the relay
	for {
		update := <-p.h.Updates()
		// Google doesn't care about bridge changes.
		if deviceUpdate := update.GetDeviceUpdate(); deviceUpdate != nil {
			p.logger.Debug("received update from bridge")

			// ADDED or REMOVED means we need to trigger a state sync
			// This will cause Google to re-poll us and see what devices exist.
			if update.Action == bridge.Update_ADDED || update.Action == bridge.Update_REMOVED {
				reqID := uuid.New().String()
				syncReq := &googlehome.ClientRequest{
					Field: &googlehome.ClientRequest_RequestSync{
						RequestSync: &googlehome.RequestSyncRequest{},
					},
					RequestId: reqID,
				}

				err := relayStream.Send(syncReq)
				if err != nil {
					p.logger.Error("unable to request sync",
						zap.String("request_id", reqID),
						zap.Error(err),
					)
					continue
				}

				p.logger.Debug("requested sync from google")
			} else if update.Action == bridge.Update_CHANGED {
				if deviceUpdate.GetDevice() == nil {
					p.logger.Info("updated device is nil, not supported",
						zap.String("bridge_id", deviceUpdate.BridgeId),
						zap.String("device_id", deviceUpdate.DeviceId),
					)
					continue
				}

				updatedState, err := bridgeToGoogleState(deviceUpdate.GetDevice())
				if err != nil {
					p.logger.Info("serialized state has error, not sending",
						zap.String("bridge_id", deviceUpdate.BridgeId),
						zap.String("device_id", deviceUpdate.DeviceId),
						zap.Error(err),
					)
					continue
				}

				googleHomeUpdate, err := json.Marshal(map[string]action.DeviceState{
					deviceUpdate.GetDevice().Id: updatedState,
				})
				if err != nil {
					p.logger.Error("unable to serialize google home device state to update",
						zap.Error(err),
					)
					continue
				}

				reqID := uuid.New().String()
				err = relayStream.Send(&googlehome.ClientRequest{
					Field: &googlehome.ClientRequest_ReportState{
						ReportState: &googlehome.ReportStateRequest{
							Payload: googleHomeUpdate,
						},
					},
					RequestId: reqID,
				})
				if err != nil {
					p.logger.Error("unable to report state",
						zap.String("request_id", reqID),
						zap.Error(err),
					)
					continue
				}

				p.logger.Debug("reported state update to google")
			}
		}
	}
}

func (p *Proxy) processSyncRequest() *googlehome.SyncResponse {
	syncResp := &googlehome.SyncResponse{}

	devices, err := p.h.ListDevices()
	if err != nil {
		p.logger.Info("unable to handle sync",
			zap.Error(err),
		)
		syncResp.ErrorDetails = err.Error()
	} else {
		var actionDevices []*action.Device
		for _, d := range devices {
			actionDevice := bridgeToGoogleDevice(d)
			if actionDevice == nil {
				p.logger.Info("device not supported",
					zap.String("device_id", d.Id),
				)
				continue
			}

			actionDevices = append(actionDevices, actionDevice)
		}

		bytes, err := json.Marshal(actionDevices)
		if err != nil {
			p.logger.Info("unable to serialize sync response devices",
				zap.Error(err),
			)
			syncResp.ErrorDetails = err.Error()
		}
		syncResp.Payload = bytes
	}

	return syncResp
}

func (p *Proxy) processQueryRequest(msg *googlehome.QueryRequest) *googlehome.QueryResponse {
	actionStates := map[string]action.DeviceState{}
	var queryErr error

	for _, deviceID := range msg.DeviceIds {
		device, err := p.h.GetDevice(deviceID)
		if err != nil {
			p.logger.Info("unable to get device",
				zap.String("device_id", deviceID),
				zap.Error(err),
			)
			queryErr = err
			break
		}

		state, err := bridgeToGoogleState(device)
		if err != nil {
			p.logger.Info("unable to serialize device",
				zap.String("device_id", deviceID),
				zap.Error(err),
			)
			queryErr = err
			break
		}

		actionStates[deviceID] = state
	}

	queryResp := &googlehome.QueryResponse{}
	if queryErr != nil {
		queryResp.ErrorDetails = queryErr.Error()
	} else {
		bytes, err := json.Marshal(actionStates)
		if err != nil {
			p.logger.Info("unable to serialize query response states",
				zap.Error(err),
			)
			queryResp.ErrorDetails = err.Error()
		}
		queryResp.Payload = bytes
	}

	return queryResp
}

func (p *Proxy) processExecuteRequest(msg *googlehome.ExecuteRequest) *googlehome.ExecuteResponse {
	missingDeviceIDs := []string{}
	failedDeviceIDs := []string{}
	successDeviceIDs := []string{}
	successState := action.DeviceState{}
	var execErr error

	for _, cmd := range msg.Commands {
		// Get the set of devices this command is going to act on
		devices := map[string]*bridge.Device{}
		for _, deviceID := range cmd.DeviceIds {
			d, err := p.h.GetDevice(deviceID)
			if err != nil {
				p.logger.Info("got request to set missing device",
					zap.String("device_id", deviceID),
					zap.Error(err),
				)

				missingDeviceIDs = append(missingDeviceIDs, deviceID)
				continue
			}

			devices[deviceID] = d
		}

		// For each execution context, update the state of all devices.
		// This is a non-triggering action, which allows us to 'build'
		// the desired new state through this loop.
		// The following step will actually cause the device states to 'update' and change.
		for _, execContext := range cmd.ExecutionContext {
			actionCmd := &action.Command{}
			err := json.Unmarshal(execContext.Payload, actionCmd)
			if err != nil {
				p.logger.Info("unable to deserialize device",
					zap.Strings("device_ids", cmd.DeviceIds),
					zap.Error(err),
				)
				execErr = err
				break
			}

			if actionCmd.OnOff != nil {
				for _, d := range devices {
					if d.State.Binary != nil {
						d.State.Binary.IsOn = actionCmd.OnOff.On
					}
				}
			} else if actionCmd.BrightnessAbsolute != nil {
				for _, d := range devices {
					if d.State.Range != nil {
						d.State.Range.Value = int32(actionCmd.BrightnessAbsolute.Brightness)
					}
				}
			} else if actionCmd.BrightnessRelative != nil {
				for _, d := range devices {
					if d.State.Range != nil {
						if actionCmd.BrightnessRelative.RelativePercent != 0 {
							d.State.Range.Value += (d.State.Range.Value * int32(actionCmd.BrightnessRelative.RelativePercent)) / 100
						} else if actionCmd.BrightnessRelative.RelativeWeight != 0 {
							d.State.Range.Value += int32(actionCmd.BrightnessRelative.RelativeWeight)
						}
					}
				}
			} else if actionCmd.ColorAbsolute != nil {
				for _, d := range devices {
					if d.State.ColorHsb != nil {
						d.State.ColorHsb.Brightness = int32(actionCmd.ColorAbsolute.HSV.Value)
						d.State.ColorHsb.Hue = int32(actionCmd.ColorAbsolute.HSV.Hue)
						d.State.ColorHsb.Saturation = int32(actionCmd.ColorAbsolute.HSV.Saturation)
					} else if d.State.ColorRgb != nil {
						d.State.ColorRgb.Red = (int32(actionCmd.ColorAbsolute.RGB) >> 16) & 0x0ff
						d.State.ColorRgb.Green = (int32(actionCmd.ColorAbsolute.RGB) >> 8) & 0x0ff
						d.State.ColorRgb.Blue = int32(actionCmd.ColorAbsolute.RGB) & 0x0ff
					}
				}
			}
			// TODO: support more commands
		}

		// Now that each device has its state set, let's attempt to apply it.
		for deviceID, d := range devices {
			upd, err := p.h.UpdateDeviceState(context.Background(), deviceID, d.State)
			if err != nil {
				p.logger.Info("error updating state",
					zap.String("device_id", deviceID),
					zap.Error(err),
				)
				failedDeviceIDs = append(failedDeviceIDs, deviceID)
				continue
			}

			successDeviceIDs = append(successDeviceIDs, deviceID)
			successState, err = bridgeToGoogleState(upd)
			if err != nil {
				p.logger.Info("error serializing updated state to google format",
					zap.String("device_id", deviceID),
					zap.Error(err),
				)
			}
			// NOTE: not really sure what to do at this point since the state
			// has already been set. This is likely rare.
		}
	}

	execResp := &googlehome.ExecuteResponse{}
	if execErr != nil {
		execResp.ErrorDetails = execErr.Error()
	} else {
		if len(successDeviceIDs) > 0 {
			cmdResp := &googlehome.ExecuteResponse_CommandResult{
				Status:    googlehome.ExecuteResponse_SUCCESS,
				DeviceIds: successDeviceIDs,
			}
			bytes, err := json.Marshal(successState)
			if err != nil {
				p.logger.Info("unable to serialize exec response state",
					zap.Error(err),
				)
				execResp.ErrorDetails = err.Error()
			}
			cmdResp.States = bytes

			execResp.Results = append(execResp.Results, cmdResp)
		}
		if len(missingDeviceIDs) > 0 {
			cmdResp := &googlehome.ExecuteResponse_CommandResult{
				Status:    googlehome.ExecuteResponse_OFFLINE,
				DeviceIds: missingDeviceIDs,
			}
			execResp.Results = append(execResp.Results, cmdResp)
		}
		if len(failedDeviceIDs) > 0 {
			cmdResp := &googlehome.ExecuteResponse_CommandResult{
				Status:    googlehome.ExecuteResponse_ERROR,
				DeviceIds: failedDeviceIDs,
				ErrorCode: "commandInsertFailed",
			}
			execResp.Results = append(execResp.Results, cmdResp)
		}
	}

	return execResp
}

func (p *Proxy) processRelayUpdates(relayStream googlehome.GoogleHomeService_StateSyncClient) {
	reqID := uuid.New().String()
	syncReq := &googlehome.ClientRequest{
		Field: &googlehome.ClientRequest_RequestSync{
			RequestSync: &googlehome.RequestSyncRequest{},
		},
		RequestId: reqID,
	}

	err := relayStream.Send(syncReq)
	if err != nil {
		p.logger.Error("unable to request initial sync",
			zap.String("request_id", reqID),
			zap.Error(err),
		)
		close(p.wait)
	}

	// Listen to updates from the relay and forward the relevant commands to the bridge
	for {
		reqMsg, err := relayStream.Recv()
		if err == io.EOF {
			p.logger.Info("remote side closed stream")
			close(p.wait)
			return
		} else if err != nil {
			p.logger.Error("failed to receive a message",
				zap.Error(err),
			)
			close(p.wait)
			return
		}

		if msg := reqMsg.GetDisconnectRequest(); msg != nil {
			p.logger.Info("received remote disconnect request")
			close(p.wait)
			return
		} else if msg := reqMsg.GetSyncRequest(); msg != nil {
			syncResp := p.processSyncRequest()

			err = relayStream.Send(&googlehome.ClientRequest{
				RequestId: reqMsg.RequestId,
				Field: &googlehome.ClientRequest_SyncResponse{
					SyncResponse: syncResp,
				},
			})
			if err != nil {
				p.logger.Error("unable to send sync response",
					zap.Error(err),
				)
				continue
			}
		} else if msg := reqMsg.GetQueryRequest(); msg != nil {
			queryResp := p.processQueryRequest(msg)

			err = relayStream.Send(&googlehome.ClientRequest{
				RequestId: reqMsg.RequestId,
				Field: &googlehome.ClientRequest_QueryResponse{
					QueryResponse: queryResp,
				},
			})
			if err != nil {
				p.logger.Error("unable to send query response",
					zap.Error(err),
				)
				continue
			}
		} else if msg := reqMsg.GetExecuteRequest(); msg != nil {
			execResp := p.processExecuteRequest(msg)

			err = relayStream.Send(&googlehome.ClientRequest{
				RequestId: reqMsg.RequestId,
				Field: &googlehome.ClientRequest_ExecuteResponse{
					ExecuteResponse: execResp,
				},
			})
			if err != nil {
				p.logger.Error("unable to send execute response",
					zap.Error(err),
				)
				continue
			}

		} else {
			p.logger.Info("received unhandled type from stream",
				zap.String("type", fmt.Sprintf("%T", reqMsg.GetField())),
			)
		}
	}
}

func bridgeToGoogleState(d *bridge.Device) (action.DeviceState, error) {
	ads := action.NewDeviceState(d.IsActive)

	if d.State == nil {
		return ads, errors.New("device state missing")
	}

	switch d.Type {
	case bridge.DeviceType_AV_RECEIVER:
		if d.State.Binary == nil || d.State.Input == nil || d.State.Audio == nil {
			return ads, errors.New("av receiver missing required attribute")
		}

		ads.RecordOnOff(d.State.Binary.IsOn)
		ads.RecordInput(d.State.Input.Input)
		ads.RecordVolume(int(d.State.Audio.Volume), d.State.Audio.IsMuted)
	case bridge.DeviceType_LIGHT:
		if d.State.Binary == nil {
			return ads, errors.New("light missing required attribute")
		}

		ads.RecordOnOff(d.State.Binary.IsOn)
		if d.State.Range != nil {
			ads.RecordBrightness(int(d.State.Range.Value))
		}
		if d.State.ColorHsb != nil {
			// nerves uses 'HSB' while Google uses 'HSV'. These are synonymous and neither are HSL (which is different).
			ads.RecordColorHSV(float64(d.State.ColorHsb.Hue), float64(d.State.ColorHsb.Saturation), float64(d.State.ColorHsb.Brightness))
		} else if d.State.ColorRgb != nil {
			rgb := ((d.State.ColorRgb.Red & 0x0ff) << 16) | ((d.State.ColorRgb.Green & 0x0ff) << 8) | (d.State.ColorRgb.Blue & 0x0ff)
			ads.RecordColorRGB(int(rgb))
		}
	case bridge.DeviceType_OUTLET:
		if d.State.Binary == nil {
			return ads, errors.New("outlet missing required attribute")
		}
		ads.RecordOnOff(d.State.Binary.IsOn)
	case bridge.DeviceType_SWITCH:
		if d.State.Binary == nil {
			return ads, errors.New("switch missing required attribute")
		}

		ads.RecordOnOff(d.State.Binary.IsOn)
		if d.State.Range != nil {
			ads.RecordBrightness(int(d.State.Range.Value))
		}
	}

	ads.Online = d.IsActive

	return ads, nil
}

func bridgeToGoogleDevice(d *bridge.Device) *action.Device {
	var ad *action.Device

	switch d.Type {
	case bridge.DeviceType_AV_RECEIVER:
		var inputs []action.DeviceInput

		if d.Input == nil {
			return nil
		}

		for _, i := range d.Input.Inputs {
			inputs = append(inputs, action.DeviceInput{
				Key: i,
			})
		}
		// Volume-based devices are already normalizing their volume between 0 and 100
		// Volume-based devices always support muting
		// Volume-based devices support state querying
		ad = action.NewSimpleAVReceiver(d.Id, inputs, 100, true, false)
	case bridge.DeviceType_LIGHT:
		ad = action.NewLight(d.Id)

		if d.State.Range != nil {
			ad.AddBrightnessTrait(false)
		}
		if d.State.ColorHsb != nil {
			ad.AddColourTrait(action.HSV, false)
		} else if d.State.ColorRgb != nil {
			ad.AddColourTrait(action.RGB, false)
		}
	case bridge.DeviceType_OUTLET:
		ad = action.NewOutlet(d.Id)
	case bridge.DeviceType_SWITCH:
		ad = action.NewSwitch(d.Id)
		if d.State.Range != nil {
			ad.AddBrightnessTrait(false)
		}
	default:
		return nil
	}

	ad.Name.Name = d.Config.Name
	ad.DeviceInfo.Model = d.ModelName
	ad.DeviceInfo.Manufacturer = d.Manufacturer
	if d.State.Version != nil {
		ad.DeviceInfo.HwVersion = d.State.Version.Hw
		ad.DeviceInfo.SwVersion = d.State.Version.Sw
	}
	ad.WillReportState = true

	return ad
}
