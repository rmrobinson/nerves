package widget

import (
	"fmt"

	"github.com/rivo/tview"
)

// TransitArrivalInfo is a temporary struct holding the arrival info for a transit vehicle
type TransitArrivalInfo struct {
	RouteName     string
	ArrivesInMins int
}

type transitRecord struct {
	*tview.Flex

	routeText       *tview.TextView
	arrivalTimeText *tview.TextView
}

func newTransitRecord() *transitRecord {
	tr := &transitRecord{
		Flex:            tview.NewFlex(),
		routeText:       tview.NewTextView(),
		arrivalTimeText: tview.NewTextView(),
	}

	tr.routeText.SetTextAlign(tview.AlignLeft)
	tr.arrivalTimeText.SetTextAlign(tview.AlignRight)

	tr.SetDirection(tview.FlexColumn).
		AddItem(tr.routeText, 0, 1, false).
		AddItem(tr.arrivalTimeText, 7, 1, false)

	return tr
}

// Transit is a widget that displays the upcoming transit arrival times.
type Transit struct {
	*tview.Flex

	app *tview.Application

	records []*transitRecord
}

// NewTransit creates a new transit widget with the specified number of rows.
// It will not show any data until Refresh() is called to display the data.
func NewTransit(app *tview.Application, rowCount int) *Transit {
	wf := &Transit{
		Flex: tview.NewFlex(),
		app:  app,
	}

	wf.SetBorder(true).
		SetTitle("Transit").
		SetTitleAlign(tview.AlignLeft)

	wf.SetDirection(tview.FlexRow)
	for i := 0; i < rowCount; i++ {
		wf.records = append(wf.records, newTransitRecord())
		wf.AddItem(wf.records[i], 1, 1, false)
	}

	return wf
}

// Refresh causes the transit data to be updated.
func (wf *Transit) Refresh(records []TransitArrivalInfo) {
	wf.app.QueueUpdateDraw(func() {
		for i := 0; i < len(wf.records); i++ {
			if i >= len(records) {
				wf.records[i].routeText.Clear()
				wf.records[i].arrivalTimeText.Clear()
				continue
			}

			record := records[i]

			wf.records[i].routeText.SetText(record.RouteName)
			if record.ArrivesInMins < 1 {
				wf.records[i].arrivalTimeText.SetText("Due")
			} else {
				wf.records[i].arrivalTimeText.SetText(fmt.Sprintf("%d mins", record.ArrivesInMins))
			}
		}
	})
}
