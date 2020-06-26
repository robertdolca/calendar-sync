package sync

import (
	"google.golang.org/api/calendar/v3"
)

func mapEvent(event *calendar.Event, copyDescription bool, copyLocation bool) *calendar.Event {
	if event == nil {
		return nil
	}
	result := &calendar.Event{
		ColorId:            event.ColorId,
		Created:            event.Created,
		End:                mapEventDateTime(event.End),
		EndTimeUnspecified: event.EndTimeUnspecified,
		Kind:               event.Kind,
		OriginalStartTime:  mapEventDateTime(event.OriginalStartTime),
		Recurrence:         event.Recurrence,
		Start:              mapEventDateTime(event.Start),
		Status:             event.Status,
		Summary:            event.Summary,
		Transparency:       event.Transparency,
	}
	if copyDescription {
		result.Description = event.Description
	}
	if copyLocation {
		result.Location = event.Location
	}
	return result
}

func mapEventDateTime(dt *calendar.EventDateTime) *calendar.EventDateTime {
	if dt == nil {
		return nil
	}
	return &calendar.EventDateTime{
		Date:     dt.Date,
		DateTime: dt.DateTime,
		TimeZone: dt.TimeZone,
	}
}
