package transit

import (
	"time"

	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

// Stop represents a single stop that one or more route trips may visit.
type stopInfo struct {
	*gtfs.Stop

	f        *Feed
	arrivals []*arrivalInfo
}

// Arrivals is the set of trips that will visit this location, sorted by arrival time.
func (s *stopInfo) Arrivals() []*arrivalInfo {
	return s.arrivals
}

// RemainingArrivalsToday returns the ordered list of arrivals that have not yet arrived today.
func (s *stopInfo) RemainingArrivalsToday() []*arrivalInfo {
	return s.arrivalsForDay(time.Now())
}

// ArrivalsToday returns the ordered list of arrivals that will visit the stop today.
func (s *stopInfo) ArrivalsToday() []*arrivalInfo {
	return s.arrivalsForDay(time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location()))
}

func (s *stopInfo) arrivalsForDay(date time.Time) []*arrivalInfo {
	var ret []*arrivalInfo

	for _, arrival := range s.arrivals {
		shouldAdd := false

		if calendar, ok := s.f.calendars[arrival.trip.ServiceID]; ok {
			if calendar.StartDate.Before(date) && calendar.EndDate.After(date) {
				switch date.Weekday() {
				case time.Monday:
					shouldAdd = calendar.Monday == true
				case time.Tuesday:
					shouldAdd = calendar.Tuesday == true
				case time.Wednesday:
					shouldAdd = calendar.Wednesday == true
				case time.Thursday:
					shouldAdd = calendar.Thursday == true
				case time.Friday:
					shouldAdd = calendar.Friday == true
				case time.Saturday:
					shouldAdd = calendar.Saturday == true
				case time.Sunday:
					shouldAdd = calendar.Sunday == true
				}
			}
		}

		if calendarDateMap, ok := s.f.calendarDates[arrival.trip.ServiceID]; ok {
			if calendarDate, ok := calendarDateMap[date.Format(gtfs.DateFormat)]; ok {
				if calendarDate.ExceptionType == "1" {
					shouldAdd = true
				} else if calendarDate.ExceptionType == "2" {
					shouldAdd = false
				}
			}
		}

		if arrival.ArrivalTime.BeforeTime(date) {
			shouldAdd = false
		}

		if shouldAdd {
			ret = append(ret, arrival)
		}
	}

	return ret
}
