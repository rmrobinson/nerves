package news

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
	"golang.org/x/net/html"
)

const (
	cbcRefreshRate = time.Minute * 10
)

// CBCFeed is a news feed provided by CBC News
type CBCFeed struct {
	logger *zap.Logger
	path   string

	api *API
}

// NewCBCFeed creates a new news feed from the CBC.
func NewCBCFeed(logger *zap.Logger, path string, api *API) *CBCFeed {
	return &CBCFeed{
		logger: logger,
		path:   path,
		api:    api,
	}
}

// Run begins a loop that will poll CBC for changes.
func (cbc *CBCFeed) Run(ctx context.Context) {
	cbc.logger.Info("run started")
	cbc.refresh(ctx)

	ticker := time.NewTicker(cbcRefreshRate)
	for {
		select {
		case <-ctx.Done():
			cbc.logger.Info("context closed, completing run")
			ticker.Stop()
			return
		case <-ticker.C:
			cbc.refresh(ctx)
		}
	}
}

func (cbc *CBCFeed) refresh(ctx context.Context) {
	cbc.logger.Debug("refreshing data")
	feed, err := cbc.getFeed(ctx)
	if err != nil {
		cbc.logger.Warn("error getting feed",
			zap.Error(err),
		)
		return
	}

	articles, err := cbc.parseFeed(feed)
	if err != nil {
		cbc.logger.Warn("error parsing feed",
			zap.Error(err),
		)
		return
	}

	cbc.api.refreshArticles(articles)
}

func (cbc *CBCFeed) getFeed(ctx context.Context) (*gofeed.Feed, error) {
	req, err := http.NewRequest(http.MethodGet, cbc.path, nil)
	if err != nil {
		cbc.logger.Warn("error creating new request",
			zap.Error(err),
		)
		return nil, err
	}

	req.WithContext(ctx)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		cbc.logger.Warn("error performing request",
			zap.Error(err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cbc.logger.Info("received non-OK response",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, nil
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		cbc.logger.Warn("error parsing feed",
			zap.Error(err),
		)
		return nil, err
	}

	return feed, nil
}

func (cbc *CBCFeed) parseFeed(feed *gofeed.Feed) ([]*Article, error) {
	var articles []*Article

	for _, item := range feed.Items {
		if item.Title == "The National" {
			continue
		}

		// This creates the article, fills in description and an image (if present)
		article := cbc.parseDescription(item.Description)

		article.Name = fmt.Sprintf("articles/cbc/%s", item.GUID)
		article.Title = item.Title
		article.Link = item.Link

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

		for _, category := range item.Categories {
			article.Categories = append(article.Categories, category)
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func (cbc *CBCFeed) parseDescription(desc string) *Article {
	article := &Article{}
	descTokenizer := html.NewTokenizer(strings.NewReader(strings.TrimSpace(desc)))

	for {
		tokenType := descTokenizer.Next()

		// Either we are at the end, or the string was malformed. Either way we will exit the loop and return.
		if tokenType == html.ErrorToken {
			err := descTokenizer.Err()
			if err != io.EOF {
				cbc.logger.Warn("error tokenizing HTML",
					zap.Error(descTokenizer.Err()),
				)
			}
			break
		}

		if tokenType == html.StartTagToken {
			token := descTokenizer.Token()
			// The description is in the paragraph block
			if token.Data == "p" {
				tokenType = descTokenizer.Next()
				// Confirm that we are in the proper block
				if tokenType == html.TextToken {
					article.Description = descTokenizer.Token().Data
				}
			}
		}

		if tokenType == html.SelfClosingTagToken {
			token := descTokenizer.Token()
			if token.Data == "img" {
				article.Image = &Image{}

				for _, attr := range token.Attr {
					switch attr.Key {
					case "src":
						article.Image.Link = attr.Val
						article.Image.Name = strings.TrimPrefix(attr.Val, "https://")
					case "width":
						width, err := strconv.ParseInt(attr.Val, 10, 32)
						if err == nil {
							article.Image.Width = int32(width)
						} else {
							cbc.logger.Warn("error parsing width",
								zap.Error(err),
							)
						}
					case "height":
						height, err := strconv.ParseInt(attr.Val, 10, 32)
						if err == nil {
							article.Image.Height = int32(height)
						} else {
							cbc.logger.Warn("error parsing height",
								zap.Error(err),
							)
						}
					case "title":
						article.Image.Title = attr.Val
					}
				}
			}
		}
	}

	return article
}
