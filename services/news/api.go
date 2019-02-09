package news

import (
	"context"
	"sort"

	"go.uber.org/zap"
)

// API exposes an implementation of the NewsServiceServer interface.
type API struct {
	logger *zap.Logger

	cbcf *CBCFeed
	bbcf *BBCFeed
}

// NewAPI creates a new instance of the API struct.
func NewAPI(logger *zap.Logger, cbcf *CBCFeed, bbcf *BBCFeed) *API {
	return &API{
		logger: logger,
		cbcf:   cbcf,
		bbcf:   bbcf,
	}
}

// StreamNewsUpdates streams live article updates to subscribers.
func (a *API) StreamNewsUpdates(*StreamNewsUpdatesRequest, NewsService_StreamNewsUpdatesServer) error {
	return nil
}

// ListArticles returns a collection of articles to the requester.
func (a *API) ListArticles(ctx context.Context, req *ListArticlesRequest) (*ListArticlesResponse, error) {
	ret := &ListArticlesResponse{}
	ret.Articles = append(ret.Articles, a.cbcf.articles...)
	ret.Articles = append(ret.Articles, a.bbcf.articles...)

	sort.Slice(ret.Articles, func(i, j int) bool {
		return ret.Articles[i].CreateTime.Seconds > ret.Articles[j].CreateTime.Seconds
	})

	return ret, nil
}
