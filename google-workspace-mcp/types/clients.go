package types

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Clients holds all Google API service clients.
// Services are initialized once and shared across tools.
// Access to services must go through tool-specific client structs.
type Clients struct {
	calendar *calendar.Service
	docs     *docs.Service
	drive    *drive.Service
	gmail    *gmail.Service
}

// RequiredScopes returns all scopes needed by the clients.
func RequiredScopes() []string {
	return []string{
		calendar.CalendarReadonlyScope,
		docs.DocumentsReadonlyScope,
		drive.DriveReadonlyScope,
		gmail.GmailReadonlyScope,
	}
}

// NewClients creates all Google API clients with read-only scopes.
// It validates that Application Default Credentials are available.
func NewClients(ctx context.Context) (*Clients, error) {
	scopes := RequiredScopes()

	// Validate ADC credentials exist
	_, err := google.FindDefaultCredentials(ctx, scopes...)
	if err != nil {
		return nil, fmt.Errorf("Google credentials not found or insufficient scopes.\n\n"+
			"Run the following command to authenticate:\n"+
			"  gcloud auth application-default login --scopes=\"%s\"",
			strings.Join(scopes, ","))
	}

	calendarService, err := calendar.NewService(ctx,
		option.WithScopes(calendar.CalendarReadonlyScope),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	docsService, err := docs.NewService(ctx,
		option.WithScopes(docs.DocumentsReadonlyScope),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docs service: %w", err)
	}

	driveService, err := drive.NewService(ctx,
		option.WithScopes(drive.DriveReadonlyScope),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	gmailService, err := gmail.NewService(ctx,
		option.WithScopes(gmail.GmailReadonlyScope),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	return &Clients{
		calendar: calendarService,
		docs:     docsService,
		drive:    driveService,
		gmail:    gmailService,
	}, nil
}

// DocsClients provides access to services needed by Docs tools.
type DocsClients struct {
	Docs  *docs.Service
	Drive *drive.Service
}

// ForDocs returns clients scoped for Docs tools.
func (c *Clients) ForDocs() *DocsClients {
	return &DocsClients{
		Docs:  c.docs,
		Drive: c.drive,
	}
}

// CalendarClients provides access to services needed by Calendar tools.
type CalendarClients struct {
	Calendar *calendar.Service
}

// ForCalendar returns clients scoped for Calendar tools.
func (c *Clients) ForCalendar() *CalendarClients {
	return &CalendarClients{
		Calendar: c.calendar,
	}
}

// GmailClients provides access to services needed by Gmail tools.
type GmailClients struct {
	Gmail *gmail.Service
}

// ForGmail returns clients scoped for Gmail tools.
func (c *Clients) ForGmail() *GmailClients {
	return &GmailClients{
		Gmail: c.gmail,
	}
}
