package transit

import (
	"errors"
	"strconv"
	"strings"
)

// Bool is a CSV marshalable boolean value
type Bool struct {
	bool
}

// MarshalCSV marshals the value into a string format
func (b *Bool) MarshalCSV() (string, error) {
	if b.bool {
		return "1", nil
	}
	return "0", nil
}

// UnmarshalCSV takes the string representation from a CSV file and attempts to convert it to a int32.
func (b *Bool) UnmarshalCSV(csv string) error {
	csv = strings.TrimSpace(csv)
	if len(csv) < 1 {
		b.bool = false
		return nil
	}

	val, err := strconv.ParseInt(strings.TrimSpace(csv), 10, 32)
	if err != nil {
		return err
	}

	if val == 1 {
		b.bool = true
	} else if val == 0 {
		b.bool = false
	}
	return errors.New("invalid bool value")
}

// Calendar is a set of days that the specified service is available.
type Calendar struct {
	ServiceID string `csv:"service_id"`
	Monday    Bool   `csv:"monday"`
	Tuesday   Bool   `csv:"tuesday"`
	Wednesday Bool   `csv:"wednesday"`
	Thursday  Bool   `csv:"thursday"`
	Friday    Bool   `csv:"friday"`
	Saturday  Bool   `csv:"saturday"`
	Sunday    Bool   `csv:"sunday"`
	StartDate string `csv:"start_date"`
	EndDate   string `csv:"end_date"`
}
