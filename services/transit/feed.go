package transit

import (
	"context"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/rmrobinson/nerves/services/transit/gtfs"
	"github.com/rmrobinson/nerves/services/transit/gtfs_realtime"
	"go.uber.org/zap"
)

const (
	realtimePollInterval = time.Second * 30
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

	stops        map[string]*stopDetails
	routes       map[string]*routeDetails
	trips        map[string]*tripDetails
	sortedRoutes []*routeDetails
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
		stops:         map[string]*stopDetails{},
		routes:        map[string]*routeDetails{},
		trips:         map[string]*tripDetails{},
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
		f.stops[s.ID] = &stopDetails{
			Stop: s,
			f:    f,
		}
	}
	for _, r := range f.dataset.Routes {
		f.routes[r.ID] = &routeDetails{
			Route: r,
		}
	}

	for _, gtfsTrip := range f.dataset.Trips {
		if _, ok := f.routes[gtfsTrip.RouteID]; !ok {
			f.logger.Info("trip specified missing route ID",
				zap.String("trip_id", gtfsTrip.ID),
				zap.String("route_id", gtfsTrip.RouteID),
			)
			continue
		}

		route := f.routes[gtfsTrip.RouteID]

		trip := &tripDetails{
			Trip:  gtfsTrip,
			route: route,
		}
		f.trips[gtfsTrip.ID] = trip

		route.trips = append(route.trips, trip)
	}

	for _, gtfsStopTime := range f.dataset.StopTimes {
		if _, ok := f.trips[gtfsStopTime.TripID]; !ok {
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

		trip := f.trips[gtfsStopTime.TripID]
		stop := f.stops[gtfsStopTime.StopID]

		stopTime := &arrivalDetails{
			StopTime: gtfsStopTime,
			stop:     f.stops[gtfsStopTime.StopID],
			trip:     f.trips[gtfsStopTime.TripID],
		}

		trip.stops = append(trip.stops, stopTime)
		stop.arrivals = append(stop.arrivals, stopTime)
	}

	// Now that the data structures are populated, perform sorting
	// Trips are prioritized by sequence
	// Stops have their arrivals ordered by arrival time
	// Routes have their trips ordered by the arrival time of the vehicle at the first stop on the trip

	for _, trip := range f.trips {
		sort.Slice(trip.stops, func(i, j int) bool {
			return trip.stops[i].Sequence < trip.stops[j].Sequence
		})
	}

	for stopID, stop := range f.stops {
		sort.Slice(stop.arrivals, func(i, j int) bool {
			return stop.arrivals[i].ArrivalTime.Before(stop.arrivals[j].ArrivalTime)
		})

		f.stops[stopID] = stop
	}

	for routeID, route := range f.routes {
		sort.Slice(route.trips, func(i, j int) bool {
			return route.trips[i].stops[0].ArrivalTime.Before(route.trips[j].stops[0].ArrivalTime)
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

// MonitorRealtimeFeed periodically polls the realtime feed endpoint and updates the times for the trips.
func (f *Feed) MonitorRealtimeFeed(ctx context.Context) {
	if len(f.realtimePath) < 1 {
		f.logger.Warn("realtime path not set, cannot monitor feed")
		return
	}

	for {
		feed, err := f.GetRealtimeFeed(ctx)
		if err != nil {
			f.logger.Warn("error retrieving feed",
				zap.Error(err),
			)
			return
		}
		if feed.Header.Incrementality != nil &&
			*feed.Header.Incrementality != gtfs_realtime.FeedHeader_FULL_DATASET {
			f.logger.Warn("realtime feed not supported")
			return
		}
		for _, entity := range feed.Entity {
			if entity.TripUpdate == nil {
				f.logger.Debug("non-trip update")
				continue
			} else if entity.TripUpdate.Trip == nil {
				f.logger.Debug("trip update missing trip details")
			}

			trip := f.trips[*entity.TripUpdate.Trip.TripId]
			if trip == nil {
				f.logger.Debug("trip not found",
					zap.String("trip_id", *entity.TripUpdate.Trip.TripId),
				)
				continue
			}

			updates := map[string]*gtfs_realtime.TripUpdate_StopTimeUpdate{}

			// Get a map of updates, indexed by stop ID
			for _, update := range entity.TripUpdate.StopTimeUpdate {
				updates[*update.StopId] = update
			}

			// Iterate over the stops for the trip. Set any estimated times, if they exist.
			for _, stop := range trip.stops {
				update := updates[stop.StopID]

				// If we have no update for this stop assume it occurred in the past and we can clear its estimated arrival time value.
				if update == nil {
					stop.estimatedArrivalTime = nil
					continue
				}

				// If we have no data, skip this stop
				if update.ScheduleRelationship != nil &&
					*update.ScheduleRelationship == gtfs_realtime.TripUpdate_StopTimeUpdate_NO_DATA {
					continue
				}

				if update.Arrival != nil {
					if update.Arrival.Time != nil {
						stop.estimatedArrivalTime = gtfs.NewCSVTime(time.Unix(*update.Arrival.Time, 0))
					} else if update.Arrival.Delay != nil {
						f.logger.Info("arrival delay present")
					}
				} else {
					stop.estimatedArrivalTime = nil
				}

				if update.Departure != nil {
					if update.Departure.Time != nil {
						stop.estimatedDepartureTime = gtfs.NewCSVTime(time.Unix(*update.Departure.Time, 0))
					} else if update.Departure.Delay != nil {
						f.logger.Info("departure delay present")
					}
				} else {
					stop.estimatedDepartureTime = nil
				}
			}
		}

		time.Sleep(realtimePollInterval)
	}
}

// GetRealtimeFeed retrieves the GTFS realtime data.
func (f *Feed) GetRealtimeFeed(ctx context.Context) (*gtfs_realtime.FeedMessage, error) {
	body, err := f.getPath(ctx, f.realtimePath)
	if err != nil {
		f.logger.Warn("error retrieving body",
			zap.Error(err),
		)
		return nil, err
	}

	feed := &gtfs_realtime.FeedMessage{}
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
