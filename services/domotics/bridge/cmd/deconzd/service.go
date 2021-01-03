package main

import (
	"context"
	"net/url"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/rmrobinson/deconz-go"
	"github.com/rmrobinson/nerves/lib/stream"
	"github.com/rmrobinson/nerves/services/domotics/bridge"
	"go.uber.org/zap"
	"google.golang.org/grpc/peer"
)

// Service represents a handle to a single Service REST API endpoint
type Service struct {
	logger *zap.Logger

	c    *deconz.Client
	done bool

	updates *stream.Source

	brInfo *bridge.Bridge

	lightUniqueIDsToPathIDs  map[string]string
	sensorUniqueIDsToPathIDs map[string]string
	wsIP                     string
	wsPort                   int
}

// NewService creates a new instance of the Deconz implementation of the Bridge gRPC contract.
func NewService(logger *zap.Logger, c *deconz.Client) *Service {
	return &Service{
		logger:                   logger,
		c:                        c,
		brInfo:                   &bridge.Bridge{},
		updates:                  stream.NewSource(logger),
		lightUniqueIDsToPathIDs:  map[string]string{},
		sensorUniqueIDsToPathIDs: map[string]string{},
	}
}

// Setup ensures this service is set up properly before running.
func (s *Service) Setup(ctx context.Context) error {
	b, err := s.c.GetGatewayState(ctx)
	if err != nil {
		s.logger.Error("unable to get deconz gateway state",
			zap.Error(err),
		)
		return err
	}

	// We need to seed the map of path IDs to device IDs
	lights, err := s.c.GetLights(ctx)
	if err != nil {
		s.logger.Error("unable to get lights",
			zap.Error(err),
		)
		return err
	}

	for id, light := range lights {
		// Light 1 is the Zigbee controller so don't mark it as available
		if id == "1" {
			continue
		}
		s.lightUniqueIDsToPathIDs[light.UniqueID] = id
	}

	sensors, err := s.c.GetSensors(ctx)
	if err != nil {
		s.logger.Error("unable to get sensors",
			zap.Error(err),
		)
		return err
	}

	for id, sensor := range sensors {
		s.sensorUniqueIDsToPathIDs[sensor.UniqueID] = id
	}

	s.brInfo.Id = b.GatewayID
	s.brInfo.ModelId = lights["1"].ModelID
	s.brInfo.Manufacturer = lights["1"].Manufacturer
	s.wsIP = b.IP
	s.wsPort = b.WebsocketPort

	return nil
}

// Run the deconz gateway. This will block indefinitely as it waits for updates from the bridge.
func (s *Service) Run() error {
	wsu := url.URL{
		Scheme: "ws",
		Host:   s.wsIP + ":" + strconv.Itoa(s.wsPort),
	}

	wsc, _, err := websocket.DefaultDialer.Dial(wsu.String(), nil)
	if err != nil {
		s.logger.Error("err creating websocket, aborting",
			zap.Error(err),
		)
		return err
	}
	defer wsc.Close()

	for {
		msg := &deconz.WebsocketUpdate{}
		err := wsc.ReadJSON(msg)

		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			s.logger.Error("err reading from websocket, aborting",
				zap.Error(err),
			)
			return err
		} else if s.done {
			return nil
		}

		switch msg.Meta.Event {
		case "added":
			if msg.Meta.Resource == "lights" {
				s.lightUniqueIDsToPathIDs[msg.Meta.UniqueID] = msg.Meta.ResourceID
				device := lightToDevice(msg.Light)
				s.updates.SendMessage(&bridge.Update{
					Action: bridge.Update_ADDED,
					Update: &bridge.Update_DeviceUpdate{
						DeviceUpdate: &bridge.DeviceUpdate{
							Device:   device,
							DeviceId: msg.Meta.UniqueID,
							BridgeId: s.brInfo.Id,
						},
					},
				})
			} else if msg.Meta.Resource == "sensors" {
				s.sensorUniqueIDsToPathIDs[msg.Meta.UniqueID] = msg.Meta.ResourceID
				device := sensorToDevice(msg.Sensor)
				s.updates.SendMessage(&bridge.Update{
					Action: bridge.Update_ADDED,
					Update: &bridge.Update_DeviceUpdate{
						DeviceUpdate: &bridge.DeviceUpdate{
							Device:   device,
							DeviceId: msg.Meta.UniqueID,
							BridgeId: s.brInfo.Id,
						},
					},
				})
			} else {
				s.logger.Debug("skipping unhandled resource type",
					zap.String("action", "added"),
					zap.String("resource_type", msg.Meta.Resource),
					zap.String("device_id", msg.Meta.UniqueID),
				)
			}
		case "changed":
			// This isn't really optimized but for consistency it's best to get data right from the source.
			if msg.Meta.Resource == "lights" || msg.Meta.Resource == "sensors" {
				device, err := s.GetDevice(context.Background(), &bridge.GetDeviceRequest{Id: msg.Meta.UniqueID})
				if err != nil {
					s.logger.Error("unable to retrieve device on update event",
						zap.String("device_id", msg.Meta.UniqueID),
						zap.Error(err),
					)
					continue
				}

				s.updates.SendMessage(&bridge.Update{
					Action: bridge.Update_CHANGED,
					Update: &bridge.Update_DeviceUpdate{
						DeviceUpdate: &bridge.DeviceUpdate{
							Device:   device,
							DeviceId: msg.Meta.UniqueID,
							BridgeId: s.brInfo.Id,
						},
					},
				})
			} else {
				s.logger.Debug("skipping unhandled resource type",
					zap.String("action", "changed"),
					zap.String("resource_type", msg.Meta.Resource),
					zap.String("device_id", msg.Meta.UniqueID),
				)
			}
		case "deleted":
			if msg.Meta.Resource == "lights" {
				delete(s.lightUniqueIDsToPathIDs, msg.Meta.UniqueID)
				s.updates.SendMessage(&bridge.Update{
					Action: bridge.Update_REMOVED,
					Update: &bridge.Update_DeviceUpdate{
						DeviceUpdate: &bridge.DeviceUpdate{
							DeviceId: msg.Meta.UniqueID,
							BridgeId: s.brInfo.Id,
						},
					},
				})
			} else if msg.Meta.Resource == "sensors" {
				delete(s.sensorUniqueIDsToPathIDs, msg.Meta.UniqueID)
				s.updates.SendMessage(&bridge.Update{
					Action: bridge.Update_REMOVED,
					Update: &bridge.Update_DeviceUpdate{
						DeviceUpdate: &bridge.DeviceUpdate{
							DeviceId: msg.Meta.UniqueID,
							BridgeId: s.brInfo.Id,
						},
					},
				})
			} else {
				s.logger.Debug("skipping unhandled resource type",
					zap.String("action", "deleted"),
					zap.String("resource_type", msg.Meta.Resource),
					zap.String("device_id", msg.Meta.UniqueID),
				)
			}
		default:
			s.logger.Debug("skipping unhandled update event",
				zap.String("type", msg.Meta.Event),
			)
			continue
		}
	}
}

// GetBridge retrieves the bridge info of this service.
func (s *Service) GetBridge(ctx context.Context, req *bridge.GetBridgeRequest) (*bridge.Bridge, error) {
	b, err := s.c.GetGatewayState(ctx)
	if err != nil {
		s.logger.Error("unable to get gateway state",
			zap.Error(err),
		)
		return nil, bridge.ErrInternal.Err()
	}

	devices, err := s.ListDevices(ctx, &bridge.ListDevicesRequest{})
	if err != nil {
		s.logger.Error("unable to get device state",
			zap.Error(err),
		)
		// This is already a gRPC error so we pass it along.
		return nil, err
	}

	return &bridge.Bridge{
		Id:           b.GatewayID,
		Manufacturer: s.brInfo.Manufacturer,
		ModelId:      s.brInfo.ModelId,
		Config: &bridge.BridgeConfig{
			Name:     b.Name,
			Timezone: b.Timezone,
			Address: &bridge.Address{
				Ip: &bridge.Address_Ip{
					Host:    b.IP,
					Netmask: b.Netmask,
					Gateway: b.GatewayIP,
				},
			},
		},
		State: &bridge.BridgeState{
			IsPaired: true,
			Version: &bridge.Version{
				Api: b.APIVersion,
				Sw:  b.SoftwareVersion,
			},
			Zigbee: &bridge.BridgeState_Zigbee{
				Channel: int32(b.ZigbeeChannel),
			},
		},
		Devices: devices.Devices,
	}, nil
}

// ListDevices retrieves all registered devices.
func (s *Service) ListDevices(ctx context.Context, req *bridge.ListDevicesRequest) (*bridge.ListDevicesResponse, error) {
	devices := []*bridge.Device{}

	lights, err := s.c.GetLights(ctx)
	if err != nil {
		s.logger.Error("unable to get lights",
			zap.Error(err),
		)
		return nil, bridge.ErrInternal.Err()
	}

	for id, light := range lights {
		// Light "1" is actually the deCONZ hardware, so don't add it.
		if id == "1" {
			continue
		}
		devices = append(devices, lightToDevice(&light))
	}

	sensors, err := s.c.GetSensors(ctx)
	if err != nil {
		s.logger.Error("unable to get sensors",
			zap.Error(err),
		)
		return nil, bridge.ErrInternal.Err()
	}

	for _, sensor := range sensors {
		devices = append(devices, sensorToDevice(&sensor))
	}

	return &bridge.ListDevicesResponse{
		Devices: devices,
	}, nil
}

// GetDevice retrieves the specified device.
func (s *Service) GetDevice(ctx context.Context, req *bridge.GetDeviceRequest) (*bridge.Device, error) {
	if id, found := s.lightUniqueIDsToPathIDs[req.Id]; found {
		light, err := s.c.GetLight(ctx, id)
		if err != nil {
			s.logger.Error("unable to get light",
				zap.String("device_id", req.Id),
				zap.String("path_id", id),
				zap.Error(err),
			)
			return nil, bridge.ErrInternal.Err()
		}

		return lightToDevice(light), nil
	}

	if id, found := s.sensorUniqueIDsToPathIDs[req.Id]; found {
		sensor, err := s.c.GetSensor(ctx, id)
		if err != nil {
			s.logger.Error("unable to get sensor",
				zap.String("device_id", req.Id),
				zap.String("path_id", id),
				zap.Error(err),
			)
			return nil, bridge.ErrInternal.Err()
		}

		return sensorToDevice(sensor), nil
	}

	return nil, bridge.ErrDeviceNotFound.Err()
}

// UpdateDeviceConfig allows the configuration of the device to be changed.
func (s *Service) UpdateDeviceConfig(ctx context.Context, req *bridge.UpdateDeviceConfigRequest) (*bridge.Device, error) {
	if len(req.Id) < 1 || req.Config == nil {
		return nil, bridge.ErrMissingParam.Err()
	}

	device, err := s.GetDevice(ctx, &bridge.GetDeviceRequest{Id: req.Id})
	if err != nil {
		s.logger.Error("unable to get device ahead of config change",
			zap.String("device_id", req.Id),
			zap.Error(err),
		)
		// This is already a gRPC error
		return nil, err
	}

	if device.Config.String() == req.Config.String() {
		s.logger.Debug("skipping device config update since desired state already present",
			zap.String("device_id", req.Id),
		)
		return device, nil
	}

	if id, found := s.lightUniqueIDsToPathIDs[req.Id]; found {
		err = s.c.SetLightConfig(ctx, id, &deconz.SetLightConfigRequest{
			Name: req.Config.Name,
		})

		if err != nil {
			s.logger.Error("unable to set light config",
				zap.String("device_id", req.Id),
				zap.String("path_id", id),
				zap.Error(err),
			)
			return nil, bridge.ErrInternal.Err()
		}

		device.Config.Name = req.Config.Name
		return device, nil
	}

	if id, found := s.sensorUniqueIDsToPathIDs[req.Id]; found {
		err = s.c.SetSensor(ctx, id, &deconz.SetSensorRequest{
			Name: req.Config.Name,
		})

		if err != nil {
			s.logger.Error("unable to set sensor config",
				zap.String("device_id", req.Id),
				zap.String("path_id", id),
				zap.Error(err),
			)
			return nil, bridge.ErrInternal.Err()
		}

		device.Config.Name = req.Config.Name

		// The Deconz websocket doesn't receive config updates (go figure) so generate the update msg here.
		s.updates.SendMessage(&bridge.Update{
			Action: bridge.Update_CHANGED,
			Update: &bridge.Update_DeviceUpdate{
				DeviceUpdate: &bridge.DeviceUpdate{
					Device:   device,
					DeviceId: device.Id,
					BridgeId: s.brInfo.Id,
				},
			},
		})

		return device, nil
	}

	return nil, bridge.ErrDeviceNotFound.Err()
}

// UpdateDeviceState updates the specified device with the provided state.
func (s *Service) UpdateDeviceState(ctx context.Context, req *bridge.UpdateDeviceStateRequest) (*bridge.Device, error) {
	if len(req.Id) < 1 || req.State == nil || req.State.Binary == nil {
		return nil, bridge.ErrMissingParam.Err()
	} else if req.State.IsReachable == false {
		return nil, bridge.ErrNotSupported.Err()
	}

	if id, found := s.lightUniqueIDsToPathIDs[req.Id]; found {
		light, err := s.c.GetLight(ctx, id)
		if err != nil {
			s.logger.Error("unable to get light",
				zap.String("device_id", req.Id),
				zap.String("path_id", id),
				zap.Error(err),
			)
			return nil, bridge.ErrInternal.Err()
		}

		stateReq := &deconz.SetLightStateRequest{
			On: req.State.Binary.IsOn,
		}

		if req.State.Range != nil {
			stateReq.Brightness = int(req.State.Range.Value)
		}
		if req.State.ColorHsb != nil {
			// If we already have things in XY, convert to XY ahead of writing
			if light.State.ColorMode == "xy" {
				s := float64(req.State.ColorHsb.Saturation) / float64(100.0)
				b := float64(req.State.ColorHsb.Brightness) / float64(100.0)
				c := colorful.Hsv(float64(req.State.ColorHsb.Hue), s, b)

				x, y, bri := getHueXYBrightnessFromColor(c, light.ModelID)

				stateReq.Brightness = int(bri)
				stateReq.XY = []float64{x, y}
			} else {
				// Convert from 360 degress/0-100% to the supported ranges.
				stateReq.Hue = int((float64(req.State.ColorHsb.Hue) * float64(65535.0)) / float64(360.0))
				stateReq.Saturation = int((float64(req.State.ColorHsb.Saturation) * float64(255.0)) / float64(100.0))
				stateReq.Brightness = int((float64(req.State.ColorHsb.Brightness) * float64(255.0)) / float64(100.0))
			}
		} else if req.State.ColorRgb != nil {
			c := colorful.Color{
				R: float64(req.State.ColorRgb.Red) / float64(255.0),
				G: float64(req.State.ColorRgb.Green) / float64(255.0),
				B: float64(req.State.ColorRgb.Blue) / float64(255.0),
			}

			x, y, bri := getHueXYBrightnessFromColor(c, light.ModelID)

			stateReq.Brightness = int(bri)
			stateReq.XY = []float64{x, y}
		} else if req.State.ColorTemperature != 0 {
			stateReq.CT = int(req.State.ColorTemperature)
		}

		err = s.c.SetLightState(ctx, id, stateReq)

		if err != nil {
			s.logger.Error("unable to set light config",
				zap.String("device_id", req.Id),
				zap.String("path_id", id),
				zap.Error(err),
			)
			return nil, bridge.ErrInternal.Err()
		}

		// Since the bridge may adjust some of the values, we get the device back to return.
		upDevice, err := s.GetDevice(ctx, &bridge.GetDeviceRequest{Id: req.Id})
		if err != nil {
			s.logger.Error("unable to get device after state change",
				zap.String("device_id", req.Id),
				zap.Error(err),
			)

			// We don't return an error here since the update already succeeded.
			// Absent more information we assume the did not apply and return that.
			// There will shortly be an update triggered which should refresh the values anyways.
			return lightToDevice(light), nil
		}

		return upDevice, nil
	}

	if _, found := s.sensorUniqueIDsToPathIDs[req.Id]; found {
		s.logger.Info("received state change request for sensor, not supported",
			zap.String("unique_id", req.Id),
		)
		return nil, bridge.ErrNotSupported.Err()
	}

	return nil, bridge.ErrDeviceNotFound.Err()
}

// StreamBridgeUpdates monitors changes for all changes which occur on the bridge.
func (s *Service) StreamBridgeUpdates(req *bridge.StreamBridgeUpdatesRequest, stream bridge.BridgeService_StreamBridgeUpdatesServer) error {
	peer, isOk := peer.FromContext(stream.Context())

	addr := "unknown"
	if isOk {
		addr = peer.Addr.String()
	}

	logger := s.logger.With(zap.String("peer_addr", addr))

	logger.Debug("bridge update stream initiated")

	sink := s.updates.NewSink()
	defer sink.Close()

	// Send the device info to start.

	listDevicesResp, err := s.ListDevices(stream.Context(), &bridge.ListDevicesRequest{})
	if err != nil {
		s.logger.Error("unable to retrieve devices",
			zap.Error(err),
		)
		// This is already a gRPC error so just return it here
		return err
	}

	for _, device := range listDevicesResp.Devices {
		update := &bridge.Update{
			Action: bridge.Update_ADDED,
			Update: &bridge.Update_DeviceUpdate{
				DeviceUpdate: &bridge.DeviceUpdate{
					Device:   device,
					DeviceId: device.Id,
					BridgeId: s.brInfo.Id,
				},
			},
		}
		logger.Debug("sending seed info",
			zap.String("device_info", update.String()),
		)

		if err := stream.Send(update); err != nil {
			logger.Error("unable to send update",
				zap.Error(err),
			)
			return err
		}
	}

	// Now we wait for updates
	for {
		update, ok := <-sink.Messages()
		if !ok {
			logger.Debug("stream closed")
			// Channel has been closed; so we'll close the connection as well
			return nil
		}

		bridgeUpdate, ok := update.(*bridge.Update)

		if !ok {
			panic("update cast incorrect")
		}

		logger.Debug("sending update",
			zap.String("info", bridgeUpdate.String()),
		)

		if err := stream.Send(bridgeUpdate); err != nil {
			return err
		}
	}
}

func lightToDevice(l *deconz.Light) *bridge.Device {
	ret := &bridge.Device{
		Id:           l.UniqueID,
		Type:         bridge.DeviceType_LIGHT,
		IsActive:     true,
		ModelId:      l.ModelID,
		Manufacturer: l.Manufacturer,
		Range: &bridge.Device_Range{
			Minimum: 0,
			Maximum: 255,
		},
		ColorTemperature: &bridge.Device_ColorTemperature{
			Minimum: int32(l.CTMin),
			Maximum: int32(l.CTMax),
		},
		Config: &bridge.DeviceConfig{
			Name: l.Name,
		},
		State: &bridge.DeviceState{
			IsReachable: l.State.Reachable,
			Version: &bridge.Version{
				Sw: l.SoftwareVersion,
			},
			Binary: &bridge.DeviceState_Binary{
				IsOn: l.State.On,
			},
			Range: &bridge.DeviceState_Range{
				Value: int32(l.State.Brightness),
			},
		},
	}

	if l.State.ColorMode == "xy" {
		// Go from Deconz/Hue xy to CIE XYZ
		x := l.State.XY[0]
		y := l.State.XY[1]
		z := float64(1.0) - x - y
		Y := float64(l.State.Brightness)
		X := (Y / y) * x
		Z := (Y / y) * z

		c := colorful.Xyz(X, Y, Z)
		// h is in range of 0-360
		// s and v are in range of 0-1 and need to be converted to match the protocol
		h, s, v := c.Hsv()

		ret.State.ColorHsb = &bridge.DeviceState_ColorHSB{
			Hue:        int32(h),
			Saturation: int32(s * float64(100.0)),
			Brightness: int32(v * float64(100)),
		}
	} else if l.State.ColorMode == "hs" {
		// We need to convert the range returned by the bulb into the ranges supported by the protocol
		ret.State.ColorHsb = &bridge.DeviceState_ColorHSB{
			Hue:        int32((float64(l.State.Hue) * float64(360.0)) / float64(65535)),
			Saturation: int32((float64(l.State.Saturation) * float64(100.0)) / float64(255.0)),
			Brightness: int32((float64(l.State.Brightness) * float64(100.0)) / float64(255.0)),
		}
	} else if l.State.ColorMode == "ct" {
		ret.State.ColorTemperature = int32(l.State.CT)
	}

	return ret
}
func sensorToDevice(s *deconz.Sensor) *bridge.Device {
	ret := &bridge.Device{
		Id:           s.UniqueID,
		Type:         bridge.DeviceType_SENSOR,
		IsActive:     true,
		ModelId:      s.ModelID,
		Manufacturer: s.ManufacturerName,
		Config: &bridge.DeviceConfig{
			Name: s.Name,
		},
		State: &bridge.DeviceState{
			IsReachable: s.Config.Reachable,
			Version: &bridge.Version{
				Sw: s.SoftwareVersion,
			},
			Binary: &bridge.DeviceState_Binary{
				IsOn: s.Config.On,
			},
		},
	}

	// TODO: expose a bunch of sensor info
	return ret
}
