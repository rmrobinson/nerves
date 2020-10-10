package bridge

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

type mockBridge struct{}

func (m *mockBridge) SetDeviceState(context.Context, *Device, *DeviceState) error {
	return nil
}

var updateDeviceStateTests = []struct {
	name string
	req  *UpdateDeviceStateRequest

	expectedErr    error
	expectedResp   *Device
	expectedUpdate *DeviceUpdate
}{
	{
		name: "success case",
		req: &UpdateDeviceStateRequest{
			Id: "1232",
			State: &DeviceState{
				IsReachable: true,
				Binary:      &DeviceState_Binary{IsOn: true},
			},
		},
		expectedErr: nil,
		expectedResp: &Device{
			Id: "1232",
			State: &DeviceState{
				IsReachable: true,
				Binary:      &DeviceState_Binary{IsOn: true},
			},
		},
		expectedUpdate: &DeviceUpdate{
			Device: &Device{
				Id: "1232",
				State: &DeviceState{
					IsReachable: true,
					Binary:      &DeviceState_Binary{IsOn: true},
				},
			},
			BridgeId: "test",
		},
	},
	{
		name: "success case without change doesn't trigger update",
		req: &UpdateDeviceStateRequest{
			Id: "1232",
			State: &DeviceState{
				IsReachable: true,
				Binary:      &DeviceState_Binary{IsOn: true},
			},
		},
		expectedErr: nil,
		expectedResp: &Device{
			Id: "1232",
			State: &DeviceState{
				IsReachable: true,
				Binary:      &DeviceState_Binary{IsOn: true},
			},
		},
		expectedUpdate: nil,
	},
	{
		name: "bad request is rejected",
		req: &UpdateDeviceStateRequest{
			Id:    "1232",
			State: nil,
		},
		expectedErr:    ErrMissingParam.Err(),
		expectedResp:   nil,
		expectedUpdate: nil,
	},
	{
		name: "attempt to change reachability is rejected",
		req: &UpdateDeviceStateRequest{
			Id: "1232",
			State: &DeviceState{
				IsReachable: false,
				Binary:      &DeviceState_Binary{IsOn: true},
			},
		},
		expectedErr:    ErrNotSupported.Err(),
		expectedResp:   nil,
		expectedUpdate: nil,
	},
	{
		name: "attempt to change invalid device is rejected",
		req: &UpdateDeviceStateRequest{
			Id: "1235",
			State: &DeviceState{
				IsReachable: true,
				Binary:      &DeviceState_Binary{IsOn: true},
			},
		},
		expectedErr:    ErrDeviceNotFound.Err(),
		expectedResp:   nil,
		expectedUpdate: nil,
	},
}

func TestUpdateDeviceState(t *testing.T) {
	logger := zaptest.NewLogger(t)

	sbs := NewSyncBridgeService(logger, &Bridge{Id: "test"}, nil, &mockBridge{})

	for _, tt := range updateDeviceStateTests {
		t.Run(tt.name, func(t *testing.T) {
			testDevice := &Device{
				Id: "1232",
				State: &DeviceState{
					IsReachable: true,
					Binary:      &DeviceState_Binary{},
				},
			}

			sbs.devices = map[string]*Device{
				testDevice.Id: testDevice,
			}

			var wg sync.WaitGroup
			var update *Update
			updateSync := sbs.updates.NewSink()

			go func() {
				if tt.expectedUpdate == nil {
					return
				}

				wg.Add(1)
				u := <-updateSync.Messages()
				update = u.(*Update)
				updateSync.Close()
				wg.Done()
			}()

			resp, err := sbs.UpdateDeviceState(context.Background(), tt.req)
			assert.Equal(t, tt.expectedResp, resp)
			assert.Equal(t, tt.expectedErr, err)

			wg.Wait()
			if tt.expectedUpdate != nil {
				assert.Equal(t, Update_CHANGED, update.Action)
				assert.Equal(t, tt.expectedUpdate, update.GetDeviceUpdate())
			}
		})
	}
}
