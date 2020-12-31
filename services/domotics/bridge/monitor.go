package bridge

import (
	context "context"
	"strings"

	"go.uber.org/zap"

	"github.com/koron/go-ssdp"
)

// MonitorHandler describes the methods a monitor can invoke when certain conditions are hit.
// Alive() is called with the server UUID and connection string when a bridge is found
// Gone() is called with the bridge ID when a bridge announces it is going away.
type MonitorHandler interface {
	Alive(string, string, string)
	GoingAway(string)
}

// Monitor is used to track bridge discovery updates.
type Monitor struct {
	logger  *zap.Logger
	handler MonitorHandler

	logNonregisteredTypes bool
	types                 []string
}

// NewMonitor creates a new monitor
func NewMonitor(logger *zap.Logger, handler MonitorHandler, types []string) *Monitor {
	return &Monitor{
		logger:  logger,
		handler: handler,
		types:   types,
	}
}

// LogNonRegisteredTypes enables logging of all received SSDP packets, even if they do not match the registered type filter.
func (m *Monitor) LogNonRegisteredTypes() {
	m.logNonregisteredTypes = true
}

// Run begins the monitor, which will listen until the context is cancelled.
func (m *Monitor) Run(ctx context.Context) {
	ssdpMonitor := ssdp.Monitor{
		Alive: m.ssdpAlive,
	}

	ssdpMonitor.Start()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("context cancelled, done monitoring")
			ssdpMonitor.Close()
			return
		}
	}
}

func (m *Monitor) ssdpAlive(msg *ssdp.AliveMessage) {
	logger := m.logger.With(
		zap.String("type", msg.Type),
		zap.String("usn", msg.USN),
		zap.String("location", msg.Location),
	)

	// We don't handle other messages
	if m.types != nil {
		found := false

		for _, t := range m.types {
			if t == msg.Type {
				found = true
				break
			}
		}

		if !found {
			if m.logNonregisteredTypes {
				logger.Debug("skipping non-registered type advertisement")
			}
			return
		}
	}

	logger.Debug("node is alive")
	connStr := msg.Location
	if strings.HasPrefix(connStr, "grpc://") {
		connStr = strings.TrimPrefix(connStr, "grpc://")
	}

	m.handler.Alive(msg.Type, msg.USN, connStr)
}

func (m *Monitor) ssdpBye(msg *ssdp.ByeMessage) {
	logger := m.logger.With(
		zap.String("type", msg.Type),
		zap.String("usn", msg.USN),
	)

	// We don't handle other messages
	if msg.Type != typeHeader {
		logger.Debug("skipping non-bridge bye")
		return
	}

	logger.Debug("bridge is going away")

	m.handler.GoingAway(msg.USN)
}
