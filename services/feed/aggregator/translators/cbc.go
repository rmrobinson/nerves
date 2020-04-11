package translators

import (
	"errors"
	"io"
	"strings"

	"github.com/mmcdole/gofeed"
	"github.com/mmcdole/gofeed/rss"
	"go.uber.org/zap"
	"golang.org/x/net/html"
)

// CBC is an RSS feed translator that handles specific CBC content formatting.
type CBC struct {
	logger            *zap.Logger
	defaultTranslator *gofeed.DefaultRSSTranslator
}

// Translate takes a given CBC generic RSS feed and handles the CBC-specific content handling.
func (cbc *CBC) Translate(feed interface{}) (*gofeed.Feed, error) {
	rss, ok := feed.(*rss.Feed)
	if !ok {
		return nil, errors.New("cbc feed did not match the expected type of *rss.Feed")
	}

	f, err := cbc.defaultTranslator.Translate(rss)
	if err != nil {
		return nil, err
	}

	for itemIdx, item := range f.Items {
		image := item.Image
		description := item.Description

		descTokenizer := html.NewTokenizer(strings.NewReader(strings.TrimSpace(item.Description)))

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
						description = descTokenizer.Token().Data
					}
				}
			}

			if tokenType == html.SelfClosingTagToken {
				token := descTokenizer.Token()
				if token.Data == "img" {
					image = &gofeed.Image{}

					for _, attr := range token.Attr {
						switch attr.Key {
						case "src":
							image.URL = attr.Val
						case "title":
							image.Title = attr.Val
						}
					}
				}
			}
		}

		f.Items[itemIdx].Image = image
		f.Items[itemIdx].Description = description
	}

	return f, nil
}

// NewCBC creates a new version of the CBC feed translator
func NewCBC(logger *zap.Logger) *CBC {
	return &CBC{
		logger:            logger,
		defaultTranslator: &gofeed.DefaultRSSTranslator{},
	}
}
