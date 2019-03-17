package policy

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/rmrobinson/nerves/services/domotics"
	"go.uber.org/zap"
)

func (a *Action) execute(ctx context.Context, logger *zap.Logger, state *State) {
	switch a.Type {
	case Action_LOG:
		logger.Info("executing action",
			zap.String("name", a.Name),
		)
	case Action_DEVICE:
		logger.Debug("received device action",
			zap.String("name", a.Name),
		)

		deviceAction := &DeviceAction{}
		err := ptypes.UnmarshalAny(a.Details, deviceAction)
		if err != nil {
			logger.Info("error unmarshaling details",
				zap.String("name", a.Name),
				zap.Error(err),
			)
			return
		}

		if device, ok := state.deviceState[deviceAction.Id]; ok {
			proto.Merge(device.State, deviceAction.State)

			// We don't save the result as the monitor channel will pick up the update when it is broadcast.
			_, err := state.deviceClient.SetDeviceState(ctx, &domotics.SetDeviceStateRequest{
				Id: deviceAction.Id,
				State: device.State,
			})
			if err != nil {
				logger.Info("error setting device state",
					zap.String("name", a.Name),
					zap.Error(err),
				)
			}
		} else {
			logger.Info("action with missing device id",
				zap.String("name", a.Name),
				zap.String("device_id", deviceAction.Id),
			)
		}
	}
}
