package bridge

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/rmrobinson/hue-go"
	"github.com/rmrobinson/nerves/services/domotics"
)

var (
	// ErrHueAddressInvalid is returned if the supplied address of the Hue light is invalid
	ErrHueAddressInvalid = errors.New("hue address invalid")
	// ErrHueResponseError is returned if the Hue bridge returned an error
	ErrHueResponseError = errors.New("hue response error")
	// ErrDeviceLacksRangeCapability is returned if a range request is made to a device not supporting it
	ErrDeviceLacksRangeCapability = errors.New("invalid argument supplied: device lacks range capabilities")
	// ErrDeviceRangeLimitExceeded is returned if the range of a value exceeds the supported range of the underlying device.
	ErrDeviceRangeLimitExceeded = errors.New("invalid argument supplied: range value outside of allowed values")
	lightAddrPrefix             = "/light/"
	sensorAddrPrefix            = "/sensor/"
)

func addrToLight(addr string) int {
	id, err := strconv.ParseInt(strings.TrimPrefix(addr, lightAddrPrefix), 10, 32)
	if err != nil {
		return 0 // this is an invalid ID
	}
	return int(id)
}
func addrToSensor(addr string) int {
	id, err := strconv.ParseInt(strings.TrimPrefix(addr, sensorAddrPrefix), 10, 32)
	if err != nil {
		return 0 // this is an invalid ID
	}
	return int(id)
}

// Hue is an implementation of a bridge for the Friends of Hue system.
type Hue struct {
	bridge *hue.Bridge
}

// NewHue takes a previously set up Hue handle and exposes it as a Hue bridge.
func NewHue(bridge *hue.Bridge) *Hue {
	return &Hue{
		bridge: bridge,
	}
}

// Setup seeds the persistent store with the proper data
func (b *Hue) Setup(ctx context.Context) error {
	return nil
}

// Bridge retrieves the persisted state of the bridge from the backing store.
func (b *Hue) Bridge(ctx context.Context) (*domotics.Bridge, error) {
	desc, err := b.bridge.Description()
	if err != nil {
		return nil, err
	}
	config, err := b.bridge.Config()
	if err != nil {
		return nil, err
	}

	ret := &domotics.Bridge{
		Id:               config.ID,
		ModelId:          desc.Device.ModelNumber,
		ModelName:        desc.Device.ModelName,
		ModelDescription: desc.Device.ModelDescription,
		Manufacturer:     desc.Device.Manufacturer,
		Config: &domotics.BridgeConfig{
			Name: desc.Device.FriendlyName,
			Address: &domotics.Address{
				Ip: &domotics.Address_Ip{
					Host:    config.IPAddress,
					Netmask: config.SubnetMask,
					Gateway: config.GatewayAddress,
				},
			},
			Timezone: config.Timezone,
		},
		State: &domotics.BridgeState{
			Zigbee: &domotics.BridgeState_Zigbee{
				Channel: config.ZigbeeChannel,
			},
			Version: &domotics.BridgeState_Version{
				Sw:  config.SwVersion,
				Api: config.APIVersion,
			},
		},
	}

	for _, icon := range desc.Device.Icons {
		ret.IconUrl = append(ret.IconUrl, desc.URLBase+"/"+icon.FileName)
	}

	return ret, nil
}

// SetBridgeConfig persists the new bridge config on the Hue bridge.
// Only the name can currently be changed.
// TODO: support setting the static IP of the bridge.
func (b *Hue) SetBridgeConfig(ctx context.Context, config *domotics.BridgeConfig) error {
	updatedConfig := &hue.ConfigArg{}
	updatedConfig.SetName(config.Name)
	return b.bridge.SetConfig(updatedConfig)
}

// SetBridgeState persists the new bridge state on the Hue bridge.
func (b *Hue) SetBridgeState(ctx context.Context, state *domotics.BridgeState) error {
	return domotics.ErrOperationNotSupported
}

// SearchForAvailableDevices is a noop that returns immediately (nothing to search for).
func (b *Hue) SearchForAvailableDevices(context.Context) error {
	if err := b.bridge.SearchForNewLights(); err != nil {
		return err
	}
	if err := b.bridge.SearchForNewSensors(); err != nil {
		return err
	}
	return nil
}

// AvailableDevices returns an empty result as all devices are always available; never 'to be added'.
func (b *Hue) AvailableDevices(ctx context.Context) ([]*domotics.Device, error) {
	lights, err := b.bridge.NewLights()
	if err != nil {
		return nil, err
	}
	sensors, err := b.bridge.NewSensors()
	if err != nil {
		return nil, err
	}

	var devices []*domotics.Device

	for _, light := range lights {
		devices = append(devices, &domotics.Device{
			Address: lightAddrPrefix + light.ID,
			Config: &domotics.DeviceConfig{
				Name: light.Name,
			},
		})
	}
	for _, sensor := range sensors {
		devices = append(devices, &domotics.Device{
			Address: sensorAddrPrefix + sensor.ID,
			Config: &domotics.DeviceConfig{
				Name: sensor.Name,
			},
		})
	}

	return devices, nil
}

// Devices retrieves the list of lights and sensors from the bridge along with their current states.
func (b *Hue) Devices(ctx context.Context) ([]*domotics.Device, error) {
	lights, err := b.bridge.Lights()
	if err != nil {
		return nil, err
	}

	sensors, err := b.bridge.Sensors()
	if err != nil {
		return nil, err
	}

	var devices []*domotics.Device

	for _, light := range lights {
		d := convertLightToDevice(light)
		devices = append(devices, d)
	}
	for _, sensor := range sensors {
		d := convertSensorToDevice(sensor)
		devices = append(devices, d)
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Address < devices[j].Address
	})
	return devices, nil
}

// Device retrieves the specified device ID.
func (b *Hue) Device(ctx context.Context, id string) (*domotics.Device, error) {
	devices, err := b.Devices(ctx)
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.Id == id {
			return device, nil
		}
	}

	return nil, domotics.ErrDeviceNotFound.Err()
}

// SetDeviceConfig updates the bridge with the new config options for the light or sensor.
func (b *Hue) SetDeviceConfig(ctx context.Context, dev *domotics.Device, config *domotics.DeviceConfig) error {
	if strings.Contains(dev.Address, lightAddrPrefix) {
		id := addrToLight(dev.Address)
		args := convertLightDiffToArgs(dev, config)

		err := b.bridge.SetLight(fmt.Sprintf("%d", id), &args)
		if err != nil {
			return err
		} else if len(args.Errors()) > 0 {
			return ErrHueResponseError
		}
		return nil
	} else if strings.Contains(dev.Address, sensorAddrPrefix) {
		id := addrToSensor(dev.Address)
		args := convertSensorDiffToArgs(dev, config)

		err := b.bridge.SetSensor(fmt.Sprintf("%d", id), &args)
		if err != nil {
			return err
		} else if len(args.Errors()) > 0 {
			return ErrHueResponseError
		}
		return nil
	}

	return ErrHueAddressInvalid
}

// SetDeviceState updates the bridge with the new state options for the light (sensors aren't supported).
func (b *Hue) SetDeviceState(ctx context.Context, dev *domotics.Device, state *domotics.DeviceState) error {
	if strings.Contains(dev.Address, lightAddrPrefix) {
		id := addrToLight(dev.Address)
		args, err := convertLightStateDiffToArgs(dev, state)
		if err != nil {
			return err
		}

		err = b.bridge.SetLightState(fmt.Sprintf("%d", id), &args)
		if err != nil {
			return err
		} else if len(args.Errors()) > 0 {
			return ErrHueResponseError
		}
		return nil
	}

	return ErrHueAddressInvalid
}

// AddDevice is not implemented yet.
func (b *Hue) AddDevice(ctx context.Context, id string) error {
	return domotics.ErrNotImplemented.Err()
}

// DeleteDevice is not implemented yet.
func (b *Hue) DeleteDevice(ctx context.Context, id string) error {
	return domotics.ErrNotImplemented.Err()
}

func convertLightToDevice(l hue.Light) *domotics.Device {
	d := &domotics.Device{}
	d.Reset()

	d.Id = l.UniqueID
	d.Address = lightAddrPrefix + l.ID
	d.IsActive = true

	d.Manufacturer = l.ManufacturerName
	d.ModelId = l.ModelID

	config := &domotics.DeviceConfig{}
	d.Config = config

	config.Name = l.Name

	state := &domotics.DeviceState{}
	d.State = state

	state.IsReachable = l.State.Reachable
	state.Version = l.SwVersion

	state.Binary = &domotics.DeviceState_BinaryState{IsOn: l.State.On}

	if l.Model == "Dimmable light" ||
		l.Model == "Color light" ||
		l.Model == "Extended color light" {

		// We are hardcoded to only supporting a uint8 worth of brightness values
		d.Range = &domotics.Device_RangeDevice{Minimum: 0, Maximum: 254}

		state.Range = &domotics.DeviceState_RangeState{Value: int32(l.State.Brightness)}
	}

	if l.Model == "Color light" ||
		l.Model == "Extended color light" ||
		l.Model == "Color temperature light" {
		if l.State.ColorMode == "xy" {
			xy := hue.XY{X: l.State.XY[0], Y: l.State.XY[1]}
			rgb := hue.RGB{}
			rgb.FromXY(xy, l.ModelID)
			state.ColorRgb = &domotics.DeviceState_RGBState{Red: int32(rgb.Red), Blue: int32(rgb.Blue), Green: int32(rgb.Green)}
		} else if l.State.ColorMode == "ct" {
			rgb := hue.RGB{}
			rgb.FromCT(l.State.ColorTemperature)
			state.ColorRgb = &domotics.DeviceState_RGBState{Red: int32(rgb.Red), Blue: int32(rgb.Blue), Green: int32(rgb.Green)}
		} else if l.State.ColorMode == "hs" {
			hsb := hue.HSB{Hue: l.State.Hue, Saturation: l.State.Saturation, Brightness: l.State.Brightness}
			rgb := hue.RGB{}
			rgb.FromHSB(hsb)
			state.ColorRgb = &domotics.DeviceState_RGBState{Red: int32(rgb.Red), Blue: int32(rgb.Blue), Green: int32(rgb.Green)}
		}
	}

	return d
}

func convertSensorToDevice(s hue.Sensor) *domotics.Device {
	d := &domotics.Device{}
	d.Reset()

	d.Id = s.UniqueID
	d.Address = sensorAddrPrefix + s.ID

	d.Manufacturer = s.ManufacturerName
	d.ModelId = s.ModelID

	config := &domotics.DeviceConfig{}
	d.Config = config

	config.Name = s.Name

	state := &domotics.DeviceState{}
	d.State = state

	state.IsReachable = s.Config.Reachable
	state.Version = s.SwVersion

	if s.Type == "ZGPSwitch" {
		button := &domotics.DeviceState_ButtonState{}
		button.IsOn = true

		switch s.State.ButtonEvent {
		case 34:
			button.Id = 1
		case 16:
			button.Id = 2
		case 17:
			button.Id = 3
		case 18:
			button.Id = 4
		}

		state.Button = append(state.Button, button)
	} else if s.Type == "ZLLSwitch" {
		button := &domotics.DeviceState_ButtonState{}

		switch s.State.ButtonEvent {
		case 1000, 1001, 1002, 1003:
			button.Id = 1
			button.IsOn = true
		case 2000, 2001, 2002, 2003:
			button.Id = 2
			button.IsOn = true
		case 3000, 3001, 3002, 3003:
			button.Id = 3
			button.IsOn = true
		case 4000, 4001, 4002, 4003:
			button.Id = 4
			button.IsOn = true
		}

		state.Button = append(state.Button, button)
	} else if s.Type == "ZLLPresence" {
		d.Range = &domotics.Device_RangeDevice{Minimum: 0, Maximum: s.SensitivityMax}
		state.Range = &domotics.DeviceState_RangeState{Value: s.Sensitivity}

		state.Presence = &domotics.DeviceState_PresenceState{IsPresent: s.State.Presence}
	} else if s.Type == "ZLLTemperature" {
		state.Temperature = &domotics.DeviceState_TemperatureState{TemperatureCelsius: s.State.Temperature / 100}
	}

	return d
}

func convertLightDiffToArgs(currDevice *domotics.Device, newConfig *domotics.DeviceConfig) hue.LightArg {
	var args hue.LightArg

	if currDevice.Config == nil && newConfig != nil {
		args.SetName(newConfig.Name)
	} else if currDevice.Config != nil && newConfig == nil {
		args.SetName("")
	} else if currDevice.Config.Name != newConfig.Name {
		args.SetName(newConfig.Name)
	}

	return args
}

func convertSensorDiffToArgs(currDevice *domotics.Device, newConfig *domotics.DeviceConfig) hue.SensorArg {
	var args hue.SensorArg

	if currDevice.Config == nil && newConfig != nil {
		args.SetName(newConfig.Name)
	} else if currDevice.Config != nil && newConfig == nil {
		args.SetName("")
	} else if currDevice.Config.Name != newConfig.Name {
		args.SetName(newConfig.Name)
	}

	return args
}

func convertLightStateDiffToArgs(currDevice *domotics.Device, newState *domotics.DeviceState) (args hue.LightStateArg, err error) {
	if currDevice.State == nil && newState != nil {
		if newState.Binary != nil {
			args.SetIsOn(newState.Binary.IsOn)
		}
		if newState.Range != nil {
			if currDevice.Range == nil {
				err = ErrDeviceLacksRangeCapability
				return
			} else if newState.Range.Value > currDevice.Range.Maximum || newState.Range.Value < currDevice.Range.Minimum {
				err = ErrDeviceRangeLimitExceeded
				return
			} else {
				args.SetBrightness(uint8(newState.Range.Value))
			}
		}
		if newState.ColorRgb != nil {
			colour := hue.RGB{
				Red:   uint8(newState.ColorRgb.Red),
				Green: uint8(newState.ColorRgb.Green),
				Blue:  uint8(newState.ColorRgb.Blue)}
			args.SetRGB(colour, currDevice.ModelId)
		}
	} else if currDevice.State != nil && newState == nil {
		args.SetIsOn(false)
	} else if currDevice.State != nil && newState != nil {
		if currDevice.State.Binary != nil && newState.Binary != nil && currDevice.State.Binary.IsOn != newState.Binary.IsOn {
			args.SetIsOn(newState.Binary.IsOn)
		}
		if newState.Range != nil {
			if currDevice.Range == nil {
				err = ErrDeviceLacksRangeCapability
				return
			} else if newState.Range.Value > currDevice.Range.Maximum || newState.Range.Value < currDevice.Range.Minimum {
				err = ErrDeviceRangeLimitExceeded
				return
			} else if currDevice.State.Range.Value != newState.Range.Value {
				args.SetBrightness(uint8(newState.Range.Value))
			}
		}
		if newState.ColorRgb != nil {
			if currDevice.State.ColorRgb == nil ||
				(currDevice.State.ColorRgb.Red != newState.ColorRgb.Red || currDevice.State.ColorRgb.Green != newState.ColorRgb.Green || currDevice.State.ColorRgb.Blue != newState.ColorRgb.Blue) {
				colour := hue.RGB{
					Red:   uint8(newState.ColorRgb.Red),
					Green: uint8(newState.ColorRgb.Green),
					Blue:  uint8(newState.ColorRgb.Blue)}
				args.SetRGB(colour, currDevice.ModelId)
			}
		}
	}

	return
}
