package main

import (
	"context"
	"flag"

	"github.com/rmrobinson/nerves/services/domotics/building"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func createBuilding(logger *zap.Logger, bc building.BuildingAdminServiceClient, name string, desc string) {
	req := &building.CreateBuildingRequest{
		Building: &building.Building{
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
func listBuildings(logger *zap.Logger, bc building.BuildingServiceClient) {
	req := &building.ListBuildingsRequest{}

	resp, err := bc.ListBuildings(context.Background(), req)
	if err != nil {
		logger.Warn("unable to list building",
			zap.Error(err),
		)
		return
	}

	logger.Info("retrieved buildings")

	for _, b := range resp.Buildings {
		logger.Info("building found",
			zap.String("id", b.Id),
			zap.String("name", b.Name),
			zap.String("desc", b.Description),
		)
	}
}
func createFloor(logger *zap.Logger, bc building.BuildingAdminServiceClient, name string, desc string, parentID string) {
	req := &building.CreateFloorRequest{
		BuildingId: parentID,
		Floor: &building.Floor{
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
func listFloors(logger *zap.Logger, bc building.BuildingServiceClient, id string) {
	req := &building.ListFloorsRequest{
		BuildingId: id,
	}

	resp, err := bc.ListFloors(context.Background(), req)
	if err != nil {
		logger.Warn("unable to list floors",
			zap.String("building_id", id),
			zap.Error(err),
		)
		return
	}

	logger.Info("retrieved floors",
		zap.String("building_id", id),
	)

	for _, f := range resp.Floors {
		logger.Info("floor found",
			zap.String("id", f.Id),
			zap.String("name", f.Name),
			zap.String("desc", f.Description),
		)
		for _, r := range f.Rooms {
			logger.Info(" room found",
				zap.String("id", r.Id),
				zap.String("name", r.Name),
				zap.String("description", r.Description),
			)
		}
	}
}
func createRoom(logger *zap.Logger, bc building.BuildingAdminServiceClient, name string, desc string, parentID string) {
	req := &building.CreateRoomRequest{
		FloorId: parentID,
		Room: &building.Room{
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
func linkBridge(logger *zap.Logger, bc building.BuildingAdminServiceClient, bridgeID string, buildingID string) {
	req := &building.AddBridgeRequest{
		ParentId: buildingID,
		BridgeId: bridgeID,
	}

	resp, err := bc.AddBuildingBridge(context.Background(), req)
	if err != nil {
		logger.Warn("unable to add bridge",
			zap.String("bridge_id", bridgeID),
			zap.Error(err),
		)
		return
	}

	logger.Info("added bridge to building",
		zap.String("bridge_id", bridgeID),
		zap.Int("bridge_count", len(resp.Bridges)),
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

	buildingAdminClient := building.NewBuildingAdminServiceClient(conn)
	buildingClient := building.NewBuildingServiceClient(conn)

	switch *mode {
	case "createBuilding":
		createBuilding(logger, buildingAdminClient, *name, *desc)
	case "listBuildings":
		listBuildings(logger, buildingClient)
	case "createFloor":
		createFloor(logger, buildingAdminClient, *name, *desc, *p)
	case "listFloors":
		listFloors(logger, buildingClient, *p)
	case "createRoom":
		createRoom(logger, buildingAdminClient, *name, *desc, *p)
	case "linkBridge":
		linkBridge(logger, buildingAdminClient, *name, *p)
	default:
		logger.Debug("unknown command specified")
	}
}
