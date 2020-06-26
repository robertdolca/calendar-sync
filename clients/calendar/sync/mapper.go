package sync

import (
	"google.golang.org/api/calendar/v3"
)

func mapEvent(event *calendar.Event) *calendar.Event {
	if event == nil {
		return nil
	}
	return &calendar.Event{
		ColorId:            event.ColorId,
		Created:            event.Created,
		Description:        event.Description,
		End:                mapEventDateTime(event.End),
		EndTimeUnspecified: event.EndTimeUnspecified,
		Kind:               event.Kind,
		Location:           event.Location,
		OriginalStartTime:  mapEventDateTime(event.OriginalStartTime),
		Recurrence:         event.Recurrence,
		Start:              mapEventDateTime(event.Start),
		Status:             event.Status,
		Summary:            event.Summary,
		Transparency:       event.Transparency,
	}
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
