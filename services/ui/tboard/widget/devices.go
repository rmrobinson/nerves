package widget

import (
	"context"

	"github.com/rivo/tview"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
)

// Devices is a widget to display an editable list of devices with their details available.
type Devices struct {
	*tview.Flex

	app  *tview.Application
	next tview.Primitive

	logger *zap.Logger

	deviceList   *tview.List
	deviceDetail *DeviceDetail

	devicesClient bridge.BridgeServiceClient
	devices       []*bridge.Device
}

// NewDevices creates a new instance of this widget with the supplied set of devices for management.
func NewDevices(app *tview.Application, logger *zap.Logger, client bridge.BridgeServiceClient) *Devices {
	d := &Devices{
		Flex:          tview.NewFlex(),
		app:           app,
		logger:        logger,
		devices:       []*bridge.Device{},
		devicesClient: client,
	}

	d.deviceList = tview.NewList().
		SetChangedFunc(d.onListEntrySelected).
		SetSelectedFunc(d.onListEntryEntered).
		SetDoneFunc(d.onListDone)

	d.deviceDetail = NewDeviceDetail(app, d.logger, d.deviceList)

	d.SetBorder(true).
		SetTitle("Devices").
		SetTitleAlign(tview.AlignLeft)

	d.SetDirection(tview.FlexColumn).
		AddItem(d.deviceList, 0, 1, true).
		AddItem(d.deviceDetail, 28, 1, true)

	listDevicesResp, err := d.devicesClient.ListDevices(context.Background(), &bridge.ListDevicesRequest{})
	if err != nil {
		logger.Warn("unable to retrieve devices",
			zap.Error(err),
		)
		listDevicesResp = &bridge.ListDevicesResponse{}
	}

	d.devices = listDevicesResp.Devices

	for _, device := range d.devices {
		d.deviceList.AddItem(device.Config.Name, device.Config.Description, 0, nil)
	}

	return d
}

func (d *Devices) onListEntrySelected(idx int, mainText string, secondaryText string, shortcut rune) {
	d.deviceDetail.Refresh(d.devicesClient, d.devices[idx])
}

func (d *Devices) onListEntryEntered(idx int, mainText string, secondaryText string, shortcut rune) {
	d.app.SetFocus(d.deviceDetail)
}

func (d *Devices) onListDone() {
	if d.next != nil {
		d.app.SetFocus(d.next)
	}
}

// SetNextWidget controls where the focus is given should this list be left.
func (d *Devices) SetNextWidget(next tview.Primitive) {
	d.next = next
}
