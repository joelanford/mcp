package types

// DocsSearchArgs contains arguments for searching Google Docs via Drive API.
type DocsSearchArgs struct {
	Query    string `json:"query"`
	PageSize int    `json:"page_size"`
}

// DocsGetContentArgs contains arguments for getting document content.
type DocsGetContentArgs struct {
	DocumentID string `json:"document_id"`
}

// DocsListInFolderArgs contains arguments for listing docs in a folder.
type DocsListInFolderArgs struct {
	FolderID string `json:"folder_id"`
	PageSize int    `json:"page_size"`
}

// DocsInspectStructureArgs contains arguments for inspecting document structure.
type DocsInspectStructureArgs struct {
	DocumentID string `json:"document_id"`
	Detailed   bool   `json:"detailed"`
}

// DocsGetCommentsArgs contains arguments for getting document comments.
type DocsGetCommentsArgs struct {
	DocumentID      string `json:"document_id"`
	IncludeResolved bool   `json:"include_resolved"`
}

// SearchResult represents a single item in search results.
type SearchResult struct {
	ID      string `json:"id"`
	Title   string `json:"title,omitempty"`
	Subject string `json:"subject,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// SearchResponse contains paginated search results.
type SearchResponse struct {
	Results       []SearchResult `json:"results"`
	NextPageToken string         `json:"next_page_token,omitempty"`
}

// CalendarListArgs contains arguments for listing calendars.
type CalendarListArgs struct{}

// CalendarGetEventsArgs contains arguments for getting calendar events.
type CalendarGetEventsArgs struct {
	CalendarID         string `json:"calendar_id"`         // Calendar ID, defaults to "primary"
	EventID            string `json:"event_id"`            // Specific event ID (optional)
	TimeMin            string `json:"time_min"`            // Start of time range in RFC3339 format (optional)
	TimeMax            string `json:"time_max"`            // End of time range in RFC3339 format (optional)
	MaxResults         int    `json:"max_results"`         // Maximum number of events to return (default 25)
	Query              string `json:"query"`               // Free text search query (optional)
	IncludeAttachments bool   `json:"include_attachments"` // Include file attachments in response
}
