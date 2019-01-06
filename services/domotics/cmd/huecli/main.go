package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/rmrobinson/hue-go"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
)

// Full disclosure, the below logic is very gross.
// The hue-go library should support pairing with a bridge,
// that would clean the below logic up significantly.

func main() {
	var (
		dbPath = flag.String("dbPath", "", "The path to the DB to update")
		bridgeIP = flag.String("bridgeIP", "", "The Hue bridge IP to pair with")
	)

	flag.Parse()

	if len(*dbPath) < 1 {
		fmt.Printf("The Hue DB path must be specified\n")
		os.Exit(1)
	}

	db := &bridge.HueDB{}
	err := db.Open(*dbPath)
	if err != nil {
		fmt.Printf("Error opening Hue DB: %s\n", err.Error())
		os.Exit(1)
	}

	if len(*bridgeIP) < 1  {
		fmt.Printf("The bridge IP must be specified\n")
		os.Exit(1)
	}

	var b hue.Bridge

	if err = b.InitIP(*bridgeIP); err != nil {
		fmt.Printf("Unable to initialize supplied bridge address: %s\n", err.Error())
		return
	}

	fmt.Printf("Please press the 'Pair' button on the Hue bridge. Once pressed, type 'Y' to proceed\n")
	fmt.Print("> ")

	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Printf("Unable to read input: %s\n", err.Error())
		return
	}
	if input != "Y\n" {
		fmt.Printf("Non 'Y' character supplied, cancelling the pairing process\n")
		return
	}

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("Unable to read hostname: %s\n", err.Error())
		return
	}

	err = b.Pair("deviced", hostname[0:18])
	if err != nil {
		fmt.Printf("Error when paring: %s\n", err.Error())
		return
	}

	bridgeID := b.ID()
	key := b.Username

	if len(key) < 1 {
		fmt.Printf("Returned key is empty; not saving\n")
		return
	}

	fmt.Printf("Saving %s for bridge %s\n", key, bridgeID)
	db.SaveProfile(context.Background(), bridgeID, key)
}
