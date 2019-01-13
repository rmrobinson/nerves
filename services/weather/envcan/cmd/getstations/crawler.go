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

type weatherStation struct {
	url      string
	city     string
	title    string
	province string
}

type crawler struct {
	logger *zap.Logger
}

func (c *crawler) getWeatherStations(ctx context.Context) []weatherStation {
	var stations []weatherStation
	for provinceCode := range provinceCodes {
		for i := 1; i < 200; i++ {
			path := fmt.Sprintf("https://weather.gc.ca/rss/city/%s-%d_e.xml", provinceCode, i)
			station, err := c.loadPath(ctx, path)
			if err == errPathNotFound {
				continue
			} else if err != nil {
				c.logger.Warn("error handling path",
					zap.String("path", path),
					zap.Error(err),
				)
				continue
			}

			station.province = provinceCode
			stations = append(stations, *station)
		}
	}

	return stations
}

func (c *crawler) loadPath(ctx context.Context, path string) (*weatherStation, error) {
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
		if resp.StatusCode == http.StatusNotFound {
			return nil, errPathNotFound
		}

		c.logger.Debug("received non-OK response",
			zap.Int("status_code", resp.StatusCode),
		)
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

	record := &weatherStation{
		url:   path,
		title: feed.Title,
	}

	if len(record.title) > 0 {
		record.city = strings.Split(record.title, "-")[0]
		record.city = strings.Split(record.city, "(")[0]
		record.city = strings.TrimSpace(record.city)
	}

	return record, nil
}
