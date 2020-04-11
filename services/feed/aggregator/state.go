package aggregator

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rmrobinson/nerves/services/feed"
	"go.uber.org/zap"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrInvalidFeed is returned if the supplied feed is invalid
	ErrInvalidFeed = status.New(codes.InvalidArgument, "feed invalid")
	// ErrFeedCreationFailed is returned if the feed couldn't be created
	ErrFeedCreationFailed = status.New(codes.Internal, "creating feed failed")
)

// Persister saves a specified feed item for subsequent retrieval.
type Persister interface {
	PutFeed(context.Context, *feed.FeedInfo) error
	GetFeed(context.Context, string) (*feed.FeedInfo, error)
	GetFeeds(context.Context) ([]*feed.FeedInfo, error)

	PutEntry(context.Context, *feed.Entry) error
	GetEntry(context.Context) (*feed.Entry, error)
	GetEntries(context.Context, *time.Time, *time.Time) ([]*feed.Entry, error)
}

// State contains the loaded feeds and monitors them.
type State struct {
	logger    *zap.Logger
	p         Persister
	transport http.RoundTripper

	providers []*provider
}

// NewState creates a new feed aggregator State
func NewState(logger *zap.Logger, p Persister) *State {
	return &State{
		logger:    logger,
		p:         p,
		transport: http.DefaultTransport,
	}
}

// Initialize retrieves the saved feeds from the persister and sets the State
// up to be monitoring these feeds.
func (s *State) Initialize(ctx context.Context) error {
	if s.p == nil {
		panic("missing persister")
	}

	feeds, err := s.p.GetFeeds(ctx)
	if err != nil {
		return nil
	}

	for _, feed := range feeds {
		p := newProvider(s.logger, s.transport, feed)
		s.providers = append(s.providers, p)
	}

	return nil
}

// Run launches a process of periodically checking the registered feeds for updates.
// It will exit when the supplied channel is closed.
func (s *State) Run(close <-chan bool) {
	s.refreshFeeds()

	ticker := time.NewTicker(30 * time.Minute)

	for {
		select {
		case <-close:
			s.logger.Debug("exiting run loop")
			return
		case <-ticker.C:
			s.logger.Debug("ticker fired, checking feeds")
			s.refreshFeeds()
		}
	}
}

func (s *State) refreshFeeds() {
	for _, provider := range s.providers {
		updated, err := provider.refresh(context.Background())
		if err != nil {
			s.logger.Error("error refreshing provider",
				zap.String("name", provider.info.Name),
				zap.Error(err),
			)
			continue
		}

		for _, entry := range updated {
			s.p.PutEntry(context.Background(), entry)
		}

		s.p.PutFeed(context.Background(), provider.info)
	}
}

// ListFeeds returns the list of active feeds.
func (s *State) ListFeeds(ctx context.Context, req *feed.ListFeedsRequest) (*feed.ListFeedsResponse, error) {
	resp := &feed.ListFeedsResponse{}
	for _, provider := range s.providers {
		resp.Feeds = append(resp.Feeds, provider.info)
	}

	return resp, nil
}

// AddFeed results in a new feed being registered and monitored.
func (s *State) AddFeed(ctx context.Context, feedURL string) (*feed.FeedInfo, *status.Status) {
	info := &feed.FeedInfo{
		Name: fmt.Sprintf("feeds/%s", uuid.New().String()),
		Url:  feedURL,
	}

	p := newProvider(s.logger, s.transport, info)
	s.providers = append(s.providers, p)

	updated, err := p.refresh(ctx)
	if err != nil {
		s.logger.Error("error refreshing feed for first time",
			zap.String("url", feedURL),
			zap.Error(err),
		)
		return nil, ErrFeedCreationFailed
	}

	err = s.p.PutFeed(ctx, info)
	if err != nil {
		s.logger.Error("error saving feed for first time",
			zap.String("url", feedURL),
			zap.Error(err),
		)

		return nil, ErrFeedCreationFailed
	}

	for _, entry := range updated {
		s.p.PutEntry(ctx, entry)
	}

	return info, nil
}
