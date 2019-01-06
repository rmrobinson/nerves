package domotics

import (
	"sync"
)

// bridgeWatcher is a bridge watcher. A user should try to keep the updates channel empty but failure to read this
// will not block updates from being propagated to other bw.
type bridgeWatcher struct {
	updates  chan *BridgeUpdate
	peerAddr string
}

// bridgeWatchers is a synchronized map that gives us the ability to safely add and remove bw while sending updates.
type bridgeWatchers struct {
	sync.Mutex
	watchers map[*bridgeWatcher]bool
}

func (bw *bridgeWatchers) add(watcher *bridgeWatcher) {
	bw.Lock()
	defer bw.Unlock()

	bw.watchers[watcher] = true
}

func (bw *bridgeWatchers) remove(watcher *bridgeWatcher) {
	bw.Lock()
	defer bw.Unlock()

	delete(bw.watchers, watcher)
}

// deviceWatcher is a device watcher. A user should try to keep the updates channel empty but failure to read this
// will not block updates from being propagated to other bw.
type deviceWatcher struct {
	updates  chan *DeviceUpdate
	peerAddr string
}

// deviceWatchers is a synchronized map that gives us the ability to safely add and remove bw while sending updates.
type deviceWatchers struct {
	sync.Mutex
	watchers map[*deviceWatcher]bool
}

func (dw *deviceWatchers) add(watcher *deviceWatcher) {
	dw.Lock()
	defer dw.Unlock()

	dw.watchers[watcher] = true
}

func (dw *deviceWatchers) remove(watcher *deviceWatcher) {
	dw.Lock()
	defer dw.Unlock()

	delete(dw.watchers, watcher)
}
