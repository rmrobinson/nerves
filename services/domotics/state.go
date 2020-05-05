package domotics

import (
	"github.com/golang/protobuf/proto"
)

// State contains the currently active view of the state of the linked service.
// It (or parts of it) may be persisted to ensure recovery of the service across reboots.
// It may be replicated to other services, however it will always represent the currently active state of a single service.
type State struct {
	buildings map[string]*Building
	floors    map[string]*Floor
	rooms     map[string]*Room
	bridges   map[string]*Bridge
}

// Dup performs a deep copy on our state
func (s *State) Dup() *State {
	ns := &State{
		buildings: map[string]*Building{},
		floors:    map[string]*Floor{},
		rooms:     map[string]*Room{},
		bridges:   map[string]*Bridge{},
	}

	// Duplicate the rooms
	for _, r := range s.rooms {
		nr := proto.Clone(r).(*Room)
		ns.rooms[r.Id] = nr
	}
	// Duplicate the bridges
	for _, b := range s.bridges {
		nb := proto.Clone(b).(*Bridge)
		ns.bridges[b.Id] = nb
	}

	// Duplicate the floors.
	for _, f := range s.floors {
		nf := proto.Clone(f).(*Floor)

		// We ensure the Room array for the floor points to the new fields.
		nf.Rooms = []*Room{}
		for _, r := range f.Rooms {
			nf.Rooms = append(nf.Rooms, ns.rooms[r.Id])
		}

		ns.floors[f.Id] = nf
	}

	for _, b := range s.buildings {
		nb := proto.Clone(b).(*Building)

		// We ensure the Floor array for the building points to the new fields.
		nb.Floors = []*Floor{}
		for _, f := range b.Floors {
			nb.Floors = append(nb.Floors, ns.floors[f.Id])
		}
		nb.Bridges = []*Bridge{}
		for _, br := range b.Bridges {
			nb.Bridges = append(nb.Bridges, ns.bridges[br.Id])
		}

		ns.buildings[b.Id] = nb
	}

	return ns
}
