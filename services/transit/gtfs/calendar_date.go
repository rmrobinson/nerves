package gtfs

// CalendarDate represents a service override on the specified date.
type CalendarDate struct {
	ServiceID     string  `csv:"service_id"`
	Date          CSVDate `csv:"date"`
	ExceptionType string  `csv:"exception_type"`
}
