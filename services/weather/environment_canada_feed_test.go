package weather

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type currentConditionToConditionTest struct {
	name string
	text string
	result *WeatherCondition
}

var currentConditionToConditionTests = []currentConditionToConditionTest{
	{
		"basic weather report",
		`<b>Observed at:</b> Region of Waterloo Int'l Airport 8:00 PM EST Thursday 03 January 2019 <br/>
<b>Condition:</b> Cloudy <br/>
<b>Temperature:</b> -1.3&deg;C <br/>
<b>Pressure:</b> 101.4 kPa <br/>
<b>Visibility:</b> 16.1 km<br/>
<b>Humidity:</b> 86 %<br/>
<b>Wind Chill:</b> -7 <br/>
<b>Dewpoint:</b> -3.4&deg;C <br/>
<b>Wind:</b> SW 21 km/h<br/>
<b>Air Quality Health Index:</b> 2 <br/>`,
		&WeatherCondition{
			Summary: "Cloudy",
			//SummaryIcon: WeatherIcon_CLOUDY,
			Temperature: -1.3,
			Pressure: 101.4,
			Visibility: 16,
			Humidity: 86,
			WindChill: -7,
			DewPoint: -3.4,
			WindSpeed: 21,
		},
	},
}

func TestCurrentConditionToCondition(t *testing.T) {
	for _, tt := range currentConditionToConditionTests {
		t.Run(tt.text, func(t *testing.T) {
			res := currentConditionsToCondition(tt.text)
			assert.Equal(t, res, tt.result)
		})
	}
}