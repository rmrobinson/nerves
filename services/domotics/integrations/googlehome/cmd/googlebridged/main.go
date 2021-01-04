package main

import (
	"flag"

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

	h := bridge.NewHub(logger)
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

	p := NewProxy(logger, h, relayClient, *agentID)
	err = p.Run()
	if err != nil {
		logger.Fatal("unable to run proxy",
			zap.Error(err),
		)
	}

	relayConn.Close()
	bridgeConn.Close()
}
