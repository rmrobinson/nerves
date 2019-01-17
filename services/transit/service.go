package transit

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

// Service is a really simple service that retrieves a transit feed
type Service struct {
	logger *zap.Logger
}

// NewService creates a new service
func NewService(logger *zap.Logger) *Service {
	return &Service{
		logger: logger,
	}
}

// GetRealtimeFeed retrieves the GTFS realtime data.
func (s *Service) GetRealtimeFeed(ctx context.Context, path string) (*FeedMessage, error) {
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		s.logger.Warn("error creating new request",
			zap.Error(err),
		)
		return nil, err
	}

	req.WithContext(ctx)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Warn("error performing request",
			zap.Error(err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Info("received non-OK response",
			zap.Int("status_code", resp.StatusCode),
		)
		return nil, nil
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.logger.Warn("error reading body",
			zap.Error(err),
		)
		return nil, err
	}

	feed := &FeedMessage{}
	err = proto.Unmarshal(data, feed)
	if err != nil {
		s.logger.Warn("error unmarshaling body",
			zap.Error(err),
		)
		return nil, err
	}

	return feed, nil
}
