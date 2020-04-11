package main

import (
	"flag"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/mmcdole/gofeed"
	"github.com/rmrobinson/nerves/services/feed/aggregator/translators"
	"go.uber.org/zap"
)

func main() {
	var (
		feedURL = flag.String("url", "", "The URL of an RSS feed to dump")
	)

	flag.Parse()
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	fp := gofeed.NewParser()

	if strings.Contains(strings.ToLower(*feedURL), "cbc") {
		fp.RSSTranslator = translators.NewCBC(logger)
	}

	feed, err := fp.ParseURL(*feedURL)
	if err != nil {
		logger.Error("feed url is invalid",
			zap.Error(err),
		)
		return
	}

	logger.Debug("feed",
		zap.String("title", feed.Title),
		zap.String("description", feed.Description),
		zap.String("url", feed.Link),
		zap.Strings("categories", feed.Categories),
	)

	for _, item := range feed.Items {
		logger.Debug("entry",
			zap.String("title", item.Title),
			zap.String("description", item.Description),
			zap.String("guid", item.GUID),
			zap.String("link", item.Link),
			zap.Strings("categories", item.Categories),
			zap.String("published", item.Published),
		)
		if item.Image != nil {
			logger.Debug("image",
				zap.String("title", item.Image.Title),
				zap.String("url", item.Image.URL),
			)
		}
	}

	spew.Dump(feed.Items[0])
}
