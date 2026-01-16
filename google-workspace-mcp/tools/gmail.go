package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/gmail/v1"

	"github.com/joelanford/mcp/google-workspace/types"
)

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
func (g *GmailTools) SearchHandler(ctx context.Context, request mcp.CallToolRequest, args types.GmailSearchArgs) (*mcp.CallToolResult, error) {
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

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
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

// GmailMessageResponse represents a single message.
type GmailMessageResponse struct {
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
func (g *GmailTools) GetMessageHandler(ctx context.Context, request mcp.CallToolRequest, args types.GmailGetMessageArgs) (*mcp.CallToolResult, error) {
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

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// extractMessage extracts message details from a Gmail message.
func extractMessage(msg *gmail.Message) GmailMessageResponse {
	response := GmailMessageResponse{
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

// GmailThreadResponse represents a complete thread.
type GmailThreadResponse struct {
	ThreadID string                 `json:"thread_id"`
	Subject  string                 `json:"subject,omitempty"`
	Messages []GmailMessageResponse `json:"messages"`
}

// GetThreadHandler handles gmail_get_thread tool calls.
func (g *GmailTools) GetThreadHandler(ctx context.Context, request mcp.CallToolRequest, args types.GmailGetThreadArgs) (*mcp.CallToolResult, error) {
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

	response := GmailThreadResponse{
		ThreadID: thread.Id,
		Messages: make([]GmailMessageResponse, 0, len(thread.Messages)),
	}

	for _, msg := range thread.Messages {
		msgResponse := extractMessage(msg)
		response.Messages = append(response.Messages, msgResponse)

		// Use the first message's subject as the thread subject
		if response.Subject == "" && msgResponse.Subject != "" {
			response.Subject = msgResponse.Subject
		}
	}

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
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

// GmailLabelsResponse contains the list of labels.
type GmailLabelsResponse struct {
	SystemLabels []GmailLabelInfo `json:"system_labels"`
	UserLabels   []GmailLabelInfo `json:"user_labels"`
}

// ListLabelsHandler handles gmail_list_labels tool calls.
func (g *GmailTools) ListLabelsHandler(ctx context.Context, request mcp.CallToolRequest, args types.GmailListLabelsArgs) (*mcp.CallToolResult, error) {
	labelList, err := g.gmailService.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return mcp.NewToolResultError("failed to list labels: " + err.Error()), nil
	}

	response := GmailLabelsResponse{
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

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
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

// GmailAttachmentResponse contains the attachment data.
type GmailAttachmentResponse struct {
	AttachmentID string `json:"attachment_id"`
	Filename     string `json:"filename,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	Size         int64  `json:"size"`
	Data         string `json:"data"` // base64-encoded
}

// GetAttachmentHandler handles gmail_get_attachment tool calls.
func (g *GmailTools) GetAttachmentHandler(ctx context.Context, request mcp.CallToolRequest, args types.GmailGetAttachmentArgs) (*mcp.CallToolResult, error) {
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

	response := GmailAttachmentResponse{
		AttachmentID: args.AttachmentID,
		Filename:     filename,
		MimeType:     mimeType,
		Size:         attachment.Size,
		Data:         attachment.Data, // Already base64url encoded by the API
	}

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
