package widget

// DeviceInfo is a mock struct containing basic device information.
type DeviceInfo struct {
	Name        string
	Description string

	IsOn  bool
	Level uint8

	Red   uint8
	Green uint8
	Blue  uint8
}
