package transit

import (
	"time"

	"github.com/rmrobinson/nerves/services/transit/gtfs"
)

type agencyDetails struct {
	*gtfs.Agency

	loc *time.Location
}

func newAgencyDetails(a *gtfs.Agency) *agencyDetails {
	loc, err := time.LoadLocation(a.TZ)
	if err != nil {
		loc = time.UTC
	}
	return &agencyDetails{
		Agency: a,
		loc: loc,
	}
}
