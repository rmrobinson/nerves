package gtfs

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	gtfsDateFormat = "20060102"
)

var (
	// ErrInvalidBoolField is returned if a boolean field has invalid data
	ErrInvalidBoolField = errors.New("invalid boolean field supplied")
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
	return d.Format(gtfsDateFormat), nil
}

// UnmarshalCSV takes the string representation from a CSV file and attempts to convert it to a float64.
func (d *CSVDate) UnmarshalCSV(csv string) (err error) {
	d.Time, err = time.Parse(gtfsDateFormat, csv)
	return err
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
