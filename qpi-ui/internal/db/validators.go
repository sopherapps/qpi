package db

import (
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ValidateTimeSlot asserts start_time is before end_time and prevents globally overlapping bookings.
func ValidateTimeSlot(app core.App, record *core.Record) error {
	startDt := record.GetDateTime("start_time")
	endDt := record.GetDateTime("end_time")
	start := startDt.Time()
	end := endDt.Time()

	if start.After(end) || start.Equal(end) {
		return errors.New("start_time must be strictly before end_time")
	}

	// Query DB to check if there are globally overlapping slots
	query := app.DB().
		Select("count(*)").
		From("time_slots").
		Where(dbx.NewExp("start_time < {:end_time} AND end_time > {:start_time}", dbx.Params{
			"end_time":   endDt.String(),
			"start_time": startDt.String(),
		}))

	if record.Id != "" {
		query = query.AndWhere(dbx.NewExp("id != {:id}", dbx.Params{"id": record.Id}))
	}

	var count int
	if err := query.Row(&count); err != nil {
		return err
	}

	if count > 0 {
		return errors.New("The requested booking slot overlaps with an existing reservation.")
	}

	return nil
}
