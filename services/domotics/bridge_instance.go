package domotics

import (
	"context"
	"reflect"
	"sync"
	"time"
)

type bridgeInstance struct {
	bridgeID string
	bridge   *Bridge
	devices  map[string]*Device

	bridgeHandle  SyncBridge
	cancelRefresh chan bool
	notifier      Notifier
	lock          sync.Mutex
}

func newBridgeInstance(bridgeHandle SyncBridge, bridge *Bridge, notifier Notifier) *bridgeInstance {
	ret := &bridgeInstance{
		bridgeHandle:  bridgeHandle,

		notifier: notifier,
		bridgeID: bridge.Id,
		bridge:   bridge,
		devices:  map[string]*Device{},
	}

	// TODO: figure this state machine out in more detail
	ret.bridge.Mode = BridgeMode_CREATED

	return ret
}

func (bi *bridgeInstance) refresh() {
	ctx := context.Background()

	bridge, err := bi.bridgeHandle.Bridge(ctx)
	if err != nil {
		bi.bridge.Mode = BridgeMode_INITIALIZED
		bi.bridge.ModeReason = err.Error()
		return
	}

	if !reflect.DeepEqual(bi.bridge, bridge) {
		bi.notifier.BridgeUpdated(bridge)
	}

	devices, err := bi.bridgeHandle.Devices(ctx)
	if err != nil {
		bi.bridge.Mode = BridgeMode_INITIALIZED
		bi.bridge.ModeReason = err.Error()
		return
	}

	newDevices := map[string]*Device{}
	for _, device := range devices {
		newDevices[device.Id] = device
	}

	// Determine what has changed between the 'current' and the 'new' versions of our device collection on the bridge.
	// Check if the new set has added anything.
	for id, newDevice := range newDevices {
		if _, ok := bi.devices[id]; !ok {
			bi.notifier.DeviceAdded(bi.bridgeID, newDevice)
		}
	}

	// Check if the current set has changed. This will have the 'newly added' devices above, but that's okay
	// since we already added them it'll end as a NOP.
	for id, currDevice := range bi.devices {
		if newDevice, ok := newDevices[id]; ok {
			if !reflect.DeepEqual(currDevice, newDevice) {
				bi.notifier.DeviceUpdated(bi.bridgeID, newDevice)
			}
		} else {
			bi.notifier.DeviceRemoved(bi.bridgeID, currDevice)
		}
	}

	// Check if the new set has added anything.

	if bi.bridge.Mode == BridgeMode_INITIALIZED {
		bi.bridge.Mode = BridgeMode_ACTIVE
		bi.bridge.ModeReason = ""
	}
}

func (bi *bridgeInstance) monitor(interval time.Duration) {
	bi.cancelRefresh = make(chan bool)

	t := time.NewTicker(interval)
	for {
		select {
		case <-t.C:
			bi.refresh()
		case <-bi.cancelRefresh:
			return
		}
	}
}
