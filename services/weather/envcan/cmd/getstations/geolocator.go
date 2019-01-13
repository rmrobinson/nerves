package main

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type geogratisSubItem struct {
	Code string `json:"code"`
}

type geogratisItem struct {
	Name      string           `json:"name"`
	Concise   geogratisSubItem `json:"concise"`
	Province  geogratisSubItem `json:"province"`
	Latitude  float64          `json:"latitude"`
	Longitude float64          `json:"longitude"`
}

type geogratisResponse struct {
	Items []*geogratisItem `json:"items"`
}

// See https://www.nrcan.gc.ca/earth-sciences/geography/place-names/tools-applications/9249 for details

type geogratisAPI struct {
	logger *zap.Logger
}

func (api *geogratisAPI) geocode(ctx context.Context, name string, provinceCode string) (*geogratisItem, error) {
	req, err := http.NewRequest(http.MethodGet, "http://geogratis.gc.ca/services/geoname/en/geonames.json", nil)
	if err != nil {
		api.logger.Warn("error creating new request",
			zap.Error(err),
		)
		return nil, err
	}

	q := req.URL.Query()
	q.Add("q", name)

	req.URL.RawQuery = q.Encode()
	req.WithContext(ctx)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		api.logger.Warn("error performing request",
			zap.Error(err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		api.logger.Info("received non-OK response",
			zap.Int("status_code", resp.StatusCode),
		)
		if resp.StatusCode == http.StatusNotFound {
			return nil, errPathNotFound
		}
		return nil, errUnhandledStatusCode
	}

	geogratisResp := &geogratisResponse{}
	err = json.NewDecoder(resp.Body).Decode(geogratisResp)
	if err != nil {
		api.logger.Info("error decoding response",
			zap.Error(err),
		)
		return nil, err
	}

	if len(geogratisResp.Items) < 1 {
		return nil, nil
	}

	for _, item := range geogratisResp.Items {
		if item.Province.Code == provinceCode {
			return item, nil
		}
	}

	return nil, nil
}
