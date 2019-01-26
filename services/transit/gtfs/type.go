package gtfs

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// DateFormat is the GTFS-described date format
	DateFormat = "20060102"
)

var (
	// ErrInvalidBoolField is returned if a boolean field has invalid data
	ErrInvalidBoolField = errors.New("invalid boolean field supplied")
	// ErrInvalidTimeField is returned if a time field has invalid data
	ErrInvalidTimeField = errors.New("invalid time field supplied")
)

// CSVBool is a CSV marshalable boolean value
type CSVBool bool

// MarshalCSV marshals the value into a string format
func (b *CSVBool) MarshalCSV() (string, error) {
	if *b {
		return "1", nil
	}
	return "0", nil
}

// UnmarshalCSV takes the string representation from a CSV file and attempts to convert it to a int32.
func (b *CSVBool) UnmarshalCSV(csv string) error {
	csv = strings.TrimSpace(csv)
	if len(csv) < 1 {
		*b = false
		return nil
	}

	val, err := strconv.ParseInt(csv, 10, 32)
	if err != nil {
		return err
	}

	if val == 1 {
		*b = true
		return nil
	} else if val == 0 {
		*b = false
		return nil
	}
	return ErrInvalidBoolField
}

// CSVDate is a GTFS date parsed from CSV
type CSVDate struct {
	time.Time
}

// MarshalCSV marshals the value into a string format
func (d *CSVDate) MarshalCSV() (string, error) {
	return d.Format(DateFormat), nil
}

// UnmarshalCSV takes the string representation from a CSV file and attempts to convert it to a time.Time.
func (d *CSVDate) UnmarshalCSV(csv string) (err error) {
	d.Time, err = time.Parse(DateFormat, csv)
	return err
}

// CSVTime is a GTFS time parsed from CSV
type CSVTime struct {
	Hour   int
	Minute int
	Second int
}

// NewCSVTime creates a new CSVTime from the supplied time.
func NewCSVTime(t time.Time) *CSVTime {
	return &CSVTime{
		t.Hour(),
		t.Minute(),
		t.Second(),
	}
}

// MarshalCSV marshals the value into a string format
func (t *CSVTime) MarshalCSV() (string, error) {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour, t.Minute, t.Second), nil
}

// UnmarshalCSV takes the string representation from a CSV file and attempts to convert it to a time.Time.
func (t *CSVTime) UnmarshalCSV(csv string) (err error) {
	at := strings.Split(csv, ":")

	if len(at) != 3 {
		return ErrInvalidTimeField
	}

	val, err := strconv.ParseInt(at[0], 10, 32)
	if err != nil {
		return err
	}
	t.Hour = int(val)
	val, err = strconv.ParseInt(at[1], 10, 32)
	if err != nil {
		return err
	}
	t.Minute = int(val)
	val, err = strconv.ParseInt(at[2], 10, 32)
	if err != nil {
		return err
	}
	t.Second = int(val)

	return nil
}

// String returns a printable version of this type
func (t *CSVTime) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour, t.Minute, t.Second)
}

// CSVFloat is a CSV marshalable float64 value
type CSVFloat float64

// MarshalCSV marshals the value into a string format
func (f CSVFloat) MarshalCSV() (string, error) {
	return fmt.Sprintf("%f", f), nil
}

// UnmarshalCSV takes the string representation from a CSV file and attempts to convert it to a float64.
func (f *CSVFloat) UnmarshalCSV(csv string) error {
	csv = strings.TrimSpace(csv)
	if len(csv) < 1 {
		*f = 0
		return nil
	}

	val, err := strconv.ParseFloat(csv, 64)
	if err != nil {
		return err
	}

	*f = CSVFloat(val)
	return nil
}

// CSVInt is a CSV marshalable int32 value
type CSVInt int

// MarshalCSV marshals the value into a string format
func (i *CSVInt) MarshalCSV() (string, error) {
	return fmt.Sprintf("%d", *i), nil
}

// UnmarshalCSV takes the string representation from a CSV file and attempts to convert it to a int32.
func (i *CSVInt) UnmarshalCSV(csv string) error {
	csv = strings.TrimSpace(csv)
	if len(csv) < 1 {
		*i = 0
		return nil
	}

	val, err := strconv.ParseInt(csv, 10, 32)
	if err != nil {
		return err
	}

	*i = CSVInt(val)
	return nil
}
