package mind

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/rmrobinson/nerves/services/domotics"
	"go.uber.org/zap"
)

const (
	domoticsBridgeRegex = "what('| i)?s the bridge list"
	domoticsDeviceRegex = "what('| i)?s the house status"
)

// Domotics is a device-request handler
type Domotics struct {
	logger *zap.Logger

	bridgeClient domotics.BridgeServiceClient
	deviceClient domotics.DeviceServiceClient
}

// NewDomotics creates a new domotics handler
func NewDomotics(logger *zap.Logger, bridgeClient domotics.BridgeServiceClient, deviceClient domotics.DeviceServiceClient) *Domotics {
	return &Domotics{
		logger: logger,
		bridgeClient: bridgeClient,
		deviceClient: deviceClient,
	}
}

// ProcessStatement implements the handler interface. Logs and returns the statement.
func (d *Domotics) ProcessStatement(ctx context.Context, stmt *Statement) (*Statement, error) {
	if stmt.MimeType != "text/plain" {
		return nil, ErrStatementNotHandled.Err()
	}

	content := string(stmt.Content)
	content = strings.ToLower(content)
	if ok, _ := regexp.MatchString(domoticsBridgeRegex, content); ok {
		return d.getBridges(), nil
	} else if ok, _ := regexp.MatchString(domoticsDeviceRegex, content); ok {
		return d.getDevices(), nil
	}

	return nil, ErrStatementNotHandled.Err()
}

func (d *Domotics) getBridges() *Statement {
	resp, err := d.bridgeClient.ListBridges(context.Background(), &domotics.ListBridgesRequest{})
	if err != nil {
		d.logger.Warn("unable to get bridges",
			zap.Error(err),
		)

		return statementFromText("Can't get the domotics bridges right now :(")
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

		return statementFromText("Can't get the domotics devices right now :(")
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
	for idx, device := range devices {
		if idx > 0 {
			text += "\n"
		}
		text += fmt.Sprintf("*%s*\n%s - %s - @%s\n", device.Config.Name, device.Id, device.Config.Description, device.Address)
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
		if len(text) > 2 {
			text = text[:len(text)-2]
		}
	}

	return statementFromText(text)
}
