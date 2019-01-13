package geoset

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type geosettest struct {
	name         string
	entries      []entry
	closestValue int
	searchLat    float64
	searchLon    float64
}

var geosettests = []geosettest{
	{
		name: "closest to UW",
		entries: []entry{
			{
				// conestoga mall
				latitude:  43.4977,
				longitude: 80.5270,
				value:     4785,
			},
			{
				// fairview mall
				latitude:  43.4242,
				longitude: 80.4392,
				value:     5174,
			},
		},
		// university of waterloo
		searchLat:    43.4723,
		searchLon:    80.5449,
		closestValue: 4785, // conestoga mall is closer
	},
}

func TestGeoSet_Closest(t *testing.T) {
	for _, tt := range geosettests {
		t.Run(tt.name, func(t *testing.T) {
			geoset := NewGeoSet()
			for _, entry := range tt.entries {
				geoset.Add(entry.latitude, entry.longitude, entry.value)
			}
			val := geoset.Closest(tt.searchLat, tt.searchLon)
			intVal, ok := val.(int)
			assert.True(t, ok)
			assert.Equal(t, tt.closestValue, intVal)
		})
	}

}
