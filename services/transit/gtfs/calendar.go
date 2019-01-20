package gtfs

// Calendar is a set of days that the specified service is available.
type Calendar struct {
	ServiceID string  `csv:"service_id"`
	Monday    CSVBool `csv:"monday"`
	Tuesday   CSVBool `csv:"tuesday"`
	Wednesday CSVBool `csv:"wednesday"`
	Thursday  CSVBool `csv:"thursday"`
	Friday    CSVBool `csv:"friday"`
	Saturday  CSVBool `csv:"saturday"`
	Sunday    CSVBool `csv:"sunday"`
	StartDate CSVDate `csv:"start_date"`
	EndDate   CSVDate `csv:"end_date"`
}
