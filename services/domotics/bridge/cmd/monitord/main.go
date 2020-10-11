package main

import (
	"context"
	"fmt"

	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
)

// ConsoleMonitor is a CLI-based implementation of the monitor.
// Key actions will be logged to the console.
type ConsoleMonitor struct{}

// Alive is called when a bridge is reporting itself as alive
func (cm *ConsoleMonitor) Alive(t string, id string, connStr string) {
	fmt.Printf("%s (type %s) available at %s\n", id, t, connStr)
}

// GoingAway is called when a bridge is reporting itself as going aways
func (cm *ConsoleMonitor) GoingAway(id string) {
	fmt.Printf("%s is going away\n", id)
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	cm := &ConsoleMonitor{}

	m := bridge.NewMonitor(logger, cm, []string{"falnet_nerves:bridge", "nanoleaf_aurora:light"})

	logger.Info("listening for advertisements")
	m.Run(context.Background())
}
