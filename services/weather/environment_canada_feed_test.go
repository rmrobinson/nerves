package weather

import (
	"testing"
	"time"

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
		t.Run(tt.name, func(t *testing.T) {
			res := currentConditionsToCondition(tt.text)
			assert.Equal(t, tt.result, res)
		})
	}
}

type forecastConditionToConditionTest struct {
	name string
	text string
	result *WeatherCondition
}

var forecastConditionToConditionTests = []forecastConditionToConditionTest{
	{
		"basic weather report",
		`Mainly cloudy. Wind becoming west 20 km/h late this afternoon. High plus 4. UV index 1 or low. Forecast issued 11:00 AM EST Saturday 05 January 2019`,
		&WeatherCondition{
			Summary: "Mainly cloudy",
			//SummaryIcon: WeatherIcon_CLOUDY,
			Temperature: 4,
			WindSpeed: 20,
			UvIndex: 1,
		},
	},
	{
		"multi-value temperature",
		`Clearing in the morning. Wind northwest 20 km/h. Temperature falling to minus 8 in the afternoon. Wind chill minus 7 in the morning and minus 14 in the afternoon. UV index 1 or low. Forecast issued 11:00 AM EST Saturday 05 January 2019`,
		&WeatherCondition{
			Summary: "Clearing in the morning",
			//SummaryIcon: WeatherIcon_CLOUDY,
			Temperature: -8,
			WindChill: -14,
			WindSpeed: 20,
			UvIndex: 1,
		},
	},
	{
		"trivial case",
		`Periods of snow. High plus 2. Forecast issued 11:00 AM EST Saturday 05 January 2019`,
		&WeatherCondition{
			Summary: "Periods of snow",
			//SummaryIcon: WeatherIcon_CLOUDY,
			Temperature: 2,
		},
	},
}

func TestForecastConditionToCondition(t *testing.T) {
	for _, tt := range forecastConditionToConditionTests {
		t.Run(tt.name, func(t *testing.T) {
			res := forecastConditionToCondition(tt.text)
			assert.Equal(t, tt.result, res)
		})
	}
}

type futureDateFromFeedDate struct {
	name string
	startDate time.Time
	futureDate string
	result time.Time
	err error
}

var futureDateFromFeedDates = []futureDateFromFeedDate{
	{
		"basic case",
		time.Date(2019, time.January, 5, 10, 0, 0, 0, time.UTC),
		"Monday",
		time.Date(2019, time.January, 7, 10, 0, 0, 0, time.UTC),
		nil,
	},
	{
		"invalid future date",
		time.Date(2019, time.January, 5, 10, 0, 0, 0, time.UTC),
		"Taco",
		time.Date(2019, time.January, 7, 10, 0, 0, 0, time.UTC),
		ErrInvalidDate,
	},
}

func TestFutureDateFromFeedDate(t *testing.T) {
	for _, tt := range futureDateFromFeedDates {
		t.Run(tt.name, func(t *testing.T) {
			res, err := futureDateFromFeedDate(tt.startDate, tt.futureDate)
			assert.Equal(t, tt.err, err)
			if err == nil {
				assert.Equal(t, tt.result, res)
			}
		})
	}
}

type floatFromFeedTextTest struct {
	name string
	text string
	result float32
	err error
}

var floatFromFeedTextTests = []floatFromFeedTextTest{
	{
		"basic case",
		"this is a text with 1 value",
		1,
		nil,
	},
	{
		"multiple values, get last one",
		"this is a text with 2 values; and 1 other value",
		1,
		nil,
	},
	{
		"negative value",
		"this is a text with minus 2 values",
		-2,
		nil,
	},
}

func TestFloatFromFeedText(t *testing.T) {
	for _, tt := range floatFromFeedTextTests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := floatFromFeedText(tt.text)
			assert.Equal(t, tt.err, err)
			if err == nil {
				assert.Equal(t, tt.result, res)
			}
		})
	}
}
