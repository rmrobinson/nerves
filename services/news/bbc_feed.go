package news

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
)

// BBCFeed is a news feed provided by the British Broadcasting Corporation
type BBCFeed struct {
	logger *zap.Logger
	path   string

	articles []*Article
}

// NewBBCFeed creates a new news feed from the BBC.
func NewBBCFeed(logger *zap.Logger, path string) *BBCFeed {
	return &BBCFeed{
		logger: logger,
		path:   path,
	}
}

// Run begins a loop that will poll BBC for changes.
func (bbc *BBCFeed) Run(ctx context.Context) {
	bbc.logger.Info("run started")
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ctx.Done():
			bbc.logger.Info("context closed, completing run")
			ticker.Stop()
			return
		case <-ticker.C:
			bbc.logger.Debug("refreshing data")
			feed, err := bbc.getFeed(ctx)
			if err != nil {
				continue
			}

			articles, err := bbc.parseFeed(feed)
			if err != nil {
				bbc.logger.Warn("error parsing feed",
					zap.Error(err),
				)
				continue
			}

			bbc.articles = articles

			var tmp []string
			for _, article := range bbc.articles {
				tmp = append(tmp, article.String())
			}

			bbc.logger.Debug("feed results",
				zap.Strings("articles", tmp),
			)
		}
	}
}

func (bbc *BBCFeed) getFeed(ctx context.Context) (*gofeed.Feed, error) {
	req, err := http.NewRequest(http.MethodGet, bbc.path, nil)
	if err != nil {
		bbc.logger.Warn("error creating new request",
			zap.Error(err),
		)
		return nil, err
	}

	req.WithContext(ctx)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		bbc.logger.Warn("error performing request",
			zap.Error(err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bbc.logger.Info("received non-OK response",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, nil
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		bbc.logger.Warn("error parsing feed",
			zap.Error(err),
		)
		return nil, err
	}

	return feed, nil
}

func (bbc *BBCFeed) parseFeed(feed *gofeed.Feed) ([]*Article, error) {
	var articles []*Article

	for _, item := range feed.Items {
		if item.Title == "BBC News Channel" {
			continue
		}

		// This creates the article, fills in description and an image (if present)
		article := &Article{
			Name:        fmt.Sprintf("articles/bbc/%s", item.GUID),
			Title:       item.Title,
			Description: item.Description,
			Link:        item.Link,
		}

		if item.PublishedParsed != nil {
			article.CreateTime, _ = ptypes.TimestampProto(*item.PublishedParsed)
		} else {
			article.CreateTime = ptypes.TimestampNow()
		}

		if item.UpdatedParsed != nil {
			article.UpdateTime, _ = ptypes.TimestampProto(*item.UpdatedParsed)
		}

		if item.Author != nil {
			article.Author = fmt.Sprintf("%s (%s)", item.Author.Name, item.Author.Email)
		}

		if media, ok := item.Extensions["media"]; ok {
			if thumbnails, ok := media["thumbnail"]; ok {
				for _, thumbnail := range thumbnails {
					image := &Image{}
					for attrName, attrVal := range thumbnail.Attrs {
						if attrName == "width" {
							tmp, err := strconv.ParseInt(attrVal, 10, 32)
							if err != nil {
								bbc.logger.Debug("unable to parse thumbnail width",
									zap.Error(err),
								)
								continue
							}
							image.Width = int32(tmp)
						} else if attrName == "height" {
							tmp, err := strconv.ParseInt(attrVal, 10, 32)
							if err != nil {
								bbc.logger.Debug("unable to parse thumbnail width",
									zap.Error(err),
								)
								continue
							}
							image.Height = int32(tmp)
						} else if attrName == "url" {
							image.Link = attrVal
						}
					}

					article.Image = image
				}
			}
		}

		articles = append(articles, article)
	}

	return articles, nil
}
