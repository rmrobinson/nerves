package widget

import (
	"github.com/rivo/tview"
	"github.com/rmrobinson/nerves/services/transit"
)

type transitRecord struct {
	*tview.Flex

	routeText            *tview.TextView
	scheduledArrivalTime *tview.TextView
	estimatedArrivalTime *tview.TextView
}

func newTransitRecord() *transitRecord {
	tr := &transitRecord{
		Flex:                 tview.NewFlex(),
		routeText:            tview.NewTextView(),
		scheduledArrivalTime: tview.NewTextView(),
		estimatedArrivalTime: tview.NewTextView(),
	}

	tr.routeText.SetTextAlign(tview.AlignLeft)
	tr.scheduledArrivalTime.SetTextAlign(tview.AlignLeft)
	tr.estimatedArrivalTime.SetTextAlign(tview.AlignRight)

	tr.SetDirection(tview.FlexRow).
		AddItem(tr.routeText, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tr.scheduledArrivalTime, 0, 1, false).
			AddItem(tr.estimatedArrivalTime, 0, 1, false), 0, 1, false)

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
		wf.AddItem(wf.records[i], 2, 1, false)
	}

	return wf
}

// Refresh causes the transit data to be updated.
func (wf *Transit) Refresh(stop *transit.Stop, records []*transit.Arrival) {
	wf.app.QueueUpdateDraw(func() {
		if stop != nil {
			wf.SetTitle(stop.Name + " Arrivals")
		} else {
			wf.SetTitle("Arrivals")
		}
		
		for i := 0; i < len(wf.records); i++ {
			if i >= len(records) {
				wf.records[i].routeText.Clear()
				wf.records[i].scheduledArrivalTime.Clear()
				wf.records[i].estimatedArrivalTime.Clear()
				continue
			}

			record := records[i]

			wf.records[i].routeText.SetText(record.RouteId + " " + record.Headsign)
			wf.records[i].scheduledArrivalTime.SetText("S: " + record.ScheduledArrivalTime)
			if len(record.EstimatedArrivalTime) > 0 {
				wf.records[i].estimatedArrivalTime.SetText("E: " + record.EstimatedArrivalTime)
			} else {
				wf.records[i].estimatedArrivalTime.SetText("E: ??:??:??")
			}
		}
	})
}
