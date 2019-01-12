package geoset

import (
	"math"
)

type entry struct {
	latitude float64
	longitude float64

	value interface{}
}

// GeoSet is a collection that allows for values to be stored by their latitude and longitude;
// and allows for lookups to find the entry closest to the supplied latitude and longitude.
type GeoSet struct {
	entries []entry
}

// NewGeoSet returns a new GeoSet
func NewGeoSet() *GeoSet {
	return &GeoSet{}
}

// Add the supplied value to the location specified with the latitude and longitude (in degrees)
func (gs *GeoSet) Add(lat float64, lon float64, value interface{}) {
	gs.entries = append(gs.entries, entry{lat, lon, value})
}

// Closest returns the entry in the set that is nearest to the supplied latitude and longitude (in degrees)
func (gs *GeoSet) Closest(lat float64, lon float64) interface{} {
	shortestDistance := math.MaxFloat64
	var value interface{}

	for _, entry := range gs.entries {
		distance := distance(lat, lon, entry.latitude, entry.longitude)
		if distance < shortestDistance {
			shortestDistance = distance
			value = entry.value
		}
	}

	return value
}

// haversine function
func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

// See http://en.wikipedia.org/wiki/Haversine_formula
func distance(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	// Radius of Earth in metres (mean earth radius, via https://en.wikipedia.org/wiki/Great-circle_distance)
	r := float64(6371000)

	// Convert degrees to radians
	la1 := lat1 * math.Pi / 180
	lo1 := lon1 * math.Pi / 180
	la2 := lat2 * math.Pi / 180
	lo2 := lon2 * math.Pi / 180

	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)
	return 2 * r * math.Asin(math.Sqrt(h))
}
