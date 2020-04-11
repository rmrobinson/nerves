package aggregator

import (
	"context"
	"time"

	"github.com/rmrobinson/nerves/services/feed"
	"go.uber.org/zap"
)

// DB implements the Persist contract for saving Feed configs and Entry records.
type DB struct {
}

// MockDB is a testing instance of a persister
type MockDB struct {
	logger  *zap.Logger
	feeds   []*feed.FeedInfo
	entries []*feed.Entry
}

// NewInMemoryDB initializes a new mock provider
func NewMockDB(logger *zap.Logger) *MockDB {
	return &MockDB{
		logger: logger,
		feeds: []*feed.FeedInfo{
			{
				Name: "feeds/cbc",
				Url:  "https://rss.cbc.ca/lineup/topstories.xml",
				Config: &feed.Config{
					RequireAuthor: true,
				},
			},
		},
	}
}

// PutFeed saves a feed
func (m *MockDB) PutFeed(ctx context.Context, feed *feed.FeedInfo) error {
	m.logger.Debug("ignoring put feed")
	return nil
}

// GetFeed returns a feed
func (m *MockDB) GetFeed(ctx context.Context, id string) (*feed.FeedInfo, error) {
	for _, feed := range m.feeds {
		if feed.Name == id {
			return feed, nil
		}
	}

	return nil, nil
}

// GetFeeds returns the saved set of feeds
func (m *MockDB) GetFeeds(ctx context.Context) ([]*feed.FeedInfo, error) {
	return m.feeds, nil
}

// PutEntry saves an entry
func (m *MockDB) PutEntry(ctx context.Context, e *feed.Entry) error {
	m.logger.Debug("putting entry",
		zap.String("name", e.Name),
	)
	m.entries = append(m.entries, e)
	return nil
}

// GetEntry retrieves an entry
func (m *MockDB) GetEntry(ctx context.Context) (*feed.Entry, error) {
	return nil, nil
}

// GetEntries retrieves an entry
func (m *MockDB) GetEntries(ctx context.Context, start *time.Time, end *time.Time) ([]*feed.Entry, error) {
	return m.entries, nil
}
