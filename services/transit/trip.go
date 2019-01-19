package transit

// Trip contains information about a trip.
type Trip struct {
	ID                   string `csv:"trip_id"`
	RouteID              string `csv:"route_id"`
	ServiceID            string `csv:"service_id"`
	Headsign             string `csv:"trip_headsign"`
	ShortName            string `csv:"trip_short_name"`
	DirectionID          string `csv:"direction_id"`
	BlockID              string `csv:"block_id"`
	ShapeID              string `csv:"shape_id"`
	WheelchairAccessible string `csv:"wheelchair_accessible"`
	BikesAllowed         string `csv:"bikes_allowed"`
}
