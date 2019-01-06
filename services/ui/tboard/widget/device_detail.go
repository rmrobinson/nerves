package widget

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/rmrobinson/nerves/services/domotics"
)

// DeviceDetail is a widget that provides for viewing and editing DeviceInfo details.
type DeviceDetail struct {
	*tview.Flex

	app    *tview.Application
	parent tview.Primitive

	isOnCheckbox    *tview.Checkbox
	levelInput      *tview.InputField
	descriptionText *tview.TextView
	redInput        *tview.InputField
	greenInput      *tview.InputField
	blueInput       *tview.InputField
	saveButton      *tview.Button
	doneButton      *tview.Button

	device *domotics.Device
}

// NewDeviceDetail creates a new instance of the DeviceDetail view.
// Nothing will be displayed until a DeviceInfo is set on this view using Refresh()
func NewDeviceDetail(app *tview.Application, parent tview.Primitive) *DeviceDetail {
	// Create the view
	dd := &DeviceDetail{
		Flex:            tview.NewFlex(),
		app:             app,
		parent:          parent,
		isOnCheckbox:    tview.NewCheckbox(),
		levelInput:      tview.NewInputField(),
		descriptionText: tview.NewTextView(),
		redInput:        tview.NewInputField(),
		greenInput:      tview.NewInputField(),
		blueInput:       tview.NewInputField(),
		saveButton:      tview.NewButton("Save"),
		doneButton:      tview.NewButton("Done"),
	}

	// Setup the properties of the component views
	dd.isOnCheckbox.SetTitle("On?").
		SetBorder(true)

	dd.levelInput.SetFieldTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tcell.ColorGray).
		SetAcceptanceFunc(isInputFieldUint8).
		SetBorder(true).
		SetTitle("Level").
		SetTitleAlign(tview.AlignLeft)

	dd.redInput.SetFieldTextColor(tcell.ColorRed).
		SetFieldBackgroundColor(tcell.ColorGray).
		SetAcceptanceFunc(isInputFieldUint8).
		SetBorder(true).
		SetTitle("Red").
		SetTitleAlign(tview.AlignLeft)

	dd.greenInput.SetFieldTextColor(tcell.ColorGreen).
		SetFieldBackgroundColor(tcell.ColorGray).
		SetAcceptanceFunc(isInputFieldUint8).
		SetBorder(true).
		SetTitle("Green").
		SetTitleAlign(tview.AlignLeft)

	dd.blueInput.SetFieldTextColor(tcell.ColorBlue).
		SetFieldBackgroundColor(tcell.ColorGray).
		SetAcceptanceFunc(isInputFieldUint8).
		SetBorder(true).
		SetTitle("Blue").
		SetTitleAlign(tview.AlignLeft)

	dd.descriptionText.
		SetTextAlign(tview.AlignLeft).
		SetTitle("Desc").
		SetBorder(true)

	// Setup the view navigation flow
	dd.isOnCheckbox.SetDoneFunc(func(key tcell.Key) {
		dd.app.SetFocus(dd.levelInput)
	})
	dd.levelInput.SetDoneFunc(func(key tcell.Key) {
		dd.app.SetFocus(dd.redInput)
	})
	dd.redInput.SetDoneFunc(func(key tcell.Key) {
		dd.app.SetFocus(dd.greenInput)
	})
	dd.greenInput.SetDoneFunc(func(key tcell.Key) {
		dd.app.SetFocus(dd.blueInput)
	})
	dd.blueInput.SetDoneFunc(func(key tcell.Key) {
		dd.app.SetFocus(dd.saveButton)
	})
	dd.saveButton.SetSelectedFunc(func() {
		dd.saveFields()
	})
	dd.saveButton.SetBlurFunc(func(key tcell.Key) {
		dd.app.SetFocus(dd.doneButton)
	})
	dd.doneButton.SetSelectedFunc(func() {
		dd.app.SetFocus(dd.parent)
	})
	dd.doneButton.SetBlurFunc(func(key tcell.Key) {
		dd.app.SetFocus(dd.isOnCheckbox)
	})

	// Set the layout of the parent and return.
	dd.SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(dd.isOnCheckbox, 5, 1, true).
			AddItem(dd.levelInput, 7, 1, false), 3, 1, true).
		AddItem(tview.NewFlex().
			AddItem(dd.redInput, 7, 1, false).
			AddItem(dd.greenInput, 7, 1, false).
			AddItem(dd.blueInput, 7, 1, false), 3, 1, false).
		AddItem(dd.descriptionText, 10, 1, false).
		AddItem(tview.NewFlex().
			AddItem(dd.saveButton, 0, 1, false).
			AddItem(dd.doneButton, 0, 1, false), 1, 1, false)

	return dd
}

// Refresh takes the supplied DeviceInfo and refreshes the view with its contents.
func (dd *DeviceDetail) Refresh(device *domotics.Device) {
	dd.app.QueueUpdateDraw(func() {
		dd.device = device

		dd.SetTitle(dd.device.Config.Name)
		dd.isOnCheckbox.SetChecked(dd.device.State.Binary.IsOn)
		dd.descriptionText.SetText(dd.device.Config.Description)

		// TODO: do not allow/show values for fields that aren't supported.
		if dd.device.State.Range != nil {
			dd.levelInput.SetText(fmt.Sprintf("%2d", dd.device.State.Range.Value))
		}
		if dd.device.State.ColorRgb != nil {
			dd.redInput.SetText(fmt.Sprintf("%3d", dd.device.State.ColorRgb.Red))
			dd.greenInput.SetText(fmt.Sprintf("%3d", dd.device.State.ColorRgb.Green))
			dd.blueInput.SetText(fmt.Sprintf("%3d", dd.device.State.ColorRgb.Blue))
		}
	})
}

// saveFields is used to persist the contents of the view back into the linked DeviceInfo.
func (dd *DeviceDetail) saveFields() {
	// TODO: do not persist fields that aren't supported
	dd.device.State.Binary.IsOn = dd.isOnCheckbox.IsChecked()
	if dd.device.State.Range != nil {
		dd.device.State.Range.Value = int32FromInputField(dd.levelInput)
	}
	if dd.device.State.ColorRgb != nil {
		dd.device.State.ColorRgb.Red = int32FromInputField(dd.redInput)
		dd.device.State.ColorRgb.Green = int32FromInputField(dd.greenInput)
		dd.device.State.ColorRgb.Blue = int32FromInputField(dd.blueInput)
	}
}

func int32FromInputField(view *tview.InputField) int32 {
	if val, err := strconv.ParseUint(strings.TrimSpace(view.GetText()), 10, 8); err == nil {
		return int32(val)
	}
	return 0
}

func isInputFieldUint8(text string, ch rune) bool {
	// If we have a centered textbox then there will be 'padding' spaces. Ignore.
	text = strings.TrimSpace(text)
	if text == "-" {
		return true
	}
	val, err := strconv.Atoi(text)
	if err != nil {
		return false
	}
	if val < 0 || val > math.MaxUint8 {
		return false
	}
	return true
}
