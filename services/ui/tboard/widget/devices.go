package widget

import (
	"github.com/rivo/tview"
)

// Devices is a widget to display an editable list of devices with their details available.
type Devices struct {
	*tview.Flex

	app *tview.Application

	deviceList   *tview.List
	deviceDetail *DeviceDetail

	devices []*DeviceInfo
}

// NewDevices creates a new instance of this widget with the supplied set of devices for management.
func NewDevices(app *tview.Application, devices []*DeviceInfo) *Devices {
	d := &Devices{
		Flex:    tview.NewFlex(),
		app:     app,
		devices: devices,
	}

	d.deviceList = tview.NewList().
		SetChangedFunc(d.onListEntrySelected).
		SetSelectedFunc(d.onListEntryEntered)

	d.deviceDetail = NewDeviceDetail(app, d.deviceList)

	d.SetBorder(true).
		SetTitle("Devices").
		SetTitleAlign(tview.AlignLeft)

	d.SetDirection(tview.FlexColumn).
		AddItem(d.deviceList, 0, 1, true).
		AddItem(d.deviceDetail, 28, 1, true)

	for _, device := range d.devices {
		d.deviceList.AddItem(device.Name, device.Description, 0, nil)
	}

	return d
}

func (d *Devices) onListEntrySelected(idx int, mainText string, secondaryText string, shortcut rune) {
	d.deviceDetail.Refresh(d.devices[idx])
}

func (d *Devices) onListEntryEntered(idx int, mainText string, secondaryText string, shortcut rune) {
	d.app.SetFocus(d.deviceDetail)
}
