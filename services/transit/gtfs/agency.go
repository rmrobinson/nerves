package gtfs

// Agency represents the transit agency supplying service.
type Agency struct {
	ID            string `csv:"agency_id"`
	Name          string `csv:"agency_name"`
	URL           string `csv:"agency_url"`
	TZ            string `csv:"agency_timezone"`
	Language      string `csv:"agency_language"`
	ContactNumber string `csv:"agency_phone"`
	FareURL       string `csv:"agency_fare_url"`
	ContactEmail  string `csv:"agency_email"`
}
