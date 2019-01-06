package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/rmrobinson/nerves/services/domotics"
	"google.golang.org/grpc"
)

func getBridges(bc domotics.BridgeServiceClient) {
	getResp, err := bc.ListBridges(context.Background(), &domotics.ListBridgesRequest{})
	if err != nil {
		fmt.Printf("Unable to get bridges: %s\n", err.Error())
		return
	}

	fmt.Printf("Got bridges\n")
	for _, bridge := range getResp.Bridges {
		fmt.Printf("%+v\n", bridge)
	}
}

func setBridgeName(bc domotics.BridgeServiceClient, id string, name string) {
	req := &domotics.SetBridgeConfigRequest{
		Id: id,
		Config: &domotics.BridgeConfig{
			Name: name,
		},
	}

	setResp, err := bc.SetBridgeConfig(context.Background(), req)
	if err != nil {
		fmt.Printf("Unable to set bridge name: %s\n", err.Error())
		return
	}

	fmt.Printf("Set bridge name\n")
	fmt.Printf("%+v\n", setResp.Bridge)
}

func getDevices(dc domotics.DeviceServiceClient) {
	getResp, err := dc.ListDevices(context.Background(), &domotics.ListDevicesRequest{})
	if err != nil {
		fmt.Printf("Unable to get devices: %s\n", err.Error())
		return
	}

	fmt.Printf("Got devices\n")
	for _, device := range getResp.Devices {
		fmt.Printf("%+v\n", device)
	}
}

func setDeviceName(dc domotics.DeviceServiceClient, id string, name string) {
	req := &domotics.SetDeviceConfigRequest{
		Id: id,
		Config: &domotics.DeviceConfig{
			Name:        name,
			Description: "Manually set",
		},
	}

	setResp, err := dc.SetDeviceConfig(context.Background(), req)
	if err != nil {
		fmt.Printf("Unable to set device name: %s\n", err.Error())
		return
	}

	fmt.Printf("Set device name\n")
	fmt.Printf("%+v\n", setResp.Device)
}

func setDeviceIsOn(dc domotics.DeviceServiceClient, id string, isOn bool) {
	d, err := dc.GetDevice(context.Background(), &domotics.GetDeviceRequest{Id: id})

	if err != nil {
		fmt.Printf("Unable to get device: %s\n", err.Error())
		return
	}

	d.Device.State.Binary.IsOn = isOn
	req := &domotics.SetDeviceStateRequest{
		Id:    id,
		State: d.Device.State,
	}

	setResp, err := dc.SetDeviceState(context.Background(), req)
	if err != nil {
		fmt.Printf("Unable to set device state: %s\n", err.Error())
		return
	}

	fmt.Printf("Set device isOn\n")
	fmt.Printf("%+v\n", setResp.Device)
}

func monitorBridges(bc domotics.BridgeServiceClient) {
	stream, err := bc.StreamBridgeUpdates(context.Background(), &domotics.StreamBridgeUpdatesRequest{})

	if err != nil {
		return
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("Error while watching bridges: %v", err)
			break
		}

		log.Printf("Change: %v, Bridge: %+v\n", msg.Action, msg.Bridge)
	}
}
func monitorDevices(dc domotics.DeviceServiceClient) {
	stream, err := dc.StreamDeviceUpdates(context.Background(), &domotics.StreamDeviceUpdatesRequest{})

	if err != nil {
		return
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("Error while watching devices: %v", err)
			break
		}

		log.Printf("Change: %v, Device: %+v\n", msg.Action, msg.Device)
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

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(*addr, opts...)

	if err != nil {
		fmt.Printf("Unable to connect: %s\n", err.Error())
		return
	}

	bridgeClient := domotics.NewBridgeServiceClient(conn)
	deviceClient := domotics.NewDeviceServiceClient(conn)

	switch *mode {
	case "getBridges":
		getBridges(bridgeClient)
	case "setBridgeConfig":
		setBridgeName(bridgeClient, *id, *name)
	case "getDevices":
		getDevices(deviceClient)
	case "setDeviceConfig":
		setDeviceName(deviceClient, *id, *name)
	case "setDeviceState":
		setDeviceIsOn(deviceClient, *id, *on)
	case "monitor":
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorBridges(bridgeClient)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorDevices(deviceClient)
		}()
		wg.Wait()
	default:
		fmt.Printf("Unknown mode specified")
	}
}
