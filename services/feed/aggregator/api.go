package aggregator

import (
	"context"

	"github.com/rmrobinson/nerves/services/feed"
	"go.uber.org/zap"
)

// API implements the gRPC server contracts for both the Feed and FeedAdmin services.
type API struct {
	logger *zap.Logger

	p Persister
}

// NewAPI creates a new API
func NewAPI(logger *zap.Logger, p Persister) *API {
	return &API{
		logger: logger,
		p:      p,
	}
}

// ListEntries retrieves the set of configured entries and returns them.
func (a *API) ListEntries(ctx context.Context, req *feed.ListEntriesRequest) (*feed.ListEntriesResponse, error) {
	entries, err := a.p.GetEntries(ctx, nil, nil)
	if err != nil {
		return nil, err
	}

	return &feed.ListEntriesResponse{
		Entries: entries,
	}, nil
}

// StreamEntryChanges sends updated entry records as they are received.
func (a *API) StreamEntryChanges(req *feed.StreamEntryChangesRequest, stream feed.FeedService_StreamEntryChangesServer) error {
	return nil
}
