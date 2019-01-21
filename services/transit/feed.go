package transit

import (
	"context"
	"io/ioutil"
	"net/http"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/rmrobinson/nerves/services/transit/gtfs"
	"go.uber.org/zap"
)

// Feed represents a single feed, potentially encapsulating both static and real-time elements.
type Feed struct {
	logger       *zap.Logger
	realtimePath string

	dataset *gtfs.Dataset

	agencies  map[string]*gtfs.Agency
	calendars map[string]*gtfs.Calendar
	// Double map; first keyed by service ID, second by override date
	calendarDates map[string]map[string]*gtfs.CalendarDate

	stops        map[string]*Stop
	routes       map[string]*Route
	sortedRoutes []*Route
}

// NewFeed creates a new feed from the supplied dataset and the realtime path.
func NewFeed(logger *zap.Logger, dataset *gtfs.Dataset, realtimePath string) *Feed {
	f := &Feed{
		logger:       logger,
		dataset:      dataset,
		realtimePath: realtimePath,

		agencies:      map[string]*gtfs.Agency{},
		calendars:     map[string]*gtfs.Calendar{},
		calendarDates: map[string]map[string]*gtfs.CalendarDate{},
		stops:         map[string]*Stop{},
		routes:        map[string]*Route{},
	}

	f.setup()
	return f
}

// setup populates the internal data structures used to support queries against this feed.
func (f *Feed) setup() {
	for _, a := range f.dataset.Agencies {
		f.agencies[a.ID] = a
	}
	for _, c := range f.dataset.Calendar {
		f.calendars[c.ServiceID] = c
	}
	for _, cd := range f.dataset.CalendarDate {
		if cdmap, ok := f.calendarDates[cd.ServiceID]; ok {
			cdmap[cd.Date.Format(gtfs.DateFormat)] = cd
		} else {
			cdmap := map[string]*gtfs.CalendarDate{}
			cdmap[cd.Date.Format(gtfs.DateFormat)] = cd
			f.calendarDates[cd.ServiceID] = cdmap
		}
	}

	for _, s := range f.dataset.Stops {
		f.stops[s.ID] = &Stop{
			Stop: s,
			f:    f,
		}
	}
	for _, r := range f.dataset.Routes {
		f.routes[r.ID] = &Route{
			Route: r,
		}
	}

	trips := map[string]*Trip{}
	for _, gtfsTrip := range f.dataset.Trips {
		if _, ok := f.routes[gtfsTrip.RouteID]; !ok {
			f.logger.Info("trip specified missing route ID",
				zap.String("trip_id", gtfsTrip.ID),
				zap.String("route_id", gtfsTrip.RouteID),
			)
			continue
		}

		route := f.routes[gtfsTrip.RouteID]

		trip := &Trip{
			Trip:  gtfsTrip,
			route: route,
		}
		trips[gtfsTrip.ID] = trip

		route.trips = append(route.trips, trip)
	}

	for _, gtfsStopTime := range f.dataset.StopTimes {
		if _, ok := trips[gtfsStopTime.TripID]; !ok {
			f.logger.Info("stop time specified missing trip ID",
				zap.String("trip_id", gtfsStopTime.TripID),
				zap.String("stop_id", gtfsStopTime.StopID),
				zap.String("arrival_time", gtfsStopTime.ArrivalTime.String()),
			)
			continue
		} else if _, ok := f.stops[gtfsStopTime.StopID]; !ok {
			f.logger.Info("stop time specified missing stop ID",
				zap.String("trip_id", gtfsStopTime.TripID),
				zap.String("stop_id", gtfsStopTime.StopID),
				zap.String("arrival_time", gtfsStopTime.ArrivalTime.String()),
			)
			continue
		}

		trip := trips[gtfsStopTime.TripID]
		stop := f.stops[gtfsStopTime.StopID]

		stopTime := &Arrival{
			StopTime: gtfsStopTime,
			stop:     f.stops[gtfsStopTime.StopID],
			trip:     trips[gtfsStopTime.TripID],
		}

		trip.stops = append(trip.stops, stopTime)
		stop.arrivals = append(stop.arrivals, stopTime)
	}

	// Now that the data structures are populated, perform sorting
	// Trips are prioritized by sequence
	// Stops have their arrivals ordered by arrival time
	// Routes have their trips ordered by the arrival time of the vehicle at the first stop on the trip

	for _, trip := range trips {
		sort.Slice(trip.stops, func(i, j int) bool {
			return trip.stops[i].Sequence < trip.stops[j].Sequence
		})
	}

	for stopID, stop := range f.stops {
		sort.Slice(stop.arrivals, func(i, j int) bool {
			h1 := stop.arrivals[i].ArrivalTime.Hour
			m1 := stop.arrivals[i].ArrivalTime.Minute
			s1 := stop.arrivals[i].ArrivalTime.Second

			h2 := stop.arrivals[j].ArrivalTime.Hour
			m2 := stop.arrivals[j].ArrivalTime.Minute
			s2 := stop.arrivals[j].ArrivalTime.Second

			if h1 < h2 {
				return true
			} else if h1 == h2 && m1 < m2 {
				return true
			} else if h1 == h2 && m1 == m2 && s1 < s2 {
				return true
			}
			return false
		})

		f.stops[stopID] = stop
	}

	for routeID, route := range f.routes {
		sort.Slice(route.trips, func(i, j int) bool {
			h1 := route.trips[i].stops[0].ArrivalTime.Hour
			m1 := route.trips[i].stops[0].ArrivalTime.Minute
			s1 := route.trips[i].stops[0].ArrivalTime.Second

			h2 := route.trips[j].stops[0].ArrivalTime.Hour
			m2 := route.trips[j].stops[0].ArrivalTime.Minute
			s2 := route.trips[j].stops[0].ArrivalTime.Second

			if h1 < h2 {
				return true
			} else if h1 == h2 && m1 < m2 {
				return true
			} else if h1 == h2 && m1 == m2 && s1 < s2 {
				return true
			}
			return false
		})

		f.routes[routeID] = route
	}

	for _, route := range f.routes {
		f.sortedRoutes = append(f.sortedRoutes, route)
	}
	sort.Slice(f.sortedRoutes, func(i, j int) bool {
		return f.sortedRoutes[i].SortOrder > f.sortedRoutes[j].SortOrder
	})

}

// GetRealtimeFeed retrieves the GTFS realtime data.
func (f *Feed) GetRealtimeFeed(ctx context.Context) (*FeedMessage, error) {
	body, err := f.getPath(ctx, f.realtimePath)
	if err != nil {
		f.logger.Warn("error retrieving body",
			zap.Error(err),
		)
		return nil, err
	}

	feed := &FeedMessage{}
	err = proto.Unmarshal(body, feed)
	if err != nil {
		f.logger.Warn("error unmarshaling body",
			zap.Error(err),
		)
		return nil, err
	}

	return feed, nil
}

func (f *Feed) getPath(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		f.logger.Warn("error creating request",
			zap.Error(err),
		)
		return nil, err
	}

	req.WithContext(ctx)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		f.logger.Warn("error performing request",
			zap.Error(err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		f.logger.Info("received non-OK response",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, nil
	}

	return ioutil.ReadAll(resp.Body)
}

// Routes returns the set of routes loaded into this feed.
func (f *Feed) Routes() []*Route {
	return f.sortedRoutes
}

// Stops returns the set of stops loaded into this feed.
func (f *Feed) Stops() map[string]*Stop {
	return f.stops
}
