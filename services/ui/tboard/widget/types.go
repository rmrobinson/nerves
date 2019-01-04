package widget

type DeviceInfo struct {
	Name        string
	Description string

	IsOn  bool
	Level uint8

	Red   uint8
	Green uint8
	Blue  uint8
}
