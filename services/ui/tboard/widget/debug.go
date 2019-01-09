package widget

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// Debug is a widget to display debug information..
type Debug struct {
	*tview.TextView

	app *tview.Application
}

// NewDebug creates a new debug widget.
func NewDebug(app *tview.Application) *Debug {
	d := &Debug{
		TextView: tview.NewTextView(),
		app:      app,
	}

	d.SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorBlue).
		SetBorder(true).
		SetTitle("Debug")

	return d
}

// Refresh updates the contents of the debug widget.
func (d *Debug) Refresh(contents string) {
	d.app.QueueUpdateDraw(func() {
		d.Clear()
		d.SetText(contents)
	})
}
