package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/calendar/v3"

	"github.com/joelanford/mcp/google-workspace/types"
)

// CalendarTools provides Google Calendar API tools.
type CalendarTools struct {
	calendarService *calendar.Service
}

// NewCalendarTools creates a new CalendarTools instance from the provided clients.
func NewCalendarTools(clients *types.CalendarClients) *CalendarTools {
	return &CalendarTools{
		calendarService: clients.Calendar,
	}
}

// ListCalendarsTool returns the tool definition for listing calendars.
func (c *CalendarTools) ListCalendarsTool() mcp.Tool {
	return mcp.NewTool("calendar_list",
		mcp.WithDescription(`Lists all calendars accessible to the authenticated user.

Returns a JSON object with an array of calendars, each containing:
  - id: The calendar identifier
  - summary: The calendar name/title
  - primary: Whether this is the user's primary calendar
  - accessRole: The user's access role (owner, writer, reader, freeBusyReader)`),
	)
}

// CalendarListResponse contains the list of calendars.
type CalendarListResponse struct {
	Calendars []CalendarInfo `json:"calendars"`
}

// CalendarInfo represents a single calendar's information.
type CalendarInfo struct {
	ID         string `json:"id"`
	Summary    string `json:"summary"`
	Primary    bool   `json:"primary,omitempty"`
	AccessRole string `json:"accessRole"`
}

// ListCalendarsHandler handles calendar_list tool calls.
func (c *CalendarTools) ListCalendarsHandler(ctx context.Context, request mcp.CallToolRequest, args types.CalendarListArgs) (*mcp.CallToolResult, error) {
	calendarList, err := c.calendarService.CalendarList.List().Context(ctx).Do()
	if err != nil {
		return mcp.NewToolResultError("failed to list calendars: " + err.Error()), nil
	}

	response := CalendarListResponse{
		Calendars: make([]CalendarInfo, 0, len(calendarList.Items)),
	}

	for _, cal := range calendarList.Items {
		response.Calendars = append(response.Calendars, CalendarInfo{
			ID:         cal.Id,
			Summary:    cal.Summary,
			Primary:    cal.Primary,
			AccessRole: cal.AccessRole,
		})
	}

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// GetEventsTool returns the tool definition for getting calendar events.
func (c *CalendarTools) GetEventsTool() mcp.Tool {
	return mcp.NewTool("calendar_get_events",
		mcp.WithDescription(`Retrieves events from a Google Calendar.

Supports three modes:
1. Single event lookup by event_id
2. Time range query with time_min and/or time_max (RFC3339 format)
3. Free text search with query parameter

If no time range is specified, defaults to events from now onwards.

Returns a JSON object with an array of events, each containing:
  - id: Event identifier
  - summary: Event title
  - start: Start time (dateTime or date for all-day events)
  - end: End time
  - location: Event location (if set)
  - description: Event description (if set)
  - htmlLink: Link to view the event in Google Calendar
  - attendees: List of attendees (if any)`),
		mcp.WithString("calendar_id",
			mcp.Description("Calendar identifier (defaults to 'primary')"),
		),
		mcp.WithString("event_id",
			mcp.Description("Specific event ID to retrieve (optional)"),
		),
		mcp.WithString("time_min",
			mcp.Description("Start of time range in RFC3339 format, e.g. '2024-01-15T00:00:00Z' (optional)"),
		),
		mcp.WithString("time_max",
			mcp.Description("End of time range in RFC3339 format (optional)"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of events to return (default 25, max 2500)"),
		),
		mcp.WithString("query",
			mcp.Description("Free text search terms to find events (optional)"),
		),
		mcp.WithBoolean("include_attachments",
			mcp.Description("Include file attachment information in the response"),
		),
	)
}

// EventsResponse contains the list of events.
type EventsResponse struct {
	Events []EventInfo `json:"events"`
}

// SingleEventResponse contains a single event.
type SingleEventResponse struct {
	Event EventInfo `json:"event"`
}

// EventInfo represents a single event's information.
type EventInfo struct {
	ID          string           `json:"id"`
	Summary     string           `json:"summary"`
	Start       string           `json:"start"`
	End         string           `json:"end"`
	Location    string           `json:"location,omitempty"`
	Description string           `json:"description,omitempty"`
	HTMLLink    string           `json:"htmlLink"`
	Attendees   []AttendeeInfo   `json:"attendees,omitempty"`
	Attachments []AttachmentInfo `json:"attachments,omitempty"`
}

// AttendeeInfo represents an event attendee.
type AttendeeInfo struct {
	Email          string `json:"email"`
	DisplayName    string `json:"displayName,omitempty"`
	ResponseStatus string `json:"responseStatus,omitempty"`
	Organizer      bool   `json:"organizer,omitempty"`
}

// AttachmentInfo represents an event attachment.
type AttachmentInfo struct {
	FileID   string `json:"fileId,omitempty"`
	FileURL  string `json:"fileUrl"`
	Title    string `json:"title"`
	MimeType string `json:"mimeType,omitempty"`
}

// GetEventsHandler handles calendar_get_events tool calls.
func (c *CalendarTools) GetEventsHandler(ctx context.Context, request mcp.CallToolRequest, args types.CalendarGetEventsArgs) (*mcp.CallToolResult, error) {
	calendarID := args.CalendarID
	if calendarID == "" {
		calendarID = "primary"
	}

	// Single event lookup
	if args.EventID != "" {
		event, err := c.calendarService.Events.Get(calendarID, args.EventID).Context(ctx).Do()
		if err != nil {
			return mcp.NewToolResultError("failed to get event: " + err.Error()), nil
		}

		response := SingleEventResponse{
			Event: eventToInfo(event, args.IncludeAttachments),
		}

		data, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}

	// List events with optional filters
	listCall := c.calendarService.Events.List(calendarID).
		Context(ctx).
		SingleEvents(true).
		OrderBy("startTime")

	// Set time range
	if args.TimeMin != "" {
		listCall = listCall.TimeMin(args.TimeMin)
	} else {
		// Default to now
		listCall = listCall.TimeMin(time.Now().Format(time.RFC3339))
	}

	if args.TimeMax != "" {
		listCall = listCall.TimeMax(args.TimeMax)
	}

	// Set max results
	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = 25
	}
	if maxResults > 2500 {
		maxResults = 2500
	}
	listCall = listCall.MaxResults(int64(maxResults))

	// Set search query
	if args.Query != "" {
		listCall = listCall.Q(args.Query)
	}

	events, err := listCall.Do()
	if err != nil {
		return mcp.NewToolResultError("failed to list events: " + err.Error()), nil
	}

	response := EventsResponse{
		Events: make([]EventInfo, 0, len(events.Items)),
	}

	for _, event := range events.Items {
		response.Events = append(response.Events, eventToInfo(event, args.IncludeAttachments))
	}

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// eventToInfo converts a calendar event to EventInfo.
func eventToInfo(event *calendar.Event, includeAttachments bool) EventInfo {
	info := EventInfo{
		ID:          event.Id,
		Summary:     event.Summary,
		Location:    event.Location,
		Description: event.Description,
		HTMLLink:    event.HtmlLink,
	}

	// Handle start time (can be dateTime or date for all-day events)
	if event.Start != nil {
		if event.Start.DateTime != "" {
			info.Start = event.Start.DateTime
		} else {
			info.Start = event.Start.Date
		}
	}

	// Handle end time
	if event.End != nil {
		if event.End.DateTime != "" {
			info.End = event.End.DateTime
		} else {
			info.End = event.End.Date
		}
	}

	// Convert attendees
	if len(event.Attendees) > 0 {
		info.Attendees = make([]AttendeeInfo, 0, len(event.Attendees))
		for _, attendee := range event.Attendees {
			info.Attendees = append(info.Attendees, AttendeeInfo{
				Email:          attendee.Email,
				DisplayName:    attendee.DisplayName,
				ResponseStatus: attendee.ResponseStatus,
				Organizer:      attendee.Organizer,
			})
		}
	}

	// Convert attachments if requested
	if includeAttachments && len(event.Attachments) > 0 {
		info.Attachments = make([]AttachmentInfo, 0, len(event.Attachments))
		for _, attachment := range event.Attachments {
			info.Attachments = append(info.Attachments, AttachmentInfo{
				FileID:   attachment.FileId,
				FileURL:  attachment.FileUrl,
				Title:    attachment.Title,
				MimeType: attachment.MimeType,
			})
		}
	}

	return info
}
