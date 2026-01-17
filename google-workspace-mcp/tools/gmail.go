package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/gmail/v1"

	"github.com/joelanford/mcp/google-workspace-mcp/types"
)

// GmailSearchRequest contains arguments for searching Gmail messages.
type GmailSearchRequest struct {
	Query     string `json:"query"`      // Gmail search query using standard operators
	PageSize  int    `json:"page_size"`  // Maximum results to return (default 10, max 100)
	PageToken string `json:"page_token"` // Pagination token from previous response
}

// GmailGetMessageRequest contains arguments for getting a Gmail message.
type GmailGetMessageRequest struct {
	MessageID string `json:"message_id"` // Gmail message ID
}

// GmailGetThreadRequest contains arguments for getting a Gmail thread.
type GmailGetThreadRequest struct {
	ThreadID string `json:"thread_id"` // Gmail thread ID
}

// GmailListLabelsRequest contains arguments for listing Gmail labels.
type GmailListLabelsRequest struct{}

// GmailGetAttachmentRequest contains arguments for getting a Gmail attachment.
type GmailGetAttachmentRequest struct {
	MessageID    string `json:"message_id"`    // Message containing the attachment
	AttachmentID string `json:"attachment_id"` // Attachment ID from gmail_get_message
}

// GmailTools provides Gmail API tools.
type GmailTools struct {
	gmailService *gmail.Service
}

// NewGmailTools creates a new GmailTools instance from the provided clients.
func NewGmailTools(clients *types.GmailClients) *GmailTools {
	return &GmailTools{
		gmailService: clients.Gmail,
	}
}

// SearchTool returns the tool definition for searching Gmail messages.
func (g *GmailTools) SearchTool() mcp.Tool {
	return mcp.NewTool("gmail_search",
		mcp.WithDescription(`Searches for emails in Gmail using Gmail's search syntax.

Supports all Gmail search operators like:
  from:, to:, subject:, has:attachment, is:unread, after:, before:, label:, etc.

Returns message and thread IDs for use with gmail_get_message and gmail_get_thread.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Gmail search query (e.g., 'from:example@gmail.com is:unread')"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Maximum number of results to return (default 10, max 100)"),
			mcp.Min(1),
			mcp.Max(100),
		),
		mcp.WithString("page_token",
			mcp.Description("Page token for retrieving subsequent pages of results"),
		),
	)
}

// GmailSearchResult represents a single search result.
// Note: Gmail messages.list only returns id and threadId.
// Use gmail_get_message for full details (subject, from, to, body, etc.).
type GmailSearchResult struct {
	MessageID string `json:"message_id"`
	ThreadID  string `json:"thread_id"`
}

// GmailSearchResponse contains search results with pagination.
type GmailSearchResponse struct {
	Results       []GmailSearchResult `json:"results"`
	NextPageToken string              `json:"next_page_token,omitempty"`
}

// SearchHandler handles gmail_search tool calls.
func (g *GmailTools) SearchHandler(ctx context.Context, request mcp.CallToolRequest, args GmailSearchRequest) (*mcp.CallToolResult, error) {
	if args.Query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	pageSize := args.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	call := g.gmailService.Users.Messages.List("me").
		Context(ctx).
		Q(args.Query).
		MaxResults(int64(pageSize))

	if args.PageToken != "" {
		call = call.PageToken(args.PageToken)
	}

	msgList, err := call.Do()
	if err != nil {
		return mcp.NewToolResultError("failed to search messages: " + err.Error()), nil
	}

	results := make([]GmailSearchResult, 0, len(msgList.Messages))
	for _, msg := range msgList.Messages {
		results = append(results, GmailSearchResult{
			MessageID: msg.Id,
			ThreadID:  msg.ThreadId,
		})
	}

	response := GmailSearchResponse{
		Results:       results,
		NextPageToken: msgList.NextPageToken,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// GetMessageTool returns the tool definition for getting a Gmail message.
func (g *GmailTools) GetMessageTool() mcp.Tool {
	return mcp.NewTool("gmail_get_message",
		mcp.WithDescription(`Retrieves a specific email message by ID.

Returns the full message content including:
  - Headers (subject, from, to, cc, date)
  - Body content (prefers plain text, falls back to HTML converted to text)
  - Attachment metadata (filename, mimeType, size, attachmentId)`),
		mcp.WithString("message_id",
			mcp.Required(),
			mcp.Description("The message ID (from gmail_search results)"),
		),
	)
}

// GmailAttachmentInfo represents attachment metadata.
type GmailAttachmentInfo struct {
	AttachmentID string `json:"attachment_id"`
	Filename     string `json:"filename"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
}

// GmailGetMessageResponse represents a single message.
type GmailGetMessageResponse struct {
	MessageID   string                `json:"message_id"`
	ThreadID    string                `json:"thread_id"`
	Subject     string                `json:"subject,omitempty"`
	From        string                `json:"from,omitempty"`
	To          string                `json:"to,omitempty"`
	Cc          string                `json:"cc,omitempty"`
	Date        string                `json:"date,omitempty"`
	Body        string                `json:"body,omitempty"`
	Attachments []GmailAttachmentInfo `json:"attachments,omitempty"`
}

// GetMessageHandler handles gmail_get_message tool calls.
func (g *GmailTools) GetMessageHandler(ctx context.Context, request mcp.CallToolRequest, args GmailGetMessageRequest) (*mcp.CallToolResult, error) {
	if args.MessageID == "" {
		return mcp.NewToolResultError("message_id is required"), nil
	}

	msg, err := g.gmailService.Users.Messages.Get("me", args.MessageID).
		Context(ctx).
		Format("full").
		Do()
	if err != nil {
		return mcp.NewToolResultError("failed to get message: " + err.Error()), nil
	}

	response := extractMessage(msg)

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// extractMessage extracts message details from a Gmail message.
func extractMessage(msg *gmail.Message) GmailGetMessageResponse {
	response := GmailGetMessageResponse{
		MessageID: msg.Id,
		ThreadID:  msg.ThreadId,
	}

	// Extract headers
	if msg.Payload != nil {
		for _, header := range msg.Payload.Headers {
			switch strings.ToLower(header.Name) {
			case "subject":
				response.Subject = header.Value
			case "from":
				response.From = header.Value
			case "to":
				response.To = header.Value
			case "cc":
				response.Cc = header.Value
			case "date":
				response.Date = header.Value
			}
		}

		// Extract body and attachments
		response.Body, response.Attachments = extractBodyAndAttachments(msg.Payload)
	}

	return response
}

// extractBodyAndAttachments extracts the body text and attachment info from a message payload.
func extractBodyAndAttachments(payload *gmail.MessagePart) (string, []GmailAttachmentInfo) {
	var plainText, htmlText string
	var attachments []GmailAttachmentInfo

	// Recursive function to process message parts
	var processPart func(part *gmail.MessagePart)
	processPart = func(part *gmail.MessagePart) {
		if part == nil {
			return
		}

		// Check if this part is an attachment
		if part.Filename != "" && part.Body != nil && part.Body.AttachmentId != "" {
			attachments = append(attachments, GmailAttachmentInfo{
				AttachmentID: part.Body.AttachmentId,
				Filename:     part.Filename,
				MimeType:     part.MimeType,
				Size:         part.Body.Size,
			})
			return
		}

		// Extract body content
		if part.Body != nil && part.Body.Data != "" {
			decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err == nil {
				content := string(decoded)
				switch {
				case strings.HasPrefix(part.MimeType, "text/plain"):
					plainText = content
				case strings.HasPrefix(part.MimeType, "text/html"):
					htmlText = content
				}
			}
		}

		// Process child parts
		for _, child := range part.Parts {
			processPart(child)
		}
	}

	processPart(payload)

	// Prefer plain text, fall back to HTML (stripped of tags)
	body := plainText
	if body == "" && htmlText != "" {
		body = stripHTMLTags(htmlText)
	}

	return body, attachments
}

// stripHTMLTags removes HTML tags from a string (simple implementation).
func stripHTMLTags(html string) string {
	var result strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}
	// Clean up excessive whitespace
	text := result.String()
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	// Collapse multiple newlines
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(text)
}

// GetThreadTool returns the tool definition for getting a Gmail thread.
func (g *GmailTools) GetThreadTool() mcp.Tool {
	return mcp.NewTool("gmail_get_thread",
		mcp.WithDescription(`Retrieves a complete email thread (conversation) by ID.

Returns all messages in the thread in chronological order, each with:
  - Headers (subject, from, to, cc, date)
  - Body content
  - Attachment metadata`),
		mcp.WithString("thread_id",
			mcp.Required(),
			mcp.Description("The thread ID (from gmail_search results)"),
		),
	)
}

// GmailGetThreadResponse represents a complete thread.
type GmailGetThreadResponse struct {
	ThreadID string                 `json:"thread_id"`
	Subject  string                 `json:"subject,omitempty"`
	Messages []GmailGetMessageResponse `json:"messages"`
}

// GetThreadHandler handles gmail_get_thread tool calls.
func (g *GmailTools) GetThreadHandler(ctx context.Context, request mcp.CallToolRequest, args GmailGetThreadRequest) (*mcp.CallToolResult, error) {
	if args.ThreadID == "" {
		return mcp.NewToolResultError("thread_id is required"), nil
	}

	thread, err := g.gmailService.Users.Threads.Get("me", args.ThreadID).
		Context(ctx).
		Format("full").
		Do()
	if err != nil {
		return mcp.NewToolResultError("failed to get thread: " + err.Error()), nil
	}

	response := GmailGetThreadResponse{
		ThreadID: thread.Id,
		Messages: make([]GmailGetMessageResponse, 0, len(thread.Messages)),
	}

	for _, msg := range thread.Messages {
		msgResponse := extractMessage(msg)
		response.Messages = append(response.Messages, msgResponse)

		// Use the first message's subject as the thread subject
		if response.Subject == "" && msgResponse.Subject != "" {
			response.Subject = msgResponse.Subject
		}
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// ListLabelsTool returns the tool definition for listing Gmail labels.
func (g *GmailTools) ListLabelsTool() mcp.Tool {
	return mcp.NewTool("gmail_list_labels",
		mcp.WithDescription(`Lists all Gmail labels in the user's mailbox.

Returns both system labels (INBOX, SENT, TRASH, etc.) and user-created labels.`),
	)
}

// GmailLabelInfo represents a single label.
type GmailLabelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// GmailListLabelsResponse contains the list of labels.
type GmailListLabelsResponse struct {
	SystemLabels []GmailLabelInfo `json:"system_labels"`
	UserLabels   []GmailLabelInfo `json:"user_labels"`
}

// ListLabelsHandler handles gmail_list_labels tool calls.
func (g *GmailTools) ListLabelsHandler(ctx context.Context, request mcp.CallToolRequest, args GmailListLabelsRequest) (*mcp.CallToolResult, error) {
	labelList, err := g.gmailService.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return mcp.NewToolResultError("failed to list labels: " + err.Error()), nil
	}

	response := GmailListLabelsResponse{
		SystemLabels: []GmailLabelInfo{},
		UserLabels:   []GmailLabelInfo{},
	}

	for _, label := range labelList.Labels {
		info := GmailLabelInfo{
			ID:   label.Id,
			Name: label.Name,
			Type: label.Type,
		}
		if label.Type == "system" {
			response.SystemLabels = append(response.SystemLabels, info)
		} else {
			response.UserLabels = append(response.UserLabels, info)
		}
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// GetAttachmentTool returns the tool definition for getting a Gmail attachment.
func (g *GmailTools) GetAttachmentTool() mcp.Tool {
	return mcp.NewTool("gmail_get_attachment",
		mcp.WithDescription(`Downloads an email attachment by ID.

Returns the attachment content as base64-encoded data along with metadata.
Use the attachment_id from gmail_get_message results.`),
		mcp.WithString("message_id",
			mcp.Required(),
			mcp.Description("The message ID containing the attachment"),
		),
		mcp.WithString("attachment_id",
			mcp.Required(),
			mcp.Description("The attachment ID (from gmail_get_message results)"),
		),
	)
}

// GmailGetAttachmentResponse contains the attachment data.
type GmailGetAttachmentResponse struct {
	AttachmentID string `json:"attachment_id"`
	Filename     string `json:"filename,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	Size         int64  `json:"size"`
	Data         string `json:"data"` // base64-encoded
}

// GetAttachmentHandler handles gmail_get_attachment tool calls.
func (g *GmailTools) GetAttachmentHandler(ctx context.Context, request mcp.CallToolRequest, args GmailGetAttachmentRequest) (*mcp.CallToolResult, error) {
	if args.MessageID == "" {
		return mcp.NewToolResultError("message_id is required"), nil
	}
	if args.AttachmentID == "" {
		return mcp.NewToolResultError("attachment_id is required"), nil
	}

	// First, get the message to find attachment metadata
	msg, err := g.gmailService.Users.Messages.Get("me", args.MessageID).
		Context(ctx).
		Format("full").
		Do()
	if err != nil {
		return mcp.NewToolResultError("failed to get message: " + err.Error()), nil
	}

	// Find the attachment metadata
	var filename, mimeType string
	var findAttachment func(part *gmail.MessagePart)
	findAttachment = func(part *gmail.MessagePart) {
		if part == nil {
			return
		}
		if part.Body != nil && part.Body.AttachmentId == args.AttachmentID {
			filename = part.Filename
			mimeType = part.MimeType
		}
		for _, child := range part.Parts {
			findAttachment(child)
		}
	}
	if msg.Payload != nil {
		findAttachment(msg.Payload)
	}

	// Get the attachment data
	attachment, err := g.gmailService.Users.Messages.Attachments.Get("me", args.MessageID, args.AttachmentID).
		Context(ctx).
		Do()
	if err != nil {
		return mcp.NewToolResultError("failed to get attachment: " + err.Error()), nil
	}

	response := GmailGetAttachmentResponse{
		AttachmentID: args.AttachmentID,
		Filename:     filename,
		MimeType:     mimeType,
		Size:         attachment.Size,
		Data:         attachment.Data, // Already base64url encoded by the API
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// MarshalCompact returns a compact text representation of the search results.
// Format: header line followed by "message_id | thread_id" per line.
func (g GmailSearchResponse) MarshalCompact() string {
	var sb strings.Builder
	if len(g.Results) > 0 {
		sb.WriteString("Message ID | Thread ID\n")
		for _, r := range g.Results {
			sb.WriteString(r.MessageID)
			sb.WriteString(" | ")
			sb.WriteString(r.ThreadID)
			sb.WriteString("\n")
		}
	}
	if g.NextPageToken != "" {
		sb.WriteString("\nNext Page Token: ")
		sb.WriteString(g.NextPageToken)
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// MarshalCompact returns a compact text representation of the message.
func (g GmailGetMessageResponse) MarshalCompact() string {
	var sb strings.Builder

	if g.From != "" {
		sb.WriteString("From: ")
		sb.WriteString(g.From)
		sb.WriteString("\n")
	}
	if g.To != "" {
		sb.WriteString("To: ")
		sb.WriteString(g.To)
		sb.WriteString("\n")
	}
	if g.Cc != "" {
		sb.WriteString("Cc: ")
		sb.WriteString(g.Cc)
		sb.WriteString("\n")
	}
	if g.Date != "" {
		sb.WriteString("Date: ")
		sb.WriteString(g.Date)
		sb.WriteString("\n")
	}
	if g.Subject != "" {
		sb.WriteString("Subject: ")
		sb.WriteString(g.Subject)
		sb.WriteString("\n")
	}

	if g.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(g.Body)
	}

	if len(g.Attachments) > 0 {
		sb.WriteString("\n\nAttachments:")
		for _, att := range g.Attachments {
			sb.WriteString("\n  ")
			sb.WriteString(att.AttachmentID)
			sb.WriteString(" | ")
			sb.WriteString(att.Filename)
			sb.WriteString(" | ")
			sb.WriteString(att.MimeType)
			sb.WriteString(" | ")
			sb.WriteString(formatSize(att.Size))
		}
	}

	return sb.String()
}

// MarshalCompact returns a compact text representation of the thread.
func (g GmailGetThreadResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString("Thread: ")
	sb.WriteString(g.ThreadID)
	if g.Subject != "" {
		sb.WriteString("\nSubject: ")
		sb.WriteString(g.Subject)
	}
	sb.WriteString("\n")

	for i, msg := range g.Messages {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		} else {
			sb.WriteString("\n")
		}
		sb.WriteString(msg.MarshalCompact())
	}

	return sb.String()
}

// MarshalCompact returns a compact text representation of the labels.
func (g GmailListLabelsResponse) MarshalCompact() string {
	var sb strings.Builder

	if len(g.SystemLabels) > 0 {
		sb.WriteString("System: ")
		for i, label := range g.SystemLabels {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(label.Name)
		}
	}

	if len(g.UserLabels) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("User: ")
		for i, label := range g.UserLabels {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(label.Name)
		}
	}

	return sb.String()
}

// MarshalCompact returns a compact text representation of the attachment.
// Note: Attachments contain binary data, so compact format just shows metadata
// and includes the base64 data which is unavoidable.
func (g GmailGetAttachmentResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString("Attachment: ")
	sb.WriteString(g.AttachmentID)
	if g.Filename != "" {
		sb.WriteString("\nFilename: ")
		sb.WriteString(g.Filename)
	}
	if g.MimeType != "" {
		sb.WriteString("\nType: ")
		sb.WriteString(g.MimeType)
	}
	sb.WriteString("\nSize: ")
	sb.WriteString(formatSize(g.Size))
	sb.WriteString("\n\nData (base64):\n")
	sb.WriteString(g.Data)
	return sb.String()
}

// formatSize formats a byte size into a human-readable string.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
