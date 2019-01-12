package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
)

var (
	errPathNotFound        = errors.New("path not found")
	errUnhandledStatusCode = errors.New("unhandled status code")
)

var provinceCodes = []string{
	"ab",
	"bc",
	"mb",
	"nb",
	"nl",
	"ns",
	"nt",
	"nu",
	"on",
	"pe",
	"qc",
	"sk",
	"yt",
}

type weatherSite struct {
	url   string
	city  string
	title string
}

type crawler struct {
	logger *zap.Logger
}

func (c *crawler) getWeatherSites(ctx context.Context) []weatherSite {
	var results []weatherSite
	for _, provinceCode := range provinceCodes {
		failCount := 0
		for i := 1; i < 500; i++ {
			if failCount > 3 {
				c.logger.Info("received 3 fails for province, skipping",
					zap.String("province_code", provinceCode),
				)
				break
			}
			path := fmt.Sprintf("https://weather.gc.ca/rss/city/%s-%d_e.xml", provinceCode, i)
			record, err := c.loadPath(ctx, path)
			if err == errPathNotFound {
				failCount++
				continue
			} else if err != nil {
				c.logger.Warn("error handling path",
					zap.String("path", path),
					zap.Error(err),
				)
				continue
			}

			failCount = 0
			results = append(results, *record)
		}
	}

	return results
}

func (c *crawler) loadPath(ctx context.Context, path string) (*weatherSite, error) {
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		c.logger.Warn("error creating new request",
			zap.Error(err),
		)
		return nil, err
	}

	req.WithContext(ctx)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.logger.Warn("error performing request",
			zap.Error(err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Info("received non-OK response",
			zap.Int("status_code", resp.StatusCode),
		)
		if resp.StatusCode == http.StatusNotFound {
			return nil, errPathNotFound
		}
		return nil, errUnhandledStatusCode
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		c.logger.Warn("error parsing feed",
			zap.Error(err),
		)
		return nil, err
	}

	record := &weatherSite{
		url:   path,
		title: feed.Title,
	}

	if len(record.title) > 0 {
		record.city = strings.TrimSpace(strings.Split(record.title, "-")[0])
	}

	return record, nil
}
