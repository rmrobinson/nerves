package main

import (
	"context"
	"flag"
	"sync"
	"time"

	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func getDevices(logger *zap.Logger, h *bridge.Hub) {
	devices, err := h.ListDevices()
	if err != nil {
		logger.Warn("unable to list devices",
			zap.Error(err),
		)
		return
	}

	var ret []string
	for _, device := range devices {
		ret = append(ret, device.String())
	}
	logger.Info("got devices",
		zap.Strings("devices", ret),
	)
}

func setDeviceName(logger *zap.Logger, h *bridge.Hub, id string, name string) {
	setResp, err := h.UpdateDeviceConfig(context.Background(), id, &bridge.DeviceConfig{
		Name:        name,
		Description: "Manually set",
	})
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

func setDeviceIsOn(logger *zap.Logger, h *bridge.Hub, id string, isOn bool) {
	d, err := h.GetDevice(id)
	if err != nil {
		logger.Warn("unable to get device",
			zap.String("device_id", id),
			zap.Error(err),
		)
		return
	}

	d.State.Binary.IsOn = isOn

	setResp, err := h.UpdateDeviceState(context.Background(), id, d.State)
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

func monitorHub(logger *zap.Logger, h *bridge.Hub) {
	updates := h.Updates()
	for {
		u, ok := <-updates

		if !ok {
			logger.Info("update stream closed")
			return
		}

		if u.GetBridgeUpdate() != nil {
			logger.Info("bridge updated",
				zap.String("action", u.Action.String()),
				zap.String("bridge_info", u.GetBridgeUpdate().Bridge.String()),
			)
		} else if u.GetDeviceUpdate() != nil {
			logger.Info("device updated",
				zap.String("action", u.Action.String()),
				zap.String("bridge_id", u.GetDeviceUpdate().BridgeId),
				zap.String("device_info", u.GetDeviceUpdate().Device.String()),
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

	h := bridge.NewHub(logger)
	h.AddBridge(bridgeClient)

	time.Sleep(time.Second)

	switch *mode {
	case "getDevices":
		getDevices(logger, h)
	case "setDeviceConfig":
		setDeviceName(logger, h, *id, *name)
	case "setDeviceState":
		setDeviceIsOn(logger, h, *id, *on)
	case "monitor":
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorHub(logger, h)
		}()

		wg.Wait()
	default:
		logger.Debug("unknown command specified")
	}

	conn.Close()
}
