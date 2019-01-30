package widget

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/rivo/tview"
	"github.com/rmrobinson/nerves/services/transit"
)

type transitRecord struct {
	*tview.Flex

	routeText            *tview.TextView
	scheduledArrivalTime *tview.TextView
	arrivalTime          *tview.TextView
}

func newTransitRecord() *transitRecord {
	tr := &transitRecord{
		Flex:                 tview.NewFlex(),
		routeText:            tview.NewTextView(),
		scheduledArrivalTime: tview.NewTextView(),
		arrivalTime:          tview.NewTextView(),
	}

	tr.routeText.SetTextAlign(tview.AlignLeft)
	tr.scheduledArrivalTime.SetTextAlign(tview.AlignLeft)
	tr.arrivalTime.SetTextAlign(tview.AlignRight)

	tr.SetDirection(tview.FlexRow).
		AddItem(tr.routeText, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tr.scheduledArrivalTime, 0, 1, false).
			AddItem(tr.arrivalTime, 0, 2, false), 0, 1, false)

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
				wf.records[i].arrivalTime.Clear()
				continue
			}

			record := records[i]

			wf.records[i].routeText.SetText(record.RouteId + " " + record.Headsign)

			msg := "(Sched) "
			var err error
			var scheduledArrivalTime time.Time
			var arrivalTime time.Time

			scheduledArrivalTime, err = ptypes.Timestamp(record.ScheduledArrivalTime)

			if record.EstimatedArrivalTime != nil {
				msg = "(Est) "
				arrivalTime, err = ptypes.Timestamp(record.EstimatedArrivalTime)
			} else {
				arrivalTime = scheduledArrivalTime
			}

			if err != nil {
				wf.records[i].arrivalTime.SetText("Err: " + err.Error())
				continue
			}

			scheduledArrivalTime = scheduledArrivalTime.In(time.Now().Location())
			arrivalTime = arrivalTime.In(time.Now().Location())

			msg += getTextForTimeDiff(time.Now(), arrivalTime)

			wf.records[i].scheduledArrivalTime.SetText("@" + scheduledArrivalTime.Format("15:04:05"))
			wf.records[i].arrivalTime.SetText(msg)
		}
	})
}

func getTextForTimeDiff(t1 time.Time, t2 time.Time) string {
	if t1.After(t2) {
		duration := t1.Sub(t2)
		return fmt.Sprintf("%d mins ago", int64(duration.Minutes()))
	}

	duration := t2.Sub(t1)

	if duration < 1 {
		return "due"
	}

	return fmt.Sprintf("in %d mins", int64(duration.Minutes()))
}
