package widget

import (
	"testing"
)

type textToConditionSymbolTest struct {
	text string
	result string
}

var textToConditionSymbolTests = []textToConditionSymbolTest{
	{
		"Mainly cloudy with 30 percent chance of flurries",
		"🌨",
	},
	{
		"Sunny",
		"☼",
	},
	{
		"A mix of sun and cloud",
		"🌤",
	},
	{
		"Light Rain",
		"🌧",
	},
	{
		"Hurricane",
		"Hurricane",
	},
}

func TestTextToConditionSymbol(t *testing.T) {
	for _, tt := range textToConditionSymbolTests {
		t.Run(tt.text, func(t *testing.T) {
			if res := textToConditionSymbol(tt.text); res != tt.result {
				t.Errorf("expected %s, got %s", tt.result, res)
			}
		})
	}
}
