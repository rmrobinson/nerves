package bridge

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	monopamp "github.com/rmrobinson/monoprice-amp-go"
	"github.com/rmrobinson/nerves/services/domotics"
)

var (
	// ErrZoneInvalid is returned if the supplied address maps to an invalid zone
	ErrZoneInvalid     = errors.New("supplied zone ID is not valid")
	// ErrChannelInvalid is returned if the supplied input maps to an invalid channel.
	ErrChannelInvalid  = errors.New("supplied channel ID is not valid")
	maxZoneID          = 6
	maxChannelID       = 6
	zoneAddrPrefix     = "/zone/"
	channelPrefix      = "Channel"
	baseMonopAmpBridge = &domotics.Bridge{
		ModelId:          "10761",
		ModelName:        "Monoprice Amp",
		ModelDescription: "6 Zone Home Audio Multizone Controller",
		Manufacturer:     "Monoprice",
	}
	baseMonopAmpDevice = &domotics.Device{
		ModelId:          "10761",
		ModelName:        "Zone",
		ModelDescription: "Monoprice Amp Zone",
		Manufacturer:     "Monoprice",
		Input: &domotics.Device_InputDevice{
			Inputs: []string{
				channelPrefix + "1",
				channelPrefix + "2",
				channelPrefix + "3",
				channelPrefix + "4",
				channelPrefix + "5",
				channelPrefix + "6",
			},
		},
	}
)

func addrToZone(addr string) int {
	zoneID, err := strconv.ParseInt(strings.TrimPrefix(addr, zoneAddrPrefix), 10, 32)
	if err != nil {
		return 0 // this is an invalid ID
	}
	return int(zoneID)
}
func zoneToAddr(id int) string {
	return fmt.Sprintf("%s%d", zoneAddrPrefix, id)
}
func inputToChannel(input string) int {
	channelID, err := strconv.ParseInt(strings.TrimPrefix(input, channelPrefix), 10, 32)
	if err != nil {
		return 0 // this is an invalid ID
	}
	return int(channelID)
}
func channelToInput(id int) string {
	return fmt.Sprintf("%s%d", channelPrefix, id)
}
func volumeFromProto(original int32) int {
	return int(original / 100) * 38
}
func volumeToProto(original int) int32 {
	return int32(original / 38) * 100
}
func balanceFromProto(original int32) int {
	return int(original / 100) * 20
}
func balanceToProto(original int) int32 {
	return int32(original / 20) * 100
}
func noteFromProto(original int32) int {
	return int(original / 100) * 14
}
func noteToProto(original int) int32 {
	return int32(original / 14) * 100
}

// MonopAmpBridge is an implementation of a bridge for the Monoprice amp/stereo output device.
type MonopAmpBridge struct {
	amp *monopamp.SerialAmplifier

	persister domotics.BridgePersister
}

// NewMonopAmpBridge takes a previously set up MonopAmp handle and exposes it as a MonopAmp bridge.
func NewMonopAmpBridge(amp *monopamp.SerialAmplifier, persister domotics.BridgePersister) *MonopAmpBridge {
	return &MonopAmpBridge{
		amp:       amp,
		persister: persister,
	}
}

// Setup seeds the persistent store with the proper data
func (b *MonopAmpBridge) Setup(ctx context.Context) error {
	// Populate the devices
	for zoneID := 1; zoneID <= maxZoneID; zoneID++ {
		d := &domotics.Device{
			// Id is populated by CreateDevice
			IsActive: true,
			Address:  zoneToAddr(zoneID),
			Config: &domotics.DeviceConfig{
				Name:        fmt.Sprintf("Amp Zone %d", zoneID),
				Description: "Amplifier output for the specified zone",
			},
		}
		proto.Merge(d, baseMonopAmpDevice)
		if err := b.persister.CreateDevice(ctx, d); err != nil {
			return err
		}
	}

	return nil
}

// Bridge retrieves the persisted state of the bridge from the backing store.
func (b *MonopAmpBridge) Bridge(ctx context.Context) (*domotics.Bridge, error) {
	bridge, err := b.persister.Bridge(ctx)
	if err != nil {
		return nil, err
	}

	ret := &domotics.Bridge{
		Config: &domotics.BridgeConfig{
			Address: &domotics.Address{
				Usb: &domotics.Address_Usb{
					Path: "",
				},
			},
			Timezone: "UTC",
		},
	}
	proto.Merge(ret, baseMonopAmpBridge)
	proto.Merge(ret, bridge)
	return ret, nil
}

// SetBridgeConfig persists the new bridge config in the backing store.
func (b *MonopAmpBridge) SetBridgeConfig(ctx context.Context, config *domotics.BridgeConfig) error {
	return b.persister.SetBridgeConfig(ctx, config)
}
// SetBridgeState persists the new bridge state in the backing store.
func (b *MonopAmpBridge) SetBridgeState(ctx context.Context, state *domotics.BridgeState) error {
	return b.persister.SetBridgeState(ctx, state)
}

// SearchForAvailableDevices is a noop that returns immediately (nothing to search for).
func (b *MonopAmpBridge) SearchForAvailableDevices(context.Context) error {
	return nil
}
// AvailableDevices returns an empty result as all devices are always available; never 'to be added'.
func (b *MonopAmpBridge) AvailableDevices(ctx context.Context) ([]*domotics.Device, error) {
	return nil, nil
}
// Devices retrieves the list of zones and the current state of each device from the serial port.
func (b *MonopAmpBridge) Devices(ctx context.Context) ([]*domotics.Device, error) {
	devices, err := b.persister.Devices(ctx)
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		zone := b.amp.Zone(addrToZone(device.Address))
		if zone == nil {
			return nil, ErrZoneInvalid
		}

		if err := zone.Refresh(); err != nil {
			return nil, err
		}

		device.State.Binary = &domotics.DeviceState_BinaryState{
			IsOn: zone.State().IsOn,
		}
		device.State.Input = &domotics.DeviceState_InputState{
			Input: channelToInput(zone.State().SourceChannelID),
		}
		device.State.Audio = &domotics.DeviceState_AudioState{
			Volume: volumeToProto(zone.State().Volume),
			Treble: noteToProto(zone.State().Treble),
			Bass: noteToProto(zone.State().Bass),
			IsMuted: zone.State().IsMuteOn,
		}
		device.State.StereoAudio = &domotics.DeviceState_StereoAudioState{
			Balance: balanceToProto(zone.State().Balance),
		}

		proto.Merge(device, baseMonopAmpDevice)
	}
	return devices, nil
}
// Device retrieves the specified device ID.
func (b *MonopAmpBridge) Device(ctx context.Context, id string) (*domotics.Device, error) {
	device, err := b.persister.Device(ctx, id)
	if err != nil {
		return nil, err
	}
	proto.Merge(device, baseMonopAmpDevice)
	return device, nil
}

// SetDeviceConfig updates the persister with the new config options for the zone.
func (b *MonopAmpBridge) SetDeviceConfig(ctx context.Context, dev *domotics.Device, config *domotics.DeviceConfig) error {
	return b.persister.SetDeviceConfig(ctx, dev, config)
}
// SetDeviceState uses the serial port to update the modified settings on the zone.
func (b *MonopAmpBridge) SetDeviceState(ctx context.Context, dev *domotics.Device, state *domotics.DeviceState) error {
	zoneID := addrToZone(dev.Address)
	if zoneID < 1 || zoneID > maxZoneID {
		return ErrZoneInvalid
	}

	zone := b.amp.Zone(zoneID)
	if zone == nil {
		return ErrZoneInvalid
	}

	// Ensure we are operating on the latest profile of the device before checking for actions to take.
	err := zone.Refresh()
	if err != nil {
		return err
	}

	zState := zone.State()

	if zState.IsOn != state.Binary.IsOn {
		if err = zone.SetPower(state.Binary.IsOn); err != nil {
			return err
		}
	}

	channelID := inputToChannel(state.Input.Input)
	if channelID < 1 || channelID > maxChannelID {
		return ErrChannelInvalid
	}

	if zState.SourceChannelID != channelID {
		if err = zone.SetSourceChannel(channelID); err != nil {
			return err
		}
	}

	treble := noteFromProto(state.Audio.Treble)
	if zState.Treble != treble {
		if err = zone.SetTreble(treble); err != nil {
			return err
		}
	}

	bass := noteFromProto(state.Audio.Bass)
	if zState.Bass != bass {
		if err = zone.SetBass(bass); err != nil {
			return err
		}
	}

	volume := volumeFromProto(state.Audio.Volume)
	if zState.Volume != volume {
		if err = zone.SetVolume(volume); err != nil {
			return err
		}
	}

	if zState.IsMuteOn != state.Audio.IsMuted {
		if err = zone.SetMute(state.Audio.IsMuted); err != nil {
			return err
		}
	}

	balance := balanceFromProto(state.StereoAudio.Balance)
	if zState.Balance != balance {
		if err = zone.SetBalance(balance); err != nil {
			return err
		}
	}

	return nil
}
// AddDevice is not supported on this bridge as there is a fixed number of zones, always ready for use.
func (b *MonopAmpBridge) AddDevice(ctx context.Context, id string) error {
	return domotics.ErrOperationNotSupported
}
// DeleteDevice is not supported on this bridge as there is a fixed number of zones, always ready for use.
func (b *MonopAmpBridge) DeleteDevice(ctx context.Context, id string) error {
	return domotics.ErrOperationNotSupported
}
