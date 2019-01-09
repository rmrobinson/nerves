package main

import (
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/rmrobinson/nerves/services/domotics"
	"github.com/rmrobinson/nerves/services/domotics/bridge/mock"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	var (
		port   = flag.Int("port", 10102, "The port for the mockdeviced process to listen on")
		dbPath = flag.String("dbPath", "", "The FS path to read for the mock bridge (used if supplied)")
	)
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	bm := domotics.NewHub(logger)

	// If we have a persistent bridge, use it.
	// Otherwise use some randomly generated data.
	if len(*dbPath) > 0 {
		db := &domotics.BridgeDB{}
		err := db.Open(*dbPath)
		if err != nil {
			logger.Fatal("error opening db path",
				zap.String("db_path", *dbPath),
				zap.Error(err),
			)
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
		logger.Fatal("error initializing listener",
			zap.Error(err),
		)
	}
	defer lis.Close()
	logger.Info("listening",
		zap.String("local_addr", connStr),
	)

	api := domotics.NewAPI(logger, bm)

	grpcServer := grpc.NewServer()
	domotics.RegisterBridgeServiceServer(grpcServer, api)
	domotics.RegisterDeviceServiceServer(grpcServer, api)
	grpcServer.Serve(lis)
}
