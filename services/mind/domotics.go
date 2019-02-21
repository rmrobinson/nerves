package mind

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/users"
	"go.uber.org/zap"
)

const (
	domoticsListBridgeRegex             = `what('| i)?s the bridge list`
	domoticsListDeviceRegex             = `what('| i)?s the house status`
	domoticsTurnDeviceIDOnOffRegex      = `turn (?P<deviceID>[[:graph:]]+) (?P<isOn>on|off)`
	domoticsTurnDeviceNameOnOffRegex    = `turn "(?P<deviceName>.*)" (?P<isOn>on|off)`
	domoticsChangeDeviceNameVolumeRegex = `turn "(?P<deviceName>.*)" (?P<volume>up|down)`
	domoticsSetDeviceNameVolumeRegex    = `set the volume o(n|f) "(?P<deviceName>.*)" to (?P<volume>[0-9]+)`
)

var (
	turnDeviceIDOnOffRegex      = regexp.MustCompile(domoticsTurnDeviceIDOnOffRegex)
	turnDeviceNameOnOffRegex    = regexp.MustCompile(domoticsTurnDeviceNameOnOffRegex)
	changeDeviceNameVolumeRegex = regexp.MustCompile(domoticsChangeDeviceNameVolumeRegex)
	setDeviceNameVolumeRegex    = regexp.MustCompile(domoticsSetDeviceNameVolumeRegex)
)

// Domotics is a device-request handler
type Domotics struct {
	logger *zap.Logger

	svc *Service

	bridgeClient domotics.BridgeServiceClient
	deviceClient domotics.DeviceServiceClient
}

// NewDomotics creates a new domotics handler
func NewDomotics(logger *zap.Logger, svc *Service, bridgeClient domotics.BridgeServiceClient, deviceClient domotics.DeviceServiceClient) *Domotics {
	return &Domotics{
		logger:       logger,
		svc:          svc,
		bridgeClient: bridgeClient,
		deviceClient: deviceClient,
	}
}

// Monitor is used to track changes to devices
func (d *Domotics) Monitor(ctx context.Context) {
	stream, err := d.deviceClient.StreamDeviceUpdates(ctx, &domotics.StreamDeviceUpdatesRequest{})
	if err != nil {
		d.logger.Info("error creating device update stream",
			zap.Error(err),
		)
		return
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			d.logger.Info("error receiving update",
				zap.Error(err),
			)
			return
		}

		stmt := statementFromDeviceUpdate(update)
		err = d.svc.BroadcastUpdate(ctx, stmt)
		if err != nil {
			d.logger.Info("error broadcasting update",
				zap.Error(err),
			)
		}
	}
}

// ProcessStatement implements the handler interface. Logs and returns the statement.
func (d *Domotics) ProcessStatement(ctx context.Context, stmt *Statement) (*Statement, error) {
	if stmt.MimeType != mimeTypeText {
		return nil, ErrStatementNotHandled.Err()
	}

	content := string(stmt.Content)
	content = strings.ToLower(content)
	if ok, _ := regexp.MatchString(domoticsListBridgeRegex, content); ok {
		return d.getBridges(), nil
	} else if ok, _ := regexp.MatchString(domoticsListDeviceRegex, content); ok {
		return d.getDevices(), nil
	} else if matched := turnDeviceIDOnOffRegex.FindStringSubmatch(content); matched != nil {
		params := map[string]string{}

		for idx, name := range turnDeviceIDOnOffRegex.SubexpNames() {
			if name != "" {
				params[name] = matched[idx]
			}
		}

		if params["isOn"] == "on" {
			return d.setDeviceIDIsOn(params["deviceID"], true), nil
		} else if params["isOn"] == "off" {
			return d.setDeviceIDIsOn(params["deviceID"], false), nil
		} else {
			return nil, ErrStatementNotHandled.Err()
		}
	} else if matched := turnDeviceNameOnOffRegex.FindStringSubmatch(content); matched != nil {
		params := map[string]string{}

		for idx, name := range turnDeviceNameOnOffRegex.SubexpNames() {
			if name != "" {
				params[name] = matched[idx]
			}
		}

		if params["isOn"] == "on" {
			return d.setDeviceNameIsOn(params["deviceName"], true), nil
		} else if params["isOn"] == "off" {
			return d.setDeviceNameIsOn(params["deviceName"], false), nil
		} else {
			return nil, ErrStatementNotHandled.Err()
		}
	} else if matched := changeDeviceNameVolumeRegex.FindStringSubmatch(content); matched != nil {
		params := map[string]string{}

		for idx, name := range changeDeviceNameVolumeRegex.SubexpNames() {
			if name != "" {
				params[name] = matched[idx]
			}
		}

		if params["volume"] == "up" {
			return d.changeDeviceNameVolume(params["deviceName"], 5), nil
		} else if params["volume"] == "down" {
			return d.changeDeviceNameVolume(params["deviceName"], -5), nil
		} else {
			return nil, ErrStatementNotHandled.Err()
		}
	} else if matched := setDeviceNameVolumeRegex.FindStringSubmatch(content); matched != nil {
		params := map[string]string{}

		for idx, name := range setDeviceNameVolumeRegex.SubexpNames() {
			if name != "" {
				params[name] = matched[idx]
			}
		}

		volume, err := strconv.ParseInt(params["volume"], 10, 64)
		if err != nil {
			d.logger.Warn("unable to set device volume",
				zap.Error(err),
			)

			return statementFromText("Invalid volume supplied"), nil
		} else if volume < 0 || volume > 100 {
			return statementFromText(fmt.Sprintf("Invalid volume supplied (must be >0 and <100, which %d isn't)", volume)), nil
		}

		return d.setDeviceNameVolume(params["deviceName"], int32(volume)), nil
	}

	return nil, ErrStatementNotHandled.Err()
}

func (d *Domotics) setDeviceIDIsOn(deviceID string, isOn bool) *Statement {
	device, err := d.getDevice(deviceID)
	if err != nil {
		d.logger.Info("error getting device",
			zap.String("device_id", deviceID),
			zap.Error(err),
		)

		return statementFromText("Can't set the device right now")
	} else if device == nil {
		return statementFromText("Can't find the device to set")
	}

	return d.setDeviceIsOn(device, isOn)
}

func (d *Domotics) setDeviceIsOn(device *domotics.Device, isOn bool) *Statement {
	if device.State.Binary == nil {
		return statementFromText("Device doesn't have an is-on option")
	}

	req := &domotics.SetDeviceStateRequest{
		Id:    device.Id,
		State: proto.Clone(device.State).(*domotics.DeviceState),
		User: &users.User{
			DisplayName: "test user",
		},
	}
	req.State.Binary.IsOn = isOn
	_, err := d.deviceClient.SetDeviceState(context.Background(), req)
	if err != nil {
		d.logger.Warn("unable to set device state",
			zap.Error(err),
		)

		return statementFromText("Can't set the device right now")
	}

	state := "on"
	if !isOn {
		state = "off"
	}
	return statementFromText(fmt.Sprintf("Turned %s %s", nameFromDevice(device), state))
}

func (d *Domotics) setDeviceVolume(device *domotics.Device, volume int32) *Statement {
	if device.State.Audio == nil {
		return statementFromText("Device doesn't have a volume option")
	}

	req := &domotics.SetDeviceStateRequest{
		Id:    device.Id,
		State: proto.Clone(device.State).(*domotics.DeviceState),
	}
	req.State.Audio.Volume = volume
	_, err := d.deviceClient.SetDeviceState(context.Background(), req)
	if err != nil {
		d.logger.Warn("unable to set device state",
			zap.Error(err),
		)

		return statementFromText("Can't set the device right now")
	}

	return statementFromText(fmt.Sprintf("Set volume to %d of %s", volume, nameFromDevice(device)))
}

func (d *Domotics) setDeviceNameIsOn(deviceName string, isOn bool) *Statement {
	device, err := d.getDeviceByName(deviceName)
	if err != nil {
		return statementFromText("Can't get devices right now")
	} else if device == nil {
		return statementFromText(fmt.Sprintf("Unable to find device named %s", deviceName))
	} else if device.State == nil || device.State.Binary == nil {
		return statementFromText("Device doesn't have an 'is on' option")
	}

	return d.setDeviceIsOn(device, isOn)
}

func (d *Domotics) setDeviceNameVolume(deviceName string, volume int32) *Statement {
	device, err := d.getDeviceByName(deviceName)
	if err != nil {
		return statementFromText("Can't get devices right now")
	} else if device == nil {
		return statementFromText(fmt.Sprintf("Unable to find device named %s", deviceName))
	} else if device.State == nil || device.State.Audio == nil {
		return statementFromText("Device doesn't have a volume option")
	}

	return d.setDeviceVolume(device, volume)
}

func (d *Domotics) changeDeviceNameVolume(deviceName string, volumeChange int32) *Statement {
	device, err := d.getDeviceByName(deviceName)
	if err != nil {
		return statementFromText("Can't get devices right now")
	} else if device == nil {
		return statementFromText(fmt.Sprintf("Unable to find device named %s", deviceName))
	} else if device.State == nil || device.State.Audio == nil {
		return statementFromText("Device doesn't have a volume option")
	}

	return d.setDeviceVolume(device, device.State.Audio.Volume+volumeChange)
}

func (d *Domotics) getDeviceByName(name string) (*domotics.Device, error) {
	resp, err := d.deviceClient.ListDevices(context.Background(), &domotics.ListDevicesRequest{})
	if err != nil {
		d.logger.Warn("unable to get devices",
			zap.Error(err),
		)

		return nil, err
	} else if resp == nil {
		d.logger.Warn("unable to get devices (empty response)")

		return nil, nil
	}

	for _, device := range resp.Devices {
		if strings.ToLower(device.Config.Name) == name {
			return device, nil
		}
	}

	return nil, nil
}

func (d *Domotics) getDevice(id string) (*domotics.Device, error) {
	resp, err := d.deviceClient.GetDevice(context.Background(), &domotics.GetDeviceRequest{
		Id: id,
	})
	if err != nil {
		d.logger.Warn("unable to get device",
			zap.Error(err),
		)

		return nil, err
	} else if resp == nil {
		d.logger.Warn("unable to get device (not found)")

		return nil, nil
	}

	return resp.Device, nil
}

func (d *Domotics) getBridges() *Statement {
	resp, err := d.bridgeClient.ListBridges(context.Background(), &domotics.ListBridgesRequest{})
	if err != nil {
		d.logger.Warn("unable to get bridges",
			zap.Error(err),
		)

		return statementFromText("Can't get the domotics bridges right now")
	} else if resp == nil {
		d.logger.Warn("unable to get bridges (empty response)")

		return statementFromText("Not sure what the bridge states are right now")
	}

	return statementFromBridges(resp.Bridges)
}

func (d *Domotics) getDevices() *Statement {
	resp, err := d.deviceClient.ListDevices(context.Background(), &domotics.ListDevicesRequest{})
	if err != nil {
		d.logger.Warn("unable to get devices",
			zap.Error(err),
		)

		return statementFromText("Can't get devices right now")
	} else if resp == nil {
		d.logger.Warn("unable to get devices (empty response)")

		return statementFromText("Not sure what the device states are right now")
	}

	return statementFromDevices(resp.Devices)
}

func statementFromBridges(bridges []*domotics.Bridge) *Statement {
	text := "Bridges:\n"
	for idx, bridge := range bridges {
		if idx > 0 {
			text += "\n"
		}
		text += fmt.Sprintf("*%s*\n%s - @%s\n", bridge.Config.Name, bridge.Id, bridge.Config.Address)
	}

	return statementFromText(text)
}

func statementFromDevices(devices []*domotics.Device) *Statement {
	text := "Devices:\n"
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Address < devices[j].Address
	})

	for idx, device := range devices {
		if idx > 0 {
			text += "\n"
		}

		desc := device.Config.Description
		if len(desc) < 1 {
			desc = device.Manufacturer + " " + device.ModelId
		}

		text += fmt.Sprintf("*%s*\n%s - %s - @%s\n", device.Config.Name, device.Id, desc, device.Address)
		if device.State.Binary != nil {
			text += fmt.Sprintf("IsOn: %t; ", device.State.Binary.IsOn)
		}
		if device.State.ColorRgb != nil {
			text += fmt.Sprintf("RGB: %d %d %d; ", device.State.ColorRgb.Red, device.State.ColorRgb.Green, device.State.ColorRgb.Blue)
		}
		if device.State.Range != nil {
			text += fmt.Sprintf("Range: %d; ", device.State.Range.Value)
		}
		if device.State.Audio != nil {
			text += fmt.Sprintf("Volume: %d, Treble: %d, Bass: %d, Muted? %t; ", device.State.Audio.Volume, device.State.Audio.Treble, device.State.Audio.Bass, device.State.Audio.IsMuted)
		}
		if device.State.StereoAudio != nil {
			text += fmt.Sprintf("Balance: %d; ", device.State.StereoAudio.Balance)
		}

		// Slice off the last space and semicolon
		if strings.HasSuffix(text, "; ") {
			text = text[:len(text)-2]
		}
	}

	return statementFromText(text)
}

func statementFromDeviceUpdate(update *domotics.DeviceUpdate) *Statement {
	text := nameFromDevice(update.Device)

	switch update.Action {
	case domotics.DeviceUpdate_ADDED:
		text += " was added"
	case domotics.DeviceUpdate_CHANGED:
		text += " was changed"

		// TODO: find what changed
		if update.Device.State != nil && update.Device.State.Binary != nil {
			text += " and is now "
			if update.Device.State.Binary.IsOn {
				text += "on"
			} else {
				text += "off"
			}
		}
	case domotics.DeviceUpdate_REMOVED:
		text += " was removed"
	}

	if update.OriginatingUser != nil {
		text += " by " + update.OriginatingUser.DisplayName
	}

	return statementFromText(text)
}

func nameFromDevice(device *domotics.Device) string {
	if device.Config != nil && len(device.Config.Name) > 0 {
		return device.Config.Name
	}

	return "Device " + device.Id
}
