package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"

	"github.com/joelanford/mcp/google-workspace-mcp/types"
)

// multipleNewlinesRe matches 3 or more consecutive newlines.
var multipleNewlinesRe = regexp.MustCompile(`\n{3,}`)

// DocsSearchRequest contains arguments for searching Google Docs via Drive API.
type DocsSearchRequest struct {
	Query          string `json:"query"`
	PageSize       int    `json:"page_size"`
	PageToken      string `json:"page_token"`       // Continue from previous page
	OrderBy        string `json:"order_by"`         // Sort order: createdTime, modifiedTime, name, name_natural
	ModifiedAfter  string `json:"modified_after"`   // RFC3339 date - only docs modified after this time
	ModifiedBefore string `json:"modified_before"`  // RFC3339 date - only docs modified before this time
	OwnerEmail     string `json:"owner_email"`      // Filter to docs owned by this email
}

// DocsGetContentRequest contains arguments for getting document content.
type DocsGetContentRequest struct {
	DocumentID string `json:"document_id"`
}

// DocsListInFolderRequest contains arguments for listing docs in a folder.
type DocsListInFolderRequest struct {
	FolderID       string `json:"folder_id"`
	PageSize       int    `json:"page_size"`
	PageToken      string `json:"page_token"`       // Continue from previous page
	OrderBy        string `json:"order_by"`         // Sort order: createdTime, modifiedTime, name, name_natural
	ModifiedAfter  string `json:"modified_after"`   // RFC3339 date filter
	ModifiedBefore string `json:"modified_before"`  // RFC3339 date filter
}

// DocsGetCommentsRequest contains arguments for getting document comments.
type DocsGetCommentsRequest struct {
	DocumentID      string `json:"document_id"`
	IncludeResolved bool   `json:"include_resolved"`
	PageToken       string `json:"page_token"`     // Continue from previous page
	PageSize        int    `json:"page_size"`      // Max comments per page (default 100)
	ModifiedAfter   string `json:"modified_after"` // RFC3339 date - only comments modified after this time
}

// DocsSearchResult represents a single item in docs search results.
type DocsSearchResult struct {
	ID      string `json:"id"`
	Title   string `json:"title,omitempty"`
	Subject string `json:"subject,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// DocsSearchResponse contains paginated search results.
type DocsSearchResponse struct {
	Results       []DocsSearchResult `json:"results"`
	NextPageToken string             `json:"next_page_token,omitempty"`
}

// MarshalCompact returns a compact text representation of the search response.
// Format: one result per line as "id | title" or "id | subject" (whichever is set)
// with optional "Next Page Token: <token>" appended if pagination continues.
func (s DocsSearchResponse) MarshalCompact() string {
	var sb strings.Builder
	for _, r := range s.Results {
		sb.WriteString(r.ID)
		sb.WriteString(" | ")
		if r.Title != "" {
			sb.WriteString(r.Title)
		} else if r.Subject != "" {
			sb.WriteString(r.Subject)
		}
		sb.WriteString("\n")
	}
	if s.NextPageToken != "" {
		sb.WriteString("\nNext Page Token: ")
		sb.WriteString(s.NextPageToken)
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// DocsTools provides Google Docs API tools.
type DocsTools struct {
	docsService  *docs.Service
	driveService *drive.Service
}

// NewDocsTools creates a new DocsTools instance from the provided clients.
func NewDocsTools(clients *types.DocsClients) *DocsTools {
	return &DocsTools{
		docsService:  clients.Docs,
		driveService: clients.Drive,
	}
}

// SearchTool returns the tool definition for searching Google Docs.
func (d *DocsTools) SearchTool() mcp.Tool {
	return mcp.NewTool("docs_search",
		mcp.WithDescription(`Searches for Google Docs by name using Drive API (mimeType filter).

Returns:
    str: A formatted list of Google Docs matching the search query.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search string to find in document names"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Maximum number of results to return (default 10)"),
			mcp.Min(1),
			mcp.Max(100),
		),
		mcp.WithString("page_token",
			mcp.Description("Page token from previous response to continue pagination"),
		),
		mcp.WithString("order_by",
			mcp.Description("Sort order: createdTime, modifiedTime, name, name_natural (append ' desc' for descending)"),
		),
		mcp.WithString("modified_after",
			mcp.Description("Only include docs modified after this date (RFC3339 format, e.g. '2025-01-01T00:00:00Z')"),
		),
		mcp.WithString("modified_before",
			mcp.Description("Only include docs modified before this date (RFC3339 format)"),
		),
		mcp.WithString("owner_email",
			mcp.Description("Only include docs owned by this email address"),
		),
	)
}

// SearchHandler handles docs_search tool calls.
func (d *DocsTools) SearchHandler(ctx context.Context, request mcp.CallToolRequest, args DocsSearchRequest) (*mcp.CallToolResult, error) {
	if args.Query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	pageSize := args.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	// Escape single quotes in query
	escapedQuery := strings.ReplaceAll(args.Query, "'", "\\'")

	// Build query: search by name, filter to Google Docs, exclude trashed
	q := fmt.Sprintf("name contains '%s' and mimeType='application/vnd.google-apps.document' and trashed=false", escapedQuery)

	// Add date filters
	if args.ModifiedAfter != "" {
		q += fmt.Sprintf(" and modifiedTime > '%s'", args.ModifiedAfter)
	}
	if args.ModifiedBefore != "" {
		q += fmt.Sprintf(" and modifiedTime < '%s'", args.ModifiedBefore)
	}
	// Add owner filter
	if args.OwnerEmail != "" {
		q += fmt.Sprintf(" and '%s' in owners", args.OwnerEmail)
	}

	call := d.driveService.Files.List().
		Context(ctx).
		Q(q).
		PageSize(int64(pageSize)).
		Fields("nextPageToken, files(id, name, createdTime, modifiedTime, webViewLink)").
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true)

	// Apply pagination
	if args.PageToken != "" {
		call = call.PageToken(args.PageToken)
	}
	// Apply sorting
	if args.OrderBy != "" {
		call = call.OrderBy(args.OrderBy)
	}

	fileList, err := call.Do()
	if err != nil {
		return mcp.NewToolResultError("failed to search documents: " + err.Error()), nil
	}

	results := make([]DocsSearchResult, 0, len(fileList.Files))
	for _, f := range fileList.Files {
		results = append(results, DocsSearchResult{
			ID:    f.Id,
			Title: f.Name,
		})
	}

	response := DocsSearchResponse{
		Results:       results,
		NextPageToken: fileList.NextPageToken,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// DocsGetContentResponse represents the structured response for document content.
type DocsGetContentResponse struct {
	DocID    string            `json:"docId"`
	DocTitle string            `json:"docTitle"`
	Tabs     []DocsTabContent  `json:"tabs"`
}

// DocsTabContent represents a single tab's content.
type DocsTabContent struct {
	TabID       string `json:"tabId"`
	TabTitle    string `json:"tabTitle"`
	TabMarkdown string `json:"tabMarkdown"`
}

// GetContentTool returns the tool definition for fetching document content.
func (d *DocsTools) GetContentTool() mcp.Tool {
	return mcp.NewTool("docs_get_content",
		mcp.WithDescription(`Retrieves a Google Doc and converts its content to Markdown.

Supports multi-tab documents. Each tab's content is converted to well-formatted Markdown with proper heading levels, lists, tables, links, and text formatting (bold, italic, strikethrough).

Returns a JSON object:
  - docId: The document ID
  - docTitle: The document title
  - tabs: Array of tab objects, each containing:
    - tabId: The tab identifier
    - tabTitle: The tab title
    - tabMarkdown: The tab content as Markdown`),
		mcp.WithString("document_id",
			mcp.Required(),
			mcp.Description("The document ID (from the URL or docs_search results)"),
		),
	)
}

// GetContentHandler handles docs_get_content tool calls.
func (d *DocsTools) GetContentHandler(ctx context.Context, request mcp.CallToolRequest, args DocsGetContentRequest) (*mcp.CallToolResult, error) {
	if args.DocumentID == "" {
		return mcp.NewToolResultError("document_id is required"), nil
	}

	doc, err := d.docsService.Documents.Get(args.DocumentID).
		IncludeTabsContent(true).
		Context(ctx).
		Do()
	if err != nil {
		return mcp.NewToolResultError("failed to get document: " + err.Error()), nil
	}

	// Build structured response
	response := DocsGetContentResponse{
		DocID:    args.DocumentID,
		DocTitle: doc.Title,
		Tabs:     []DocsTabContent{},
	}

	// Process all tabs (with recursive child tab support)
	if len(doc.Tabs) > 0 {
		response.Tabs = collectAllTabs(doc.Tabs, doc.Title)
	} else if doc.Body != nil {
		// Fallback for legacy single-tab documents
		var content strings.Builder
		extractMarkdownContent(&content, doc.Body.Content, nil, 0)
		response.Tabs = append(response.Tabs, DocsTabContent{
			TabID:       "",
			TabTitle:    doc.Title,
			TabMarkdown: normalizeNewlines(content.String()),
		})
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// normalizeNewlines collapses runs of 3+ newlines down to 2 (one blank line).
func normalizeNewlines(s string) string {
	return multipleNewlinesRe.ReplaceAllString(s, "\n\n")
}

// collectAllTabs recursively collects all tabs and their children into DocsTabContent slices.
func collectAllTabs(tabs []*docs.Tab, docTitle string) []DocsTabContent {
	var result []DocsTabContent

	for _, tab := range tabs {
		if tab.TabProperties != nil && tab.DocumentTab != nil {
			tabTitle := tab.TabProperties.Title
			if tabTitle == "" {
				tabTitle = docTitle
			}

			// Extract markdown content for this tab
			var content strings.Builder
			if tab.DocumentTab.Body != nil {
				extractMarkdownContent(&content, tab.DocumentTab.Body.Content, tab.DocumentTab.Lists, 0)
			}

			result = append(result, DocsTabContent{
				TabID:       tab.TabProperties.TabId,
				TabTitle:    tabTitle,
				TabMarkdown: normalizeNewlines(content.String()),
			})
		}

		// Process child tabs recursively
		if len(tab.ChildTabs) > 0 {
			result = append(result, collectAllTabs(tab.ChildTabs, docTitle)...)
		}
	}

	return result
}

// extractMarkdownContent extracts text from document structural elements and converts to markdown.
// headingOffset adjusts heading levels (e.g., 2 means HEADING_1 becomes ###).
func extractMarkdownContent(sb *strings.Builder, elements []*docs.StructuralElement, lists map[string]docs.List, headingOffset int) {
	for _, elem := range elements {
		if elem.Paragraph != nil {
			processParagraph(sb, elem.Paragraph, lists, headingOffset)
		}
		if elem.Table != nil {
			processTable(sb, elem.Table, lists, headingOffset)
		}
		if elem.SectionBreak != nil {
			// Skip section breaks at the start of the document (nothing written yet or only whitespace)
			if sb.Len() > 0 && strings.TrimSpace(sb.String()) != "" {
				sb.WriteString("\n---\n\n")
			}
		}
	}
}

// processParagraph converts a paragraph to markdown.
// headingOffset adjusts heading levels (e.g., 2 means HEADING_1 becomes ###).
func processParagraph(sb *strings.Builder, para *docs.Paragraph, lists map[string]docs.List, headingOffset int) {
	if para == nil {
		return
	}

	// Determine heading level from paragraph style
	headingLevel := 0
	if para.ParagraphStyle != nil {
		switch para.ParagraphStyle.NamedStyleType {
		case "TITLE":
			headingLevel = 1
		case "HEADING_1":
			headingLevel = 1
		case "HEADING_2":
			headingLevel = 2
		case "HEADING_3":
			headingLevel = 3
		case "HEADING_4":
			headingLevel = 4
		case "HEADING_5":
			headingLevel = 5
		case "HEADING_6":
			headingLevel = 6
		}
	}

	// Apply offset and cap at 6
	headingPrefix := ""
	if headingLevel > 0 {
		adjustedLevel := min(headingLevel+headingOffset, 6)
		headingPrefix = strings.Repeat("#", adjustedLevel) + " "
	}

	// Handle bullet points
	bulletPrefix := ""
	if para.Bullet != nil {
		nestingLevel := 0
		if para.Bullet.NestingLevel > 0 {
			nestingLevel = int(para.Bullet.NestingLevel)
		}
		indent := strings.Repeat("  ", nestingLevel)

		// Check if it's an ordered list
		isOrdered := false
		if lists != nil && para.Bullet.ListId != "" {
			if list, ok := lists[para.Bullet.ListId]; ok {
				if list.ListProperties != nil && len(list.ListProperties.NestingLevels) > nestingLevel {
					nl := list.ListProperties.NestingLevels[nestingLevel]
					if nl.GlyphType == "DECIMAL" || nl.GlyphType == "ALPHA" || nl.GlyphType == "ROMAN" {
						isOrdered = true
					}
				}
			}
		}

		if isOrdered {
			bulletPrefix = indent + "1. "
		} else {
			bulletPrefix = indent + "- "
		}
	}

	// Build paragraph content with inline formatting
	var paraContent strings.Builder
	for _, e := range para.Elements {
		if e.TextRun != nil {
			text := e.TextRun.Content
			if text == "" {
				continue
			}

			// Apply inline formatting
			formatted := formatTextRun(e.TextRun)
			paraContent.WriteString(formatted)
		}
	}

	content := paraContent.String()

	// Skip empty paragraphs (but keep newlines for spacing)
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		if !strings.HasSuffix(sb.String(), "\n\n") {
			sb.WriteString("\n")
		}
		return
	}

	// Write with appropriate prefix
	if headingPrefix != "" {
		// Ensure blank line before heading (if not at document start)
		if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n\n") {
			sb.WriteString("\n")
		}
		// Remove trailing newline for heading, add it after
		content = strings.TrimSuffix(content, "\n")
		sb.WriteString(headingPrefix)
		sb.WriteString(content)
		sb.WriteString("\n\n")
	} else if bulletPrefix != "" {
		content = strings.TrimSuffix(content, "\n")
		// Skip empty or malformed list items (just dashes, whitespace)
		trimmedContent := strings.TrimSpace(content)
		if trimmedContent == "" || trimmedContent == "-" {
			return
		}
		sb.WriteString(bulletPrefix)
		sb.WriteString(content)
		sb.WriteString("\n")
	} else {
		// Regular paragraph - ensure proper spacing
		content = strings.TrimSuffix(content, "\n")
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}
}

// formatTextRun applies markdown formatting to a text run.
func formatTextRun(tr *docs.TextRun) string {
	if tr == nil || tr.Content == "" {
		return ""
	}

	text := tr.Content
	style := tr.TextStyle

	// Replace vertical tabs (U+000B) with double newlines
	text = strings.ReplaceAll(text, "\u000B", "\n\n")

	// Convert smart typography to ASCII equivalents for compatibility
	text = strings.ReplaceAll(text, "\u2018", "'")  // left single quote
	text = strings.ReplaceAll(text, "\u2019", "'")  // right single quote / apostrophe
	text = strings.ReplaceAll(text, "\u201C", "\"") // left double quote
	text = strings.ReplaceAll(text, "\u201D", "\"") // right double quote
	text = strings.ReplaceAll(text, "\u2014", "--") // em dash

	// Don't format whitespace-only content
	if strings.TrimSpace(text) == "" {
		return text
	}

	// Check for link
	if style != nil && style.Link != nil && style.Link.Url != "" {
		// Preserve trailing whitespace/newlines
		trimmed := strings.TrimSpace(text)
		if trimmed != "" {
			suffix := text[len(strings.TrimRight(text, " \t\n")):]
			return fmt.Sprintf("[%s](%s)%s", trimmed, style.Link.Url, suffix)
		}
	}

	// Apply bold and italic with trimmed whitespace
	if style != nil && (style.Bold || style.Italic || style.Strikethrough) {
		// Extract leading/trailing whitespace (including newlines) to keep outside markers
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return text
		}
		leadingSpace := text[:len(text)-len(strings.TrimLeft(text, " \t\n"))]
		trailingSpace := text[len(strings.TrimRight(text, " \t\n")):]

		// Apply formatting to trimmed content only
		formatted := trimmed
		if style.Bold && style.Italic {
			formatted = "***" + formatted + "***"
		} else if style.Bold {
			formatted = "**" + formatted + "**"
		} else if style.Italic {
			formatted = "*" + formatted + "*"
		}

		if style.Strikethrough {
			formatted = "~~" + formatted + "~~"
		}

		text = leadingSpace + formatted + trailingSpace
	}

	return text
}

// processTable converts a table to markdown.
func processTable(sb *strings.Builder, table *docs.Table, lists map[string]docs.List, headingOffset int) {
	if table == nil || len(table.TableRows) == 0 {
		return
	}

	sb.WriteString("\n")

	for rowIdx, row := range table.TableRows {
		sb.WriteString("|")
		for _, cell := range row.TableCells {
			var cellContent strings.Builder
			extractMarkdownContent(&cellContent, cell.Content, lists, headingOffset)
			// Clean up cell content - remove newlines, trim
			cellText := strings.TrimSpace(cellContent.String())
			cellText = strings.ReplaceAll(cellText, "\n", " ")
			sb.WriteString(" ")
			sb.WriteString(cellText)
			sb.WriteString(" |")
		}
		sb.WriteString("\n")

		// Add header separator after first row
		if rowIdx == 0 {
			sb.WriteString("|")
			for range row.TableCells {
				sb.WriteString(" --- |")
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
}

// ListInFolderTool returns the tool definition for listing docs in a folder.
func (d *DocsTools) ListInFolderTool() mcp.Tool {
	return mcp.NewTool("docs_list_in_folder",
		mcp.WithDescription(`Lists Google Docs within a specific Drive folder.

Returns:
    str: A formatted list of Google Docs in the specified folder.`),
		mcp.WithString("folder_id",
			mcp.Description("The folder ID (defaults to 'root' for root folder)"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Maximum number of results to return (default 100)"),
			mcp.Min(1),
			mcp.Max(1000),
		),
		mcp.WithString("page_token",
			mcp.Description("Page token from previous response to continue pagination"),
		),
		mcp.WithString("order_by",
			mcp.Description("Sort order: createdTime, modifiedTime, name, name_natural (append ' desc' for descending)"),
		),
		mcp.WithString("modified_after",
			mcp.Description("Only include docs modified after this date (RFC3339 format)"),
		),
		mcp.WithString("modified_before",
			mcp.Description("Only include docs modified before this date (RFC3339 format)"),
		),
	)
}

// ListInFolderHandler handles docs_list_in_folder tool calls.
func (d *DocsTools) ListInFolderHandler(ctx context.Context, request mcp.CallToolRequest, args DocsListInFolderRequest) (*mcp.CallToolResult, error) {
	folderID := args.FolderID
	if folderID == "" {
		folderID = "root"
	}

	pageSize := args.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	// Build query: docs in folder, exclude trashed
	q := fmt.Sprintf("'%s' in parents and mimeType='application/vnd.google-apps.document' and trashed=false", folderID)

	// Add date filters
	if args.ModifiedAfter != "" {
		q += fmt.Sprintf(" and modifiedTime > '%s'", args.ModifiedAfter)
	}
	if args.ModifiedBefore != "" {
		q += fmt.Sprintf(" and modifiedTime < '%s'", args.ModifiedBefore)
	}

	call := d.driveService.Files.List().
		Context(ctx).
		Q(q).
		PageSize(int64(pageSize)).
		Fields("nextPageToken, files(id, name, createdTime, modifiedTime, webViewLink)").
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true)

	// Apply pagination
	if args.PageToken != "" {
		call = call.PageToken(args.PageToken)
	}
	// Apply sorting
	if args.OrderBy != "" {
		call = call.OrderBy(args.OrderBy)
	}

	fileList, err := call.Do()
	if err != nil {
		return mcp.NewToolResultError("failed to list documents: " + err.Error()), nil
	}

	results := make([]DocsSearchResult, 0, len(fileList.Files))
	for _, f := range fileList.Files {
		results = append(results, DocsSearchResult{
			ID:    f.Id,
			Title: f.Name,
		})
	}

	response := DocsSearchResponse{
		Results:       results,
		NextPageToken: fileList.NextPageToken,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// GetCommentsTool returns the tool definition for fetching document comments.
func (d *DocsTools) GetCommentsTool() mcp.Tool {
	return mcp.NewTool("docs_get_comments",
		mcp.WithDescription(`Retrieves comments and replies from a Google Doc.`),
		mcp.WithString("document_id",
			mcp.Required(),
			mcp.Description("The document ID"),
		),
		mcp.WithBoolean("include_resolved",
			mcp.Description("Include resolved comments (default false, only shows open comments)"),
		),
		mcp.WithString("page_token",
			mcp.Description("Page token from previous response to continue pagination"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Maximum number of comments per page (default 100)"),
			mcp.Min(1),
			mcp.Max(100),
		),
		mcp.WithString("modified_after",
			mcp.Description("Only include comments modified after this date (RFC3339 format)"),
		),
	)
}

// DocsComment represents a comment on a document.
type DocsComment struct {
	ID           string             `json:"id"`
	Author       string             `json:"author"`
	AuthorIsMe   bool               `json:"author_is_me"`
	Content      string             `json:"content"`
	QuotedText   string             `json:"quoted_text,omitempty"`
	CreatedTime  string             `json:"created_time"`
	ModifiedTime string             `json:"modified_time,omitempty"`
	Resolved     bool               `json:"resolved"`
	Replies      []DocsCommentReply `json:"replies,omitempty"`
}

// DocsCommentReply represents a reply to a comment.
type DocsCommentReply struct {
	ID          string `json:"id"`
	Author      string `json:"author"`
	AuthorIsMe  bool   `json:"author_is_me"`
	Content     string `json:"content"`
	CreatedTime string `json:"created_time"`
}

// DocsGetCommentsResponse contains the comments for a document.
type DocsGetCommentsResponse struct {
	DocumentID    string        `json:"document_id"`
	Comments      []DocsComment `json:"comments"`
	NextPageToken string        `json:"next_page_token,omitempty"`
}

// GetCommentsHandler handles docs_get_comments tool calls.
func (d *DocsTools) GetCommentsHandler(ctx context.Context, request mcp.CallToolRequest, args DocsGetCommentsRequest) (*mcp.CallToolResult, error) {
	if args.DocumentID == "" {
		return mcp.NewToolResultError("document_id is required"), nil
	}

	call := d.driveService.Comments.List(args.DocumentID).
		Context(ctx).
		Fields("nextPageToken, comments(id, author, content, quotedFileContent, createdTime, modifiedTime, resolved, replies)").
		IncludeDeleted(false)

	// Apply pagination
	if args.PageToken != "" {
		call = call.PageToken(args.PageToken)
	}
	// Apply page size
	if args.PageSize > 0 {
		call = call.PageSize(int64(args.PageSize))
	} else {
		call = call.PageSize(100)
	}
	// Apply modified after filter (API supports startModifiedTime)
	if args.ModifiedAfter != "" {
		call = call.StartModifiedTime(args.ModifiedAfter)
	}

	commentList, err := call.Do()
	if err != nil {
		return mcp.NewToolResultError("failed to get comments: " + err.Error()), nil
	}

	var comments []DocsComment
	for _, c := range commentList.Comments {
		// Skip resolved comments unless requested
		if c.Resolved && !args.IncludeResolved {
			continue
		}

		comment := DocsComment{
			ID:           c.Id,
			Content:      c.Content,
			CreatedTime:  c.CreatedTime,
			ModifiedTime: c.ModifiedTime,
			Resolved:     c.Resolved,
		}

		if c.Author != nil {
			comment.Author = c.Author.DisplayName
			comment.AuthorIsMe = c.Author.Me
		}

		if c.QuotedFileContent != nil {
			comment.QuotedText = c.QuotedFileContent.Value
		}

		for _, r := range c.Replies {
			reply := DocsCommentReply{
				ID:          r.Id,
				Content:     r.Content,
				CreatedTime: r.CreatedTime,
			}
			if r.Author != nil {
				reply.Author = r.Author.DisplayName
				reply.AuthorIsMe = r.Author.Me
			}
			comment.Replies = append(comment.Replies, reply)
		}

		comments = append(comments, comment)
	}

	response := DocsGetCommentsResponse{
		DocumentID:    args.DocumentID,
		Comments:      comments,
		NextPageToken: commentList.NextPageToken,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// MarshalCompact returns a compact text representation of the document content.
func (d DocsGetContentResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString("=== Document: ")
	sb.WriteString(d.DocTitle)
	sb.WriteString(" ===\nID: ")
	sb.WriteString(d.DocID)
	sb.WriteString("\n")

	for _, tab := range d.Tabs {
		sb.WriteString("\n--- Tab: ")
		sb.WriteString(tab.TabTitle)
		if tab.TabID != "" {
			sb.WriteString(" (id: ")
			sb.WriteString(tab.TabID)
			sb.WriteString(")")
		}
		sb.WriteString(" ---\n")
		sb.WriteString(tab.TabMarkdown)
		if !strings.HasSuffix(tab.TabMarkdown, "\n") {
			sb.WriteString("\n")
		}
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// MarshalCompact returns a compact text representation of the comments response.
func (d DocsGetCommentsResponse) MarshalCompact() string {
	var sb strings.Builder

	for i, c := range d.Comments {
		if i > 0 {
			sb.WriteString("\n")
		}
		// Comment header: "Comment <id> by <author> at <time> [resolved]"
		sb.WriteString("Comment ")
		sb.WriteString(c.ID)
		sb.WriteString(" by ")
		if c.AuthorIsMe {
			sb.WriteString("Me")
		} else {
			sb.WriteString(c.Author)
		}
		sb.WriteString(" at ")
		sb.WriteString(c.CreatedTime)
		if c.Resolved {
			sb.WriteString(" [resolved]")
		}
		sb.WriteString("\n")

		// Quoted text
		if c.QuotedText != "" {
			sb.WriteString("> ")
			sb.WriteString(strings.ReplaceAll(c.QuotedText, "\n", "\n> "))
			sb.WriteString("\n")
		}

		// Comment content
		sb.WriteString(c.Content)
		sb.WriteString("\n")

		// Replies
		for _, r := range c.Replies {
			sb.WriteString("  Reply ")
			sb.WriteString(r.ID)
			sb.WriteString(" by ")
			if r.AuthorIsMe {
				sb.WriteString("Me")
			} else {
				sb.WriteString(r.Author)
			}
			sb.WriteString(" at ")
			sb.WriteString(r.CreatedTime)
			sb.WriteString("\n  ")
			sb.WriteString(strings.ReplaceAll(r.Content, "\n", "\n  "))
			sb.WriteString("\n")
		}
	}

	if d.NextPageToken != "" {
		sb.WriteString("\nNext Page Token: ")
		sb.WriteString(d.NextPageToken)
	}

	return strings.TrimSuffix(sb.String(), "\n")
}
