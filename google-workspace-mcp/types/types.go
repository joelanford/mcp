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

// GmailSearchArgs contains arguments for searching Gmail messages.
type GmailSearchArgs struct {
	Query     string `json:"query"`      // Gmail search query using standard operators
	PageSize  int    `json:"page_size"`  // Maximum results to return (default 10, max 100)
	PageToken string `json:"page_token"` // Pagination token from previous response
}

// GmailGetMessageArgs contains arguments for getting a Gmail message.
type GmailGetMessageArgs struct {
	MessageID string `json:"message_id"` // Gmail message ID
}

// GmailGetThreadArgs contains arguments for getting a Gmail thread.
type GmailGetThreadArgs struct {
	ThreadID string `json:"thread_id"` // Gmail thread ID
}

// GmailListLabelsArgs contains arguments for listing Gmail labels.
type GmailListLabelsArgs struct{}

// GmailGetAttachmentArgs contains arguments for getting a Gmail attachment.
type GmailGetAttachmentArgs struct {
	MessageID    string `json:"message_id"`    // Message containing the attachment
	AttachmentID string `json:"attachment_id"` // Attachment ID from gmail_get_message
}
