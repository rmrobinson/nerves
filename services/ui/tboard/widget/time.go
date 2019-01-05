package widget

import (
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// Time is a widget to display the time for the current location.
type Time struct {
	*tview.TextView

	app *tview.Application

	location *time.Location
}

// NewTime creates a new time widget using the supplied timezone.
func NewTime(app *tview.Application, location *time.Location) *Time {
	t := &Time{
		TextView: tview.NewTextView(),
		app:      app,
		location: location,
	}

	t.SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorLime).
		SetBorder(true).
		SetTitle(location.String())

	return t
}

// Run begins running the time widget.
func (t *Time) Run() {
	for {
		t.app.QueueUpdateDraw(func() {
			now := time.Now().In(t.location)
			part1 := now.Format("Mon, 02 Jan 2006")
			part2 := now.Format("15:04:05 MST")
			t.SetText(part1 + "\n" + part2)
		})
		time.Sleep(time.Millisecond * 100)
	}
}
