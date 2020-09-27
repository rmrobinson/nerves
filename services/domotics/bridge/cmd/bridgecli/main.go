package main

import (
	"context"
	"flag"
	"io"
	"sync"

	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func getBridge(logger *zap.Logger, bc bridge.BridgeServiceClient) {
	b, err := bc.GetBridge(context.Background(), &bridge.GetBridgeRequest{})
	if err != nil {
		logger.Warn("unable to get bridge",
			zap.Error(err),
		)
		return
	}

	logger.Info("got bridge",
		zap.String("bridge_name", b.Config.Name),
	)
}

func getDevices(logger *zap.Logger, dc bridge.BridgeServiceClient) {
	getResp, err := dc.ListDevices(context.Background(), &bridge.ListDevicesRequest{})
	if err != nil {
		logger.Warn("unable to list devices",
			zap.Error(err),
		)
		return
	}

	var ret []string
	for _, device := range getResp.Devices {
		ret = append(ret, device.String())
	}
	logger.Info("got devices",
		zap.Strings("devices", ret),
	)
}

func setDeviceName(logger *zap.Logger, dc bridge.BridgeServiceClient, id string, name string) {
	req := &bridge.UpdateDeviceConfigRequest{
		Id: id,
		Config: &bridge.DeviceConfig{
			Name:        name,
			Description: "Manually set",
		},
	}

	setResp, err := dc.UpdateDeviceConfig(context.Background(), req)
	if err != nil {
		logger.Warn("unable to set device name",
			zap.String("device_id", id),
			zap.String("device_name", name),
			zap.Error(err),
		)
		return
	}

	logger.Info("set device name",
		zap.String("device_id", id),
		zap.String("device_name", name),
		zap.String("result", setResp.String()),
	)
}

func setDeviceIsOn(logger *zap.Logger, dc bridge.BridgeServiceClient, id string, isOn bool) {
	d, err := dc.GetDevice(context.Background(), &bridge.GetDeviceRequest{Id: id})
	if err != nil {
		logger.Warn("unable to get device",
			zap.String("device_id", id),
			zap.Error(err),
		)
		return
	}

	d.State.Binary.IsOn = isOn
	req := &bridge.UpdateDeviceStateRequest{
		Id:    id,
		State: d.State,
	}

	setResp, err := dc.UpdateDeviceState(context.Background(), req)
	if err != nil {
		logger.Warn("unable to set device",
			zap.String("device_id", id),
			zap.Error(err),
		)
		return
	}

	logger.Info("set device",
		zap.String("device_id", id),
		zap.Bool("is_on", d.State.Binary.IsOn),
		zap.String("result", setResp.String()),
	)
}

func monitorBridge(logger *zap.Logger, bc bridge.BridgeServiceClient) {
	stream, err := bc.StreamBridgeUpdates(context.Background(), &bridge.StreamBridgeUpdatesRequest{})
	if err != nil {
		return
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			logger.Warn("error watching bridge",
				zap.Error(err),
			)
			break
		}

		if msg.GetBridgeUpdate() != nil {
			logger.Info("bridge updated",
				zap.String("action", msg.Action.String()),
				zap.String("bridge_info", msg.GetBridgeUpdate().Bridge.String()),
			)
		} else if msg.GetDeviceUpdate() != nil {
			logger.Info("device updated",
				zap.String("action", msg.Action.String()),
				zap.String("bridge_id", msg.GetDeviceUpdate().BridgeId),
				zap.String("device_info", msg.GetDeviceUpdate().Device.String()),
			)
		}
	}
}

func main() {
	var (
		addr = flag.String("addr", "", "The address to connect to")
		mode = flag.String("mode", "", "The mode of operation for the client")
		id   = flag.String("id", "", "The device ID to change")
		name = flag.String("name", "", "The device name to set")
		on   = flag.Bool("on", false, "The device ison state to set")
	)

	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(*addr, opts...)
	if err != nil {
		logger.Fatal("unable to connect",
			zap.String("addr", *addr),
			zap.Error(err),
		)
		return
	}

	bridgeClient := bridge.NewBridgeServiceClient(conn)

	switch *mode {
	case "getBridge":
		getBridge(logger, bridgeClient)
	case "getDevices":
		getDevices(logger, bridgeClient)
	case "setDeviceConfig":
		setDeviceName(logger, bridgeClient, *id, *name)
	case "setDeviceState":
		setDeviceIsOn(logger, bridgeClient, *id, *on)
	case "monitor":
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorBridge(logger, bridgeClient)
		}()

		wg.Wait()
	default:
		logger.Debug("unknown command specified")
	}
}
