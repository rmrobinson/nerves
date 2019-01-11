package news

import (
	"context"

	"go.uber.org/zap"
)

// API exposes an implementation of the NewsServiceServer interface.
type API struct {
	logger *zap.Logger

	cbcf *CBCFeed
}

// NewAPI creates a new instance of the API struct.
func NewAPI(logger *zap.Logger, cbcf *CBCFeed) *API {
	return &API{
		logger: logger,
		cbcf:   cbcf,
	}
}

// StreamNewsUpdates streams live article updates to subscribers.
func (a *API) StreamNewsUpdates(*StreamNewsUpdatesRequest, NewsService_StreamNewsUpdatesServer) error {
	return nil
}

// ListArticles returns a collection of articles to the requester.
func (a *API) ListArticles(ctx context.Context, req *ListArticlesRequest) (*ListArticlesResponse, error) {
	return &ListArticlesResponse{
		Articles: a.cbcf.articles,
	}, nil
}
