package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"

	"go.uber.org/zap"
)

var provinceCodes = map[string]string{
	"ab": "48",
	"bc": "59",
	"mb": "46",
	"nb": "13",
	"nl": "10",
	"ns": "12",
	"nt": "61",
	"nu": "62",
	"on": "35",
	"pe": "11",
	"qc": "24",
	"sk": "47",
	"yt": "60",
}

type weatherInfo struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Name  string `json:"name"`

	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	SiteType         string  `json:"site_type"`
	SiteProvinceCode string  `json:"site_province_code"`
}

func main() {
	var (
		outputPath = flag.String("output", "/tmp/weather.json", "The path to save the results to")
	)
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	f, err := os.Create(*outputPath)
	if err != nil {
		logger.Fatal("unable to create results file",
			zap.Error(err),
		)
		return
	}

	c := crawler{
		logger: logger,
	}
	geoAPI := &geogratisAPI{
		logger: logger,
	}

	stations := c.getWeatherStations(context.Background())

	var records []weatherInfo
	for _, site := range stations {
		geocoderResults, err := geoAPI.geocode(context.Background(), site.city, provinceCodes[site.province])
		if err != nil {
			logger.Warn("error geocoding",
				zap.String("city_name", site.city),
				zap.Error(err),
			)
			continue
		} else if geocoderResults == nil {
			logger.Info("no results found",
				zap.String("city_name", site.city),
			)
			continue
		}

		record := weatherInfo{
			URL:              site.url,
			Title:            site.title,
			Name:             site.city,
			Latitude:         geocoderResults.Latitude,
			Longitude:        geocoderResults.Longitude,
			SiteType:         geocoderResults.Concise.Code,
			SiteProvinceCode: geocoderResults.Province.Code,
		}

		records = append(records, record)
	}

	je := json.NewEncoder(f)
	for _, record := range records {
		err = je.Encode(record)
		if err != nil {
			logger.Info("error writing record",
				zap.Error(err),
			)
		}
	}
}
