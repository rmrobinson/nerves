package main

import (
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rmrobinson/nanoleaf-go"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Service is a handler that will be informed of Nanoleaf SSDP broadcasts on the given UUID.
// The SSDP lifecycle will cause a gRPC bridge that communicates with the detected Nanoleaf panel
// to be created or shut down.
type Service struct {
	logger *zap.Logger
	id     string
	apiKey string

	client *nanoleaf.Client
	bridge *Nanoleaf

	advertiser *bridge.Advertiser
	listener   net.Listener
	server     *grpc.Server
}

// NewService creates a new Nanoleaf management service.
func NewService(logger *zap.Logger, id string, apiKey string) *Service {
	return &Service{
		logger: logger,
		id:     id,
		apiKey: apiKey,
	}
}

// Alive is called when a bridge is reporting itself as alive
func (s *Service) Alive(t string, id string, connStr string) {
	if id != s.id {
		return
	} else if t != "nanoleaf_aurora:light" {
		return
	}

	if s.bridge != nil {
		s.logger.Debug("received alive SSDP but bridge already set, ignoring")
		return
	}

	parsedConnStr, err := url.Parse(connStr)
	if err != nil {
		s.logger.Info("unable to parse conn str",
			zap.String("conn_str", connStr),
			zap.Error(err),
		)
		return
	}
	port := 80

	if len(parsedConnStr.Port()) > 0 {
		port, err = strconv.Atoi(parsedConnStr.Port())
		if err != nil {
			s.logger.Info("unable to parse port",
				zap.String("port", parsedConnStr.Port()),
				zap.String("conn_str", connStr),
				zap.Error(err),
			)
		}
		// We'll continue with the default port for now
	}

	s.listener, err = net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		s.logger.Error("error initializing listener",
			zap.Error(err),
		)
		return
	}
	s.logger.Info("listening",
		zap.String("local_addr", s.listener.Addr().String()),
	)

	s.client = nanoleaf.NewClient(&http.Client{}, parsedConnStr.Hostname(), port, s.apiKey)
	s.bridge = NewNanoleaf(s.logger, s.id, s.client)

	s.advertiser = bridge.NewAdvertiser(s.logger, s.id, s.listener.Addr().String())
	s.server = grpc.NewServer()

	bridge.RegisterBridgeServiceServer(s.server, s.bridge)
	bridge.RegisterPingServiceServer(s.server, s.advertiser)

	go s.server.Serve(s.listener)
	go s.advertiser.Run()
}

// GoingAway is called when a bridge is reporting itself as going aways
func (s *Service) GoingAway(id string) {
	s.advertiser.Shutdown()
	s.advertiser = nil

	s.server.GracefulStop()
	// the server stopping also closes the listener
	s.server = nil
	s.listener = nil

	s.bridge = nil
	s.client = nil
}
