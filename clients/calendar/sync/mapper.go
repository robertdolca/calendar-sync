package sync

import (
	"google.golang.org/api/calendar/v3"
)

func mapEvent(event *calendar.Event, mappingOptions MappingOptions) *calendar.Event {
	if event == nil {
		return nil
	}
	result := &calendar.Event{
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
	if mappingOptions.CopyColor {
		result.ColorId = event.ColorId
	}
	if mappingOptions.CopyDescription {
		result.Description = event.Description
	}
	if mappingOptions.CopyLocation {
		result.Location = event.Location
	}
	result.Visibility = mappingOptions.Visibility
	if mappingOptions.TitleOverride != "" {
		result.Summary = mappingOptions.TitleOverride
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
