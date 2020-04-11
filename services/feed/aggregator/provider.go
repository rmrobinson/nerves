package aggregator

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"github.com/rmrobinson/nerves/services/feed"
	"github.com/rmrobinson/nerves/services/feed/aggregator/translators"
	"go.uber.org/zap"
)

var (
	errMissingItem = errors.New("item missing")
)

type provider struct {
	logger    *zap.Logger
	url       string
	transport http.RoundTripper

	info *feed.FeedInfo
	// Collection of recently retrieved entries, keyed by the entry GUID
	// This isn't intended to be comprehensive, but instead something
	// to improve the performance of the 'diffing' of feed changes against.
	recentEntries map[string]*feed.Entry
}

func newProvider(logger *zap.Logger, transport http.RoundTripper, info *feed.FeedInfo) *provider {
	return &provider{
		logger:        logger,
		url:           info.Url,
		info:          info,
		recentEntries: map[string]*feed.Entry{},
	}
}

func (p *provider) refresh(ctx context.Context) ([]*feed.Entry, error) {
	f, err := p.retrieve(ctx)
	if err != nil {
		return nil, err
	}

	// Update the feed info section first
	// We treat anything already set as more authoritative than what comes back.
	if p.info.DisplayName == "" {
		p.info.DisplayName = f.Title
	}
	if p.info.Description == "" {
		p.info.Description = f.Description
	}

	// This is something we treat as authoritative though.
	p.info.LanguageCode = f.Language
	if p.info.CreateTime == nil {
		p.info.CreateTime = ptypes.TimestampNow()
	}

	if f.PublishedParsed != nil {
		updateTime, err := ptypes.TimestampProto(*f.PublishedParsed)
		if err != nil {
			return nil, err
		}
		p.info.UpdateTime = updateTime
	}

	p.info.LastCheckTime = ptypes.TimestampNow()

	// Go through and extract all the items from the feed
	var changed []*feed.Entry

	for _, item := range f.Items {
		if p.info.Config != nil && p.info.Config.RequireAuthor && item.Author == nil {
			continue
		}

		entry, err := p.entryFromItem(item)
		if err != nil {
			p.logger.Error("unable to convert item to entry",
				zap.String("provider", p.info.Name),
				zap.String("guid", item.GUID),
				zap.Error(err),
			)
			// Skip, can't do anything with it
			continue
		}

		if oldEntry, present := p.recentEntries[item.GUID]; present {
			newer := false
			// If the new item has an update time, confirm if it's after.
			if entry.UpdateTime != nil {
				if oldEntry.UpdateTime == nil {
					// assume it's newer
					newer = true
				} else if entry.UpdateTime.GetSeconds() > oldEntry.UpdateTime.GetSeconds() {
					newer = true
				}
			}

			if newer {
				changed = append(changed, entry)
				p.recentEntries[item.GUID] = entry
			}
		} else {
			changed = append(changed, entry)
			p.recentEntries[item.GUID] = entry
		}
	}

	return changed, nil
}

func (p *provider) retrieve(ctx context.Context) (*gofeed.Feed, error) {
	fp := gofeed.NewParser()
	fp.Client = &http.Client{
		Transport: p.transport,
	}

	if strings.Contains(strings.ToLower(p.url), "cbc") {
		fp.RSSTranslator = translators.NewCBC(p.logger)
	}

	return fp.ParseURLWithContext(p.url, ctx)
}

func (p *provider) entryFromItem(item *gofeed.Item) (*feed.Entry, error) {
	if item == nil {
		return nil, errMissingItem
	}

	e := &feed.Entry{
		Name:         fmt.Sprintf("%s/entries/%s", p.info.Name, uuid.New().String()),
		Title:        item.Title,
		Description:  item.Description,
		Link:         item.Link,
		LanguageCode: p.info.LanguageCode,
	}

	if item.Author != nil {
		e.Author = item.Author.Name
	}
	if item.Image != nil {
		e.Image = &feed.Image{
			Name:  fmt.Sprintf("%s/image/%s", e.Name, uuid.New().String()),
			Link:  item.Image.URL,
			Title: item.Title,
		}
	}

	for _, category := range item.Categories {
		e.Categories = append(e.Categories, category)
	}

	if item.PublishedParsed != nil {
		publishedTime, err := ptypes.TimestampProto(*item.PublishedParsed)
		if err != nil {
			return nil, err
		}
		e.CreateTime = publishedTime
	} else {
		e.CreateTime = ptypes.TimestampNow()
	}

	if item.UpdatedParsed != nil {
		updatedTime, err := ptypes.TimestampProto(*item.UpdatedParsed)
		if err != nil {
			return nil, err
		}
		e.UpdateTime = updatedTime
	}

	return e, nil
}
