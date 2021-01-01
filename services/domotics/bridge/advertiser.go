package bridge

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/koron/go-ssdp"
	"go.uber.org/zap"
)

const (
	advertiseInterval = time.Second * 10
	typeHeader        = "falnet_nerves:bridge"
	maxAgeHeader      = 1800
	serverHeader      = "Falnet NDP/0.1"
)

// Advertiser encapsulates the logic required to advertise this bridge to the network.
// The relevant connection info will be supplied during construction.
// If the 'any' IP is supplied the first global unicast IP of the host will be chosen for advertising.
// When advertisements should be sent, call Run(); when shutting down simply call Shutdown().
type Advertiser struct {
	logger *zap.Logger

	id      string
	connStr string

	done chan bool
}

// NewAdvertiser sets up a new advertiser.
// ID will have the 'uuid' prefix added before broadcasting.
// connStr must be a <host>:<port> formatted string that will be included as the location to advertise to.
// If an 'any' IP, i.e 0.0.0.0 or [::] is supplied as the host it will be converted to the first global unicast
// IP address on the system so non-local endpoints can properly locate the endpoint.
func NewAdvertiser(logger *zap.Logger, id string, connStr string) *Advertiser {
	return &Advertiser{
		logger:  logger,
		id:      id,
		connStr: connStr,
		done:    make(chan bool),
	}
}

// Run begins the advertisement loop. Execute in a goroutine as this will not return.
func (a *Advertiser) Run() {
	connHost, connPort, err := net.SplitHostPort(a.connStr)
	if err != nil {
		a.logger.Error("connStr is malformed, cannot split so can't advertise",
			zap.Error(err),
		)
	}

	locationAddr := connHost

	connIP := net.ParseIP(connHost)
	if connIP == nil {
		a.logger.Error("supplied connStr host can't be parsed as an IP so can't advertise")
		return
	}

	if connIP.IsUnspecified() {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			a.logger.Error("unable to get interface addresses",
				zap.Error(err),
			)
			return
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.IsLoopback() {
					continue
				}

				if ipnet.IP.IsGlobalUnicast() {
					locationAddr = ipnet.IP.String()
				}
			}
		}
	}

	usn := fmt.Sprintf("uuid:%s", a.id)
	location := fmt.Sprintf("grpc://%s:%s", locationAddr, connPort)
	ad, err := ssdp.Advertise(typeHeader, usn, location, serverHeader, maxAgeHeader)
	if err != nil {
		a.logger.Error("unable to create advertiser",
			zap.Error(err),
		)
		return
	}
	defer ad.Close()

	aliveTick := time.NewTicker(advertiseInterval)
	defer aliveTick.Stop()

	a.logger.Info("advertising service over ssdp",
		zap.String("usn", usn),
		zap.String("location", location),
	)

	for {
		select {
		case <-a.done:
			ad.Bye()
		case <-aliveTick.C:
			ad.Alive()
		}
	}
}

// Shutdown shuts this advertiser down.
func (a *Advertiser) Shutdown() {
	a.done <- true
}

// Ping satisfies the ping service interface to allow this to respond to health checks.
func (a *Advertiser) Ping(context.Context, *PingRequest) (*Pong, error) {
	return &Pong{}, nil
}
