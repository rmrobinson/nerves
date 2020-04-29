package main

import (
	"context"
	"flag"

	"github.com/rmrobinson/nerves/services/domotics"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func createBuilding(logger *zap.Logger, bc domotics.BuildingAdminServiceClient, name string, desc string) {
	req := &domotics.CreateBuildingRequest{
		Building: &domotics.Building{
			Name:        name,
			Description: desc,
		},
	}

	resp, err := bc.CreateBuilding(context.Background(), req)
	if err != nil {
		logger.Warn("unable to create building",
			zap.String("name", name),
			zap.Error(err),
		)
		return
	}

	logger.Info("created building",
		zap.String("id", resp.Id),
		zap.String("name", resp.Name),
		zap.String("desc", resp.Description),
	)
}
func createFloor(logger *zap.Logger, bc domotics.BuildingAdminServiceClient, name string, desc string, parentID string) {
	req := &domotics.CreateFloorRequest{
		BuildingId: parentID,
		Floor: &domotics.Floor{
			Name:        name,
			Description: desc,
		},
	}

	resp, err := bc.CreateFloor(context.Background(), req)
	if err != nil {
		logger.Warn("unable to create floor",
			zap.String("name", name),
			zap.Error(err),
		)
		return
	}

	logger.Info("created floor",
		zap.String("id", resp.Id),
		zap.String("name", resp.Name),
		zap.String("desc", resp.Description),
	)
}
func createRoom(logger *zap.Logger, bc domotics.BuildingAdminServiceClient, name string, desc string, parentID string) {
	req := &domotics.CreateRoomRequest{
		FloorId: parentID,
		Room: &domotics.Room{
			Name:        name,
			Description: desc,
		},
	}

	resp, err := bc.CreateRoom(context.Background(), req)
	if err != nil {
		logger.Warn("unable to create room",
			zap.String("name", name),
			zap.Error(err),
		)
		return
	}

	logger.Info("created room",
		zap.String("id", resp.Id),
		zap.String("name", resp.Name),
		zap.String("desc", resp.Description),
	)
}

func main() {
	var (
		addr = flag.String("addr", "", "The address to connect to")
		mode = flag.String("mode", "", "The mode of this tool")
		name = flag.String("name", "", "The device name to set")
		desc = flag.String("desc", "", "The description to set")
		p    = flag.String("parent", "", "The parent of the object")
	)

	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(*addr, opts...)
	if err != nil {
		logger.Fatal("unable to connect",
			zap.String("addr", *addr),
			zap.Error(err),
		)
		return
	}

	buildingClient := domotics.NewBuildingAdminServiceClient(conn)

	switch *mode {
	case "createBuilding":
		createBuilding(logger, buildingClient, *name, *desc)
	case "createFloor":
		createFloor(logger, buildingClient, *name, *desc, *p)
	case "createRoom":
		createRoom(logger, buildingClient, *name, *desc, *p)
	default:
		logger.Debug("unknown command specified")
	}
}
