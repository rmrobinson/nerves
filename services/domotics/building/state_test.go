package building

import (
	"testing"

	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"github.com/stretchr/testify/assert"
)

func TestState_Dup(t *testing.T) {
	br := &bridge.Bridge{
		Id: "test bridge id",
	}
	r := &Room{
		Id:          "test room id",
		Name:        "test room",
		Description: "test room desc",
	}
	f := &Floor{
		Id:          "test floor id",
		Name:        "test floor",
		Description: "test floor desc",
		Level:       1,
		Rooms: []*Room{
			r,
		},
	}
	b := &Building{
		Id:          "test building id",
		Name:        "test building",
		Description: "test building desc",
		Floors: []*Floor{
			f,
		},
		Bridges: []*bridge.Bridge{
			br,
		},
	}
	orig := &State{
		buildings: map[string]*Building{},
		floors:    map[string]*Floor{},
		rooms:     map[string]*Room{},
		bridges:   map[string]*bridge.Bridge{},
	}

	orig.buildings[b.Id] = b
	orig.floors[f.Id] = f
	orig.rooms[r.Id] = r
	orig.bridges[br.Id] = br

	dup := orig.Dup()

	assert.Equal(t, *orig.buildings[b.Id].Floors[0], *dup.buildings[b.Id].Floors[0])
	assert.Equal(t, *orig.floors[f.Id].Rooms[0], *dup.floors[f.Id].Rooms[0])
	assert.Equal(t, *orig.rooms[r.Id], *dup.rooms[r.Id])
	assert.Equal(t, *orig.bridges[br.Id], *dup.bridges[br.Id])
}
