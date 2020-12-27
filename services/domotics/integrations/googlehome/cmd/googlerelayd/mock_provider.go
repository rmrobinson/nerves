package main

import (
	"context"

	action "github.com/rmrobinson/google-smart-home-action-go"
	"go.uber.org/zap"
)

// MockLightbulb represents a mock lightbulb device.
type MockLightbulb struct {
	ID         string
	Name       string
	IsOn       bool
	Brightness int

	Color struct {
		Hue        float64
		Saturation float64
		Value      float64
	}
}

// GetState returns the current state of the lightbulb formatted for Google Assistant
func (l *MockLightbulb) GetState() action.DeviceState {
	return action.NewDeviceState(true).RecordOnOff(l.IsOn).RecordBrightness(l.Brightness).RecordColorHSV(l.Color.Hue, l.Color.Saturation, l.Color.Value)
}

// MockReceiver represents a mock AV receiver device.
type MockReceiver struct {
	ID        string
	Name      string
	IsOn      bool
	Volume    int
	Muted     bool
	CurrInput string
}

// GetState returns the current state of the receiver formatted for Google Assistant
func (r *MockReceiver) GetState() action.DeviceState {
	return action.NewDeviceState(true).RecordOnOff(r.IsOn).RecordInput(r.CurrInput).RecordVolume(r.Volume, r.Muted)
}

// MockProvider implements a Google Assistent provider with 'mock' data
type MockProvider struct {
	logger  *zap.Logger
	service *action.Service
	agentID string

	lights   map[string]MockLightbulb
	receiver MockReceiver
}

// NewMockProvider creates a new mock provider with the specified details.
func NewMockProvider(logger *zap.Logger, service *action.Service, lights map[string]MockLightbulb, receiver MockReceiver, agentID string) *MockProvider {
	return &MockProvider{
		logger:   logger,
		service:  service,
		lights:   lights,
		receiver: receiver,
		agentID:  agentID,
	}
}

// Sync returns the set of known devices.
func (m *MockProvider) Sync(context.Context, string) (*action.SyncResponse, error) {
	m.logger.Debug("sync")

	resp := &action.SyncResponse{}
	for _, lb := range m.lights {
		ad := action.NewLight(lb.ID)
		ad.Name = action.DeviceName{
			DefaultNames: []string{
				"Test lamp",
			},
			Name: lb.Name,
		}
		ad.WillReportState = false
		ad.RoomHint = "Test Room"
		ad.DeviceInfo = action.DeviceInfo{
			Manufacturer: "Faltung Systems",
			Model:        "tl001",
			HwVersion:    "0.2",
			SwVersion:    "0.3",
		}
		ad.AddOnOffTrait(false, false).AddBrightnessTrait(false).AddColourTrait(action.HSV, false)

		resp.Devices = append(resp.Devices, ad)
	}

	inputs := []action.DeviceInput{
		{
			Key: "input_1",
			Names: []action.DeviceInputName{
				{
					Synonyms: []string{
						"Input 1",
						"Google Chromecast Audio",
					},
					LanguageCode: "en",
				},
			},
		},
		{
			Key: "input_2",
			Names: []action.DeviceInputName{
				{
					Synonyms: []string{
						"Input 2",
						"Raspberry Pi",
					},
					LanguageCode: "en",
				},
			},
		},
	}
	ar := action.NewSimpleAVReceiver(m.receiver.ID, inputs, 100, true, false)
	ar.Name = action.DeviceName{
		DefaultNames: []string{
			"Test receiver",
		},
		Name: m.receiver.Name,
	}
	ar.WillReportState = true
	ar.RoomHint = "Test Room"
	ar.DeviceInfo = action.DeviceInfo{
		Manufacturer: "Faltung Systems",
		Model:        "tavr001",
		HwVersion:    "0.2",
		SwVersion:    "0.3",
	}

	resp.Devices = append(resp.Devices, ar)

	return resp, nil
}

// Disconnect removes this agent ID provider.
func (m *MockProvider) Disconnect(context.Context, string) error {
	m.logger.Debug("disconnect")
	return nil
}

// Query retrieves the requested device data
func (m *MockProvider) Query(_ context.Context, req *action.QueryRequest) (*action.QueryResponse, error) {
	m.logger.Debug("query")

	resp := &action.QueryResponse{
		States: map[string]action.DeviceState{},
	}

	for _, deviceArg := range req.Devices {
		if light, found := m.lights[deviceArg.ID]; found {
			resp.States[deviceArg.ID] = light.GetState()
		} else if m.receiver.ID == deviceArg.ID {
			resp.States[deviceArg.ID] = m.receiver.GetState()
		}
	}

	return resp, nil
}

// Execute makes the specified devices change state.
func (m *MockProvider) Execute(_ context.Context, req *action.ExecuteRequest) (*action.ExecuteResponse, error) {
	m.logger.Debug("execute")

	resp := &action.ExecuteResponse{
		UpdatedState: action.NewDeviceState(true),
	}

	for _, commandArg := range req.Commands {
		for _, command := range commandArg.Commands {
			m.logger.Debug("received command",
				zap.String("command", command.Name),
			)

			for _, deviceArg := range commandArg.TargetDevices {
				if m.receiver.ID == deviceArg.ID {
					if command.OnOff != nil {
						m.receiver.IsOn = command.OnOff.On
						resp.UpdatedState.RecordOnOff(m.receiver.IsOn)
					} else if command.SetVolume != nil {
						m.receiver.Volume = command.SetVolume.Level
						resp.UpdatedState.RecordVolume(m.receiver.Volume, false)
					} else if command.AdjustVolume != nil {
						m.receiver.Volume += command.AdjustVolume.Amount
						resp.UpdatedState.RecordVolume(m.receiver.Volume, false)
					} else if command.SetInput != nil {
						m.receiver.CurrInput = command.SetInput.NewInput
						resp.UpdatedState.RecordInput(m.receiver.CurrInput)
					} else {
						m.logger.Info("unsupported command",
							zap.String("command", command.Name),
						)
						continue
					}

					resp.UpdatedDevices = append(resp.UpdatedDevices, deviceArg.ID)
					continue
				} else if device, found := m.lights[deviceArg.ID]; found {
					if command.OnOff != nil {
						device.IsOn = command.OnOff.On
						resp.UpdatedState.RecordOnOff(device.IsOn)
						m.lights[deviceArg.ID] = device
					} else if command.BrightnessAbsolute != nil {
						device.Brightness = command.BrightnessAbsolute.Brightness
						resp.UpdatedState.RecordBrightness(device.Brightness)
						m.lights[deviceArg.ID] = device
					} else if command.BrightnessRelative != nil {
						device.Brightness += command.BrightnessRelative.RelativeWeight
						resp.UpdatedState.RecordBrightness(device.Brightness)
						m.lights[deviceArg.ID] = device
					} else if command.ColorAbsolute != nil {
						device.Color.Hue = command.ColorAbsolute.HSV.Hue
						device.Color.Saturation = command.ColorAbsolute.HSV.Saturation
						device.Color.Value = command.ColorAbsolute.HSV.Value
						resp.UpdatedState.RecordColorHSV(device.Color.Hue, device.Color.Saturation, device.Color.Value)
						m.lights[deviceArg.ID] = device
					} else {
						m.logger.Info("unsupported command",
							zap.String("command", command.Name))
						continue
					}

					resp.UpdatedDevices = append(resp.UpdatedDevices, deviceArg.ID)
					continue
				}

				m.logger.Info("device not found",
					zap.String("device_id", deviceArg.ID),
					zap.String("command", command.Name),
				)
			}
		}
	}

	return resp, nil
}
