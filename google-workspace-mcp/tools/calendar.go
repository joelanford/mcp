package tools

import (
	"context"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/calendar/v3"

	"github.com/joelanford/mcp/google-workspace-mcp/types"
)

// CalendarListRequest contains arguments for listing calendars.
type CalendarListRequest struct{}

// CalendarGetEventsRequest contains arguments for getting calendar events.
type CalendarGetEventsRequest struct {
	CalendarID         string `json:"calendar_id"`         // Calendar ID, defaults to "primary"
	EventID            string `json:"event_id"`            // Specific event ID (optional)
	TimeMin            string `json:"time_min"`            // Start of time range in RFC3339 format (optional)
	TimeMax            string `json:"time_max"`            // End of time range in RFC3339 format (optional)
	MaxResults         int    `json:"max_results"`         // Maximum number of events to return (default 25)
	Query              string `json:"query"`               // Free text search query (optional)
	IncludeAttachments bool   `json:"include_attachments"` // Include file attachments in response
	PageToken          string `json:"page_token"`          // Continue from previous page
	OrderBy            string `json:"order_by"`            // Sort order: startTime (default) or updated
}

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
func (c *CalendarTools) ListCalendarsHandler(ctx context.Context, request mcp.CallToolRequest, args CalendarListRequest) (*mcp.CallToolResult, error) {
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

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
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
  - attendees: List of attendees (if any)
  - next_page_token: Token for fetching the next page (if more results exist)`),
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
		mcp.WithString("page_token",
			mcp.Description("Page token from previous response to continue pagination"),
		),
		mcp.WithString("order_by",
			mcp.Description("Sort order: startTime (default) or updated"),
		),
	)
}

// CalendarGetEventsResponse contains the list of events.
type CalendarGetEventsResponse struct {
	Events        []CalendarEventInfo `json:"events"`
	NextPageToken string      `json:"next_page_token,omitempty"`
}

// CalendarGetEventResponse contains a single event.
type CalendarGetEventResponse struct {
	Event CalendarEventInfo `json:"event"`
}

// CalendarEventInfo represents a single event's information.
type CalendarEventInfo struct {
	ID          string           `json:"id"`
	Summary     string           `json:"summary"`
	Start       string           `json:"start"`
	End         string           `json:"end"`
	Location    string           `json:"location,omitempty"`
	Description string           `json:"description,omitempty"`
	HTMLLink    string           `json:"htmlLink"`
	Attendees   []CalendarAttendeeInfo   `json:"attendees,omitempty"`
	Attachments []CalendarAttachmentInfo `json:"attachments,omitempty"`
}

// CalendarAttendeeInfo represents an event attendee.
type CalendarAttendeeInfo struct {
	Email          string `json:"email"`
	DisplayName    string `json:"displayName,omitempty"`
	ResponseStatus string `json:"responseStatus,omitempty"`
	Organizer      bool   `json:"organizer,omitempty"`
}

// CalendarAttachmentInfo represents an event attachment.
type CalendarAttachmentInfo struct {
	FileID   string `json:"fileId,omitempty"`
	FileURL  string `json:"fileUrl"`
	Title    string `json:"title"`
	MimeType string `json:"mimeType,omitempty"`
}

// GetEventsHandler handles calendar_get_events tool calls.
func (c *CalendarTools) GetEventsHandler(ctx context.Context, request mcp.CallToolRequest, args CalendarGetEventsRequest) (*mcp.CallToolResult, error) {
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

		response := CalendarGetEventResponse{
			Event: eventToInfo(event, args.IncludeAttachments),
		}

		data, err := types.MarshalResponse(response)
		if err != nil {
			return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
		}
		return mcp.NewToolResultText(data), nil
	}

	// List events with optional filters
	listCall := c.calendarService.Events.List(calendarID).
		Context(ctx).
		SingleEvents(true)

	// Set sort order
	if args.OrderBy != "" {
		listCall = listCall.OrderBy(args.OrderBy)
	} else {
		listCall = listCall.OrderBy("startTime")
	}

	// Apply pagination
	if args.PageToken != "" {
		listCall = listCall.PageToken(args.PageToken)
	}

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

	response := CalendarGetEventsResponse{
		Events:        make([]CalendarEventInfo, 0, len(events.Items)),
		NextPageToken: events.NextPageToken,
	}

	for _, event := range events.Items {
		response.Events = append(response.Events, eventToInfo(event, args.IncludeAttachments))
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// eventToInfo converts a calendar event to CalendarEventInfo.
func eventToInfo(event *calendar.Event, includeAttachments bool) CalendarEventInfo {
	info := CalendarEventInfo{
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
		info.Attendees = make([]CalendarAttendeeInfo, 0, len(event.Attendees))
		for _, attendee := range event.Attendees {
			info.Attendees = append(info.Attendees, CalendarAttendeeInfo{
				Email:          attendee.Email,
				DisplayName:    attendee.DisplayName,
				ResponseStatus: attendee.ResponseStatus,
				Organizer:      attendee.Organizer,
			})
		}
	}

	// Convert attachments if requested
	if includeAttachments && len(event.Attachments) > 0 {
		info.Attachments = make([]CalendarAttachmentInfo, 0, len(event.Attachments))
		for _, attachment := range event.Attachments {
			info.Attachments = append(info.Attachments, CalendarAttachmentInfo{
				FileID:   attachment.FileId,
				FileURL:  attachment.FileUrl,
				Title:    attachment.Title,
				MimeType: attachment.MimeType,
			})
		}
	}

	return info
}

// MarshalCompact returns a compact text representation of the calendar list.
func (c CalendarListResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString("Calendars:")

	for _, cal := range c.Calendars {
		if cal.Primary {
			sb.WriteString("\n* ")
		} else {
			sb.WriteString("\n  ")
		}
		sb.WriteString(cal.ID)
		if cal.Summary != "" && cal.Summary != cal.ID {
			sb.WriteString(" (")
			sb.WriteString(cal.Summary)
			sb.WriteString(")")
		}
		if cal.Primary {
			sb.WriteString(" [primary]")
		}
		sb.WriteString(" ")
		sb.WriteString(cal.AccessRole)
	}
	return sb.String()
}

// MarshalCompact returns a compact text representation of the events list.
func (e CalendarGetEventsResponse) MarshalCompact() string {
	var sb strings.Builder

	for i, event := range e.Events {
		if i > 0 {
			sb.WriteString("\n")
		}
		writeEventCompact(&sb, event)
	}

	if e.NextPageToken != "" {
		sb.WriteString("\n\nNext Page Token: ")
		sb.WriteString(e.NextPageToken)
	}

	return sb.String()
}

// MarshalCompact returns a compact text representation of a single event.
func (s CalendarGetEventResponse) MarshalCompact() string {
	var sb strings.Builder
	writeEventCompact(&sb, s.Event)
	return sb.String()
}

// writeEventCompact writes a single event in compact format.
func writeEventCompact(sb *strings.Builder, event CalendarEventInfo) {
	// Parse and format date/time: "2025-01-19 09:00-09:30 | Title | Location"
	startDate, startTime := parseDateTime(event.Start)
	_, endTime := parseDateTime(event.End)

	sb.WriteString(startDate)
	if startTime != "" {
		sb.WriteString(" ")
		sb.WriteString(startTime)
		if endTime != "" {
			sb.WriteString("-")
			sb.WriteString(endTime)
		}
	}
	sb.WriteString(" | ")
	sb.WriteString(event.Summary)
	if event.Location != "" {
		sb.WriteString(" | ")
		sb.WriteString(event.Location)
	}

	// Description
	if event.Description != "" {
		sb.WriteString("\n  Description: ")
		// Truncate long descriptions and handle newlines
		desc := strings.ReplaceAll(event.Description, "\n", " ")
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		sb.WriteString(desc)
	}

	// Link
	if event.HTMLLink != "" {
		sb.WriteString("\n  Link: ")
		sb.WriteString(event.HTMLLink)
	}

	// Attendees
	if len(event.Attendees) > 0 {
		sb.WriteString("\n  Attendees: ")
		for i, att := range event.Attendees {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(att.Email)
			if att.ResponseStatus != "" {
				sb.WriteString(" (")
				sb.WriteString(att.ResponseStatus)
				sb.WriteString(")")
			}
		}
	}

	// Attachments
	if len(event.Attachments) > 0 {
		sb.WriteString("\n  Attachments:")
		for _, att := range event.Attachments {
			sb.WriteString("\n    ")
			if att.FileID != "" {
				sb.WriteString(att.FileID)
				sb.WriteString(" | ")
			}
			sb.WriteString(att.Title)
			if att.MimeType != "" {
				sb.WriteString(" | ")
				sb.WriteString(att.MimeType)
			}
		}
	}
}

// parseDateTime parses an RFC3339 datetime or date string.
// Returns (date, time) where time may be empty for all-day events.
func parseDateTime(dt string) (string, string) {
	if dt == "" {
		return "", ""
	}
	// All-day event: just a date like "2025-01-19"
	if len(dt) == 10 {
		return dt, ""
	}
	// RFC3339: "2025-01-19T09:00:00-05:00" or similar
	t, err := time.Parse(time.RFC3339, dt)
	if err != nil {
		// Try parsing without timezone
		t, err = time.Parse("2006-01-02T15:04:05", dt)
		if err != nil {
			return dt, ""
		}
	}
	return t.Format("2006-01-02"), t.Format("15:04")
}
