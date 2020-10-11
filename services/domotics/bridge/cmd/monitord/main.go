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
func (cm *ConsoleMonitor) Alive(id string, connStr string) {
	fmt.Printf("%s available at %s\n", id, connStr)
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

	m := bridge.NewMonitor(logger, cm)

	logger.Info("monitoring for bridges")
	m.Run(context.Background())
}
