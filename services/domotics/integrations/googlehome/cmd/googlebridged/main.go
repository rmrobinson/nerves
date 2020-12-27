package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	action "github.com/rmrobinson/google-smart-home-action-go"

	"github.com/google/uuid"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/rmrobinson/nerves/services/domotics/integrations/googlehome"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	var (
		bridgeAddr = flag.String("bridge-addr", "", "The bridge address to connect to")
		relayAddr  = flag.String("relay-addr", "", "The Google Home relay address to connect to")
		agentID    = flag.String("agent-id", "", "The ID of the Google Smart Home Agent to register as")
	)

	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	bridgeConn, err := grpc.Dial(*bridgeAddr, opts...)
	if err != nil {
		logger.Fatal("unable to connect to bridge",
			zap.String("addr", *bridgeAddr),
			zap.Error(err),
		)
		return
	}

	bridgeClient := bridge.NewBridgeServiceClient(bridgeConn)

	info := &bridge.Bridge{
		Id:           uuid.New().String(),
		ModelId:      "gb1",
		ModelName:    "googlebridged",
		Manufacturer: "Faltung Systems",
	}

	h := bridge.NewHub(logger, info)
	h.AddBridge(bridgeClient)

	relayConn, err := grpc.Dial(*relayAddr, opts...)
	if err != nil {
		logger.Fatal("unable to connect to relay",
			zap.String("addr", *relayAddr),
			zap.Error(err),
		)
		return
	}
	relayClient := googlehome.NewGoogleHomeServiceClient(relayConn)
	relayStream, err := relayClient.StateSync(context.Background())

	// TODO: listen for updates from Hub and propogate to relay
	go func() {
		for {
			update := <-h.Updates()
			// Google doesn't care about bridge changes.
			if deviceUpdate := update.GetDeviceUpdate(); deviceUpdate != nil {
				// ADDED or REMOVED means we need to trigger a state sync
				// This will cause Google to repoll us and see what devices exist.
				if update.Action == bridge.Update_ADDED || update.Action == bridge.Update_REMOVED {
					syncReq := &googlehome.ClientRequest{
						Field: &googlehome.ClientRequest_RequestSync{
							RequestSync: &googlehome.RequestSyncRequest{},
						},
					}

					err := relayStream.Send(syncReq)
					if err != nil {
						logger.Error("unable to request sync",
							zap.Error(err),
						)
						continue
					}
				} else if update.Action == bridge.Update_CHANGED {
					updatedDevice := bridgeToGoogleDevice(deviceUpdate.GetDevice())
					googleHomeUpdate, err := json.Marshal(updatedDevice)
					if err != nil {
						logger.Error("unable to serialize google home device to update",
							zap.Error(err),
						)
						continue
					}
					reportStateReq := &googlehome.ClientRequest{
						Field: &googlehome.ClientRequest_ReportState{
							ReportState: &googlehome.ReportStateRequest{
								Payload: googleHomeUpdate,
							},
						},
					}

					err = relayStream.Send(reportStateReq)
					if err != nil {
						logger.Error("unable to report state",
							zap.Error(err),
						)
						continue
					}
				}
			}
		}
	}()

	waitc := make(chan struct{})
	go func() {
		// We need to start by writing the agent ID request
		registerReq := &googlehome.ClientRequest{
			Field: &googlehome.ClientRequest_RegisterAgent{
				RegisterAgent: &googlehome.RegisterAgentRequest{
					AgentId: *agentID,
				},
			},
		}
		err := relayStream.Send(registerReq)
		if err != nil {
			logger.Error("unable to register agent id",
				zap.String("agent_id", *agentID),
				zap.Error(err),
			)
			return
		}

		for {
			reqMsg, err := relayStream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				logger.Fatal("failed to receive a message",
					zap.Error(err),
				)
			}

			if msg := reqMsg.GetCommandRequest(); msg != nil {
				// TODO: Decode message
			} else {
				logger.Info("received unhandled type from stream",
					zap.String("type", fmt.Sprintf("%T", reqMsg.GetField())),
				)
			}
		}
	}()

	<-waitc

	relayConn.Close()
	bridgeConn.Close()
}

func bridgeToGoogleDevice(d *bridge.Device) *action.Device {

}
