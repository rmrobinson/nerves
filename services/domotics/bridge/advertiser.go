package bridge

import (
	"fmt"
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
// The relevant network info can be supplied during construction.
// When advertisements should be sent, call Run(); when shutting down simply call Shutdown().
type Advertiser struct {
	logger *zap.Logger

	id      string
	connStr string

	done chan bool
}

// NewAdvertiser sets up a new advertiser.
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
	usn := fmt.Sprintf("uuid:%s", a.id)
	location := fmt.Sprintf("grpc://%s", a.connStr)
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
