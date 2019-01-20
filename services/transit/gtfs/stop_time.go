package gtfs

// StopTime represents the time a specific stop is visited on a specific trip.
type StopTime struct {
	TripID                string   `csv:"trip_id"`
	ArrivalTime           string   `csv:"arrival_time"`
	DepartureTime         string   `csv:"departure_time"`
	StopID                string   `csv:"stop_id"`
	Sequence              CSVInt   `csv:"stop_sequence"`
	Headsign              string   `csv:"stop_headsign"`
	PickupType            string   `csv:"pickup_type"`
	DropOffType           string   `csv:"drop_off_type"`
	ShapeDistanceTraveled CSVFloat `csv:"shape_dist_traveled"`
	Timepoint             string   `csv:"timepoint"`
}
