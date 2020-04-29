package domotics

import (
	"context"

	"github.com/davecgh/go-spew/spew"
)

// InMemoryPersister satisfies the requirements of the 'StatePersister' interface in memory.
type InMemoryPersister struct {
	s *State
}

// NewInMemoryPersister creates a new instance of an in-memory persister
func NewInMemoryPersister() *InMemoryPersister {
	return &InMemoryPersister{
		s: &State{
			buildings: map[string]*Building{},
			floors:    map[string]*Floor{},
			rooms:     map[string]*Room{},
		},
	}
}

// Persist takes the supplied state and creates a copy if it in memory for retrieval.
func (imp *InMemoryPersister) Persist(ctx context.Context, s *State) error {
	imp.s = s.Dup()

	spew.Dump(imp.s)
	return nil
}

// Load retrieves whatever is currently stored and returns it.
func (imp *InMemoryPersister) Load(ctx context.Context) (*State, error) {
	return imp.s.Dup(), nil
}
