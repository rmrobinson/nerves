package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type weatherInfo struct {
	url string
	title string
	name string

	latitude float64
	longitude float64
	siteType string
	siteProvinceCode string
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	c := crawler{
		logger: logger,
	}
	geoAPI := &geogratisAPI{
		logger: logger,
	}

	sites := c.getWeatherSites(context.Background())

	var records []weatherInfo
	for _, site := range sites {
		geocoderResults, err := geoAPI.geocode(context.Background(), site.city)
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
			url: site.url,
			title: site.title,
			name: site.city,
			latitude: geocoderResults.Latitude,
			longitude: geocoderResults.Longitude,
			siteType: geocoderResults.Concise.Code,
			siteProvinceCode: geocoderResults.Province.Code,
		}

		records = append(records, record)
	}

	for _, record := range records {
		fmt.Printf("%v\n", record)
	}
}
