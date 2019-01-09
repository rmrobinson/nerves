package main

import (
	"context"
	"flag"
	"io"
	"sync"

	"github.com/rmrobinson/nerves/services/domotics"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func getBridges(logger *zap.Logger, bc domotics.BridgeServiceClient) {
	getResp, err := bc.ListBridges(context.Background(), &domotics.ListBridgesRequest{})
	if err != nil {
		logger.Warn("unable to list bridges",
			zap.Error(err),
		)
		return
	}

	var ret []string
	for _, bridge := range getResp.Bridges {
		ret = append(ret, bridge.String())
	}
	logger.Info("got bridges",
		zap.Strings("bridges", ret),
	)
}

func setBridgeName(logger *zap.Logger, bc domotics.BridgeServiceClient, id string, name string) {
	req := &domotics.SetBridgeConfigRequest{
		Id: id,
		Config: &domotics.BridgeConfig{
			Name: name,
		},
	}

	setResp, err := bc.SetBridgeConfig(context.Background(), req)
	if err != nil {
		logger.Warn("unable to set bridge name",
			zap.String("bridge_id", id),
			zap.String("bridge_name", name),
			zap.Error(err),
		)
		return
	}

	logger.Info("set bridge name",
		zap.String("bridge_id", id),
		zap.String("bridge_name", name),
		zap.String("result", setResp.Bridge.String()),
	)
}

func getDevices(logger *zap.Logger, dc domotics.DeviceServiceClient) {
	getResp, err := dc.ListDevices(context.Background(), &domotics.ListDevicesRequest{})
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

func setDeviceName(logger *zap.Logger, dc domotics.DeviceServiceClient, id string, name string) {
	req := &domotics.SetDeviceConfigRequest{
		Id: id,
		Config: &domotics.DeviceConfig{
			Name:        name,
			Description: "Manually set",
		},
	}

	setResp, err := dc.SetDeviceConfig(context.Background(), req)
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
		zap.String("result", setResp.Device.String()),
	)
}

func setDeviceIsOn(logger *zap.Logger, dc domotics.DeviceServiceClient, id string, isOn bool) {
	d, err := dc.GetDevice(context.Background(), &domotics.GetDeviceRequest{Id: id})
	if err != nil {
		logger.Warn("unable to get device",
			zap.String("device_id", id),
			zap.Error(err),
		)
		return
	}

	d.Device.State.Binary.IsOn = isOn
	req := &domotics.SetDeviceStateRequest{
		Id:    id,
		State: d.Device.State,
	}

	setResp, err := dc.SetDeviceState(context.Background(), req)
	if err != nil {
		logger.Warn("unable to set device",
			zap.String("device_id", id),
			zap.Error(err),
		)
		return
	}

	logger.Info("set device",
		zap.String("device_id", id),
		zap.Bool("is_on", d.Device.State.Binary.IsOn),
		zap.String("result", setResp.Device.String()),
	)
}

func monitorBridges(logger *zap.Logger, bc domotics.BridgeServiceClient) {
	stream, err := bc.StreamBridgeUpdates(context.Background(), &domotics.StreamBridgeUpdatesRequest{})
	if err != nil {
		return
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			logger.Warn("error watching bridges",
				zap.Error(err),
			)
			break
		}

		logger.Info("device change",
			zap.String("change", msg.Action.String()),
			zap.String("bridge_info", msg.Bridge.String()),
		)
	}
}
func monitorDevices(logger *zap.Logger, dc domotics.DeviceServiceClient) {
	stream, err := dc.StreamDeviceUpdates(context.Background(), &domotics.StreamDeviceUpdatesRequest{})
	if err != nil {
		return
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			logger.Warn("error watching devices",
				zap.Error(err),
			)
			break
		}

		logger.Info("device change",
			zap.String("change", msg.Action.String()),
			zap.String("device_info", msg.Device.String()),
		)
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

	bridgeClient := domotics.NewBridgeServiceClient(conn)
	deviceClient := domotics.NewDeviceServiceClient(conn)

	switch *mode {
	case "getBridges":
		getBridges(logger, bridgeClient)
	case "setBridgeConfig":
		setBridgeName(logger, bridgeClient, *id, *name)
	case "getDevices":
		getDevices(logger, deviceClient)
	case "setDeviceConfig":
		setDeviceName(logger, deviceClient, *id, *name)
	case "setDeviceState":
		setDeviceIsOn(logger, deviceClient, *id, *on)
	case "monitor":
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorBridges(logger, bridgeClient)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorDevices(logger, deviceClient)
		}()
		wg.Wait()
	default:
		logger.Debug("unknown command specified")
	}
}
