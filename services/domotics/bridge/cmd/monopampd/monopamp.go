package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	monopamp "github.com/rmrobinson/monoprice-amp-go"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
)

var (
	// ErrZoneInvalid is returned if the supplied address maps to an invalid zone
	ErrZoneInvalid = errors.New("supplied zone ID is not valid")
	// ErrChannelInvalid is returned if the supplied input maps to an invalid channel.
	ErrChannelInvalid  = errors.New("supplied channel ID is not valid")
	maxZoneID          = 6
	maxChannelID       = 6
	zoneAddrPrefix     = "/zone/"
	channelPrefix      = "Channel"
	baseMonopAmpBridge = &bridge.Bridge{
		ModelId:          "10761",
		ModelName:        "Monoprice Amp",
		ModelDescription: "6 Zone Home Audio Multizone Controller",
		Manufacturer:     "Monoprice",
	}
	baseMonopAmpDevice = &bridge.Device{
		ModelId:          "10761",
		ModelName:        "Zone",
		ModelDescription: "Monoprice Amp Zone",
		Manufacturer:     "Monoprice",
		Input: &bridge.Device_Input{
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
	commandSpaceInterval = time.Millisecond * 100
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
	return int((original * 38) / 100)
}
func volumeToProto(original int) int32 {
	return int32((original * 100) / 38)
}
func balanceFromProto(original int32) int {
	return int(original / 5)
}
func balanceToProto(original int) int32 {
	return int32((original * 5) / 20)
}
func noteFromProto(original int32) int {
	return int((original * 14) / 100)
}
func noteToProto(original int) int32 {
	return int32((original * 100) / 14)
}

// MonopAmp is an implementation of a bridge for the Monoprice amp/stereo output device.
type MonopAmp struct {
	amp *monopamp.SerialAmplifier

	id   string
	path string
}

// NewMonopAmp takes a previously set up MonopAmp handle and exposes it as a MonopAmp bridge.
func NewMonopAmp(amp *monopamp.SerialAmplifier, id string, path string) *MonopAmp {
	return &MonopAmp{
		amp:  amp,
		id:   id,
		path: path,
	}
}

func (b *MonopAmp) getDevices(ctx context.Context) (map[string]*bridge.Device, error) {
	devices := map[string]*bridge.Device{}

	// Populate the devices
	for zoneID := 1; zoneID <= maxZoneID; zoneID++ {
		d := &bridge.Device{
			Id:       fmt.Sprintf("%s-%d", b.id, zoneID),
			IsActive: true,
			Address:  zoneToAddr(zoneID),
			Config: &bridge.DeviceConfig{
				Name:        fmt.Sprintf("Amp Zone %d", zoneID),
				Description: "Amplifier output for the specified zone",
			},
		}
		proto.Merge(d, baseMonopAmpDevice)

		devices[d.Id] = d
	}

	for _, device := range devices {
		if err := b.deviceFromAmp(device); err != nil {
			return nil, err
		}
	}

	return devices, nil
}

func (b *MonopAmp) getBridge(ctx context.Context) (*bridge.Bridge, error) {
	ret := &bridge.Bridge{
		Config: &bridge.BridgeConfig{
			Address: &bridge.Address{
				Usb: &bridge.Address_Usb{
					Path: b.path,
				},
			},
			Timezone: "UTC",
		},
	}
	proto.Merge(ret, baseMonopAmpBridge)
	return ret, nil
}

func (b *MonopAmp) deviceFromAmp(device *bridge.Device) error {
	zoneID := addrToZone(device.Address)
	zone := b.amp.Zone(zoneID)
	if zone == nil {
		return ErrZoneInvalid
	}

	if err := zone.Refresh(); err != nil {
		// Attempt to reset the connection once before erroring out.
		err = b.amp.Reset()
		if err != nil {
			return err
		}
		zone = b.amp.Zone(zoneID)
		if zone == nil {
			return ErrZoneInvalid
		}

		err = zone.Refresh()
		if err != nil {
			return err
		}
	}

	device.State.Binary = &bridge.DeviceState_Binary{
		IsOn: zone.State().IsOn,
	}
	device.State.Input = &bridge.DeviceState_Input{
		Input: channelToInput(zone.State().SourceChannelID),
	}
	device.State.Audio = &bridge.DeviceState_Audio{
		Volume:  volumeToProto(zone.State().Volume),
		Treble:  noteToProto(zone.State().Treble),
		Bass:    noteToProto(zone.State().Bass),
		IsMuted: zone.State().IsMuteOn,
	}
	device.State.StereoAudio = &bridge.DeviceState_StereoAudio{
		Balance: balanceToProto(zone.State().Balance),
	}

	proto.Merge(device, baseMonopAmpDevice)
	return nil
}

// SetDeviceState uses the serial port to update the modified settings on the zone.
func (b *MonopAmp) SetDeviceState(ctx context.Context, dev *bridge.Device, state *bridge.DeviceState) error {
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
		// Attempt to reset the connection once before erroring out.
		err = b.amp.Reset()
		if err != nil {
			return err
		}
		zone = b.amp.Zone(zoneID)
		if zone == nil {
			return ErrZoneInvalid
		}

		err = zone.Refresh()
		if err != nil {
			return err
		}
	}

	zState := zone.State()

	if state.Binary != nil {
		if zState.IsOn != state.Binary.IsOn {
			time.Sleep(commandSpaceInterval)
			if err = zone.SetPower(state.Binary.IsOn); err != nil {
				return err
			}
		}
	}

	if state.Input != nil {
		channelID := inputToChannel(state.Input.Input)
		if channelID < 1 || channelID > maxChannelID {
			return ErrChannelInvalid
		}

		if zState.SourceChannelID != channelID {
			time.Sleep(commandSpaceInterval)
			if err = zone.SetSourceChannel(channelID); err != nil {
				return err
			}
		}
	}

	if state.Audio != nil {
		treble := noteFromProto(state.Audio.Treble)
		if zState.Treble != treble {
			time.Sleep(commandSpaceInterval)
			if err = zone.SetTreble(treble); err != nil {
				return err
			}
		}

		bass := noteFromProto(state.Audio.Bass)
		if zState.Bass != bass {
			time.Sleep(commandSpaceInterval)
			if err = zone.SetBass(bass); err != nil {
				return err
			}
		}

		volume := volumeFromProto(state.Audio.Volume)
		if zState.Volume != volume {
			time.Sleep(commandSpaceInterval)
			if err = zone.SetVolume(volume); err != nil {
				return err
			}
		}

		if zState.IsMuteOn != state.Audio.IsMuted {
			time.Sleep(commandSpaceInterval)
			if err = zone.SetMute(state.Audio.IsMuted); err != nil {
				return err
			}
		}
	}

	if state.StereoAudio != nil {
		balance := balanceFromProto(state.StereoAudio.Balance)
		if zState.Balance != balance {
			time.Sleep(commandSpaceInterval)
			if err = zone.SetBalance(balance); err != nil {
				return err
			}
		}
	}

	return nil
}
