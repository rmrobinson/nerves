package news

import (
	"context"
	"sort"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.uber.org/zap"
)

// API exposes an implementation of the NewsServiceServer interface.
type API struct {
	logger *zap.Logger

	articles map[string]*Article
}

// NewAPI creates a new instance of the API struct.
func NewAPI(logger *zap.Logger) *API {
	return &API{
		logger:   logger,
		articles: map[string]*Article{},
	}
}

func (a *API) refreshArticles(articles []*Article) {
	for _, article := range articles {
		if existingArticle, ok := a.articles[article.Name]; ok {
			if !proto.Equal(existingArticle, article) && article.UpdatedAfter(existingArticle) {
				a.articles[article.Name] = article
			}

			continue
		}

		a.articles[article.Name] = article
	}
}

// StreamNewsUpdates streams live article updates to subscribers.
func (a *API) StreamNewsUpdates(*StreamNewsUpdatesRequest, NewsService_StreamNewsUpdatesServer) error {
	return nil
}

// ListArticles returns a collection of articles to the requester.
func (a *API) ListArticles(ctx context.Context, req *ListArticlesRequest) (*ListArticlesResponse, error) {
	ret := &ListArticlesResponse{}

	for _, article := range a.articles {
		ret.Articles = append(ret.Articles, article)
	}

	sort.Slice(ret.Articles, func(i, j int) bool {
		return ret.Articles[i].CreateTime.Seconds > ret.Articles[j].CreateTime.Seconds
	})

	return ret, nil
}

// UpdatedAfter is used to determine if this article was updated after the supplied article.
func (a *Article) UpdatedAfter(o *Article) bool {
	var aUpdated time.Time
	if a.UpdateTime != nil {
		aUpdated, _ = ptypes.Timestamp(a.UpdateTime)
	} else {
		aUpdated, _ = ptypes.Timestamp(a.CreateTime)
	}

	var oUpdated time.Time
	if o.UpdateTime != nil {
		oUpdated, _ = ptypes.Timestamp(o.UpdateTime)
	} else {
		oUpdated, _ = ptypes.Timestamp(o.CreateTime)
	}

	return aUpdated.After(oUpdated)
}
