package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/domotics/bridge/mock"
	"google.golang.org/grpc"
)

func main() {
	var (
		port   = flag.Int("port", 10102, "The port for the mockdeviced process to listen on")
		dbPath = flag.String("dbPath", "", "The FS path to read for the mock bridge (used if supplied)")
	)
	flag.Parse()

	bm := domotics.NewHub()

	// If we have a persistent bridge, use it.
	// Otherwise use some randomly generated data.
	if len(*dbPath) > 0 {
		db := &domotics.BridgeDB{}
		err := db.Open(*dbPath)
		if err != nil {
			log.Printf("Error opening db path %s: %s\n", *dbPath, err.Error())
			os.Exit(1)
		}
		defer db.Close()

		pbb := mock.NewPersistentBridge(db)
		bm.AddBridge(pbb, time.Hour)
		go pbb.Run()
	} else {
		msb := mock.NewSyncBridge()
		bm.AddBridge(msb, 5*time.Second)
		go msb.Run()

		mab := mock.NewAsyncBridge()
		bm.AddAsyncBridge(mab)
		go mab.Run()
	}

	connStr := fmt.Sprintf("%s:%d", "", *port)
	lis, err := net.Listen("tcp", connStr)
	if err != nil {
		log.Printf("Error initializing listener: %s\n", err.Error())
		os.Exit(1)
	}
	defer lis.Close()
	log.Printf("Listening on %s\n", connStr)

	api := domotics.NewAPI(bm)

	grpcServer := grpc.NewServer()
	domotics.RegisterBridgeServiceServer(grpcServer, api)
	domotics.RegisterDeviceServiceServer(grpcServer, api)
	grpcServer.Serve(lis)
}
