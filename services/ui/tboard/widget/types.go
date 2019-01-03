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

type WeatherConditionInfo struct {
	Description string

	TemperatureCelsius float32
	WindChillCelsius   float32

	HumidityPercentage uint8
	PressureKPa        float32
	WindSpeedKmPerHr   uint32
	VisibilityKm       uint32
	DewPointCelsius    float32

	UVIndex uint8
}
