package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"

	"github.com/joelanford/mcp/google-workspace/types"
)

// multipleNewlinesRe matches 3 or more consecutive newlines.
var multipleNewlinesRe = regexp.MustCompile(`\n{3,}`)

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
	)
}

// SearchHandler handles docs_search tool calls.
func (d *DocsTools) SearchHandler(ctx context.Context, request mcp.CallToolRequest, args types.DocsSearchArgs) (*mcp.CallToolResult, error) {
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

	fileList, err := d.driveService.Files.List().
		Context(ctx).
		Q(q).
		PageSize(int64(pageSize)).
		Fields("files(id, name, createdTime, modifiedTime, webViewLink)").
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		Do()
	if err != nil {
		return mcp.NewToolResultError("failed to search documents: " + err.Error()), nil
	}

	results := make([]types.SearchResult, 0, len(fileList.Files))
	for _, f := range fileList.Files {
		results = append(results, types.SearchResult{
			ID:    f.Id,
			Title: f.Name,
		})
	}

	response := types.SearchResponse{
		Results:       results,
		NextPageToken: fileList.NextPageToken,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// DocContentResponse represents the structured response for document content.
type DocContentResponse struct {
	DocID    string       `json:"docId"`
	DocTitle string       `json:"docTitle"`
	Tabs     []TabContent `json:"tabs"`
}

// TabContent represents a single tab's content.
type TabContent struct {
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
func (d *DocsTools) GetContentHandler(ctx context.Context, request mcp.CallToolRequest, args types.DocsGetContentArgs) (*mcp.CallToolResult, error) {
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
	response := DocContentResponse{
		DocID:    args.DocumentID,
		DocTitle: doc.Title,
		Tabs:     []TabContent{},
	}

	// Process all tabs (with recursive child tab support)
	if len(doc.Tabs) > 0 {
		response.Tabs = collectAllTabs(doc.Tabs, doc.Title)
	} else if doc.Body != nil {
		// Fallback for legacy single-tab documents
		var content strings.Builder
		extractMarkdownContent(&content, doc.Body.Content, nil, 0)
		response.Tabs = append(response.Tabs, TabContent{
			TabID:       "",
			TabTitle:    doc.Title,
			TabMarkdown: normalizeNewlines(content.String()),
		})
	}

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// normalizeNewlines collapses runs of 3+ newlines down to 2 (one blank line).
func normalizeNewlines(s string) string {
	return multipleNewlinesRe.ReplaceAllString(s, "\n\n")
}

// collectAllTabs recursively collects all tabs and their children into TabContent slices.
func collectAllTabs(tabs []*docs.Tab, docTitle string) []TabContent {
	var result []TabContent

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

			result = append(result, TabContent{
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
		adjustedLevel := headingLevel + headingOffset
		if adjustedLevel > 6 {
			adjustedLevel = 6
		}
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
	text = strings.ReplaceAll(text, "\u2018", "'") // left single quote
	text = strings.ReplaceAll(text, "\u2019", "'") // right single quote / apostrophe
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
	)
}

// ListInFolderHandler handles docs_list_in_folder tool calls.
func (d *DocsTools) ListInFolderHandler(ctx context.Context, request mcp.CallToolRequest, args types.DocsListInFolderArgs) (*mcp.CallToolResult, error) {
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

	fileList, err := d.driveService.Files.List().
		Context(ctx).
		Q(q).
		PageSize(int64(pageSize)).
		Fields("files(id, name, createdTime, modifiedTime, webViewLink)").
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		Do()
	if err != nil {
		return mcp.NewToolResultError("failed to list documents: " + err.Error()), nil
	}

	results := make([]types.SearchResult, 0, len(fileList.Files))
	for _, f := range fileList.Files {
		results = append(results, types.SearchResult{
			ID:    f.Id,
			Title: f.Name,
		})
	}

	response := types.SearchResponse{
		Results:       results,
		NextPageToken: fileList.NextPageToken,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// GetCommentsTool returns the tool definition for fetching document comments.
func (d *DocsTools) GetCommentsTool() mcp.Tool {
	return mcp.NewTool("docs_get_comments",
		mcp.WithDescription("Retrieves comments and replies from a Google Doc"),
		mcp.WithString("document_id",
			mcp.Required(),
			mcp.Description("The document ID"),
		),
		mcp.WithBoolean("include_resolved",
			mcp.Description("Include resolved comments (default false, only shows open comments)"),
		),
	)
}

// DocComment represents a comment on a document.
type DocComment struct {
	ID           string         `json:"id"`
	Author       string         `json:"author"`
	AuthorEmail  string         `json:"author_email,omitempty"`
	Content      string         `json:"content"`
	QuotedText   string         `json:"quoted_text,omitempty"`
	CreatedTime  string         `json:"created_time"`
	ModifiedTime string         `json:"modified_time,omitempty"`
	Resolved     bool           `json:"resolved"`
	Replies      []CommentReply `json:"replies,omitempty"`
}

// CommentReply represents a reply to a comment.
type CommentReply struct {
	ID          string `json:"id"`
	Author      string `json:"author"`
	AuthorEmail string `json:"author_email,omitempty"`
	Content     string `json:"content"`
	CreatedTime string `json:"created_time"`
}

// DocCommentsResponse contains the comments for a document.
type DocCommentsResponse struct {
	DocumentID string       `json:"document_id"`
	Comments   []DocComment `json:"comments"`
}

// GetCommentsHandler handles docs_get_comments tool calls.
func (d *DocsTools) GetCommentsHandler(ctx context.Context, request mcp.CallToolRequest, args types.DocsGetCommentsArgs) (*mcp.CallToolResult, error) {
	if args.DocumentID == "" {
		return mcp.NewToolResultError("document_id is required"), nil
	}

	call := d.driveService.Comments.List(args.DocumentID).
		Context(ctx).
		Fields("comments(id, author, content, quotedFileContent, createdTime, modifiedTime, resolved, replies)").
		IncludeDeleted(false)

	commentList, err := call.Do()
	if err != nil {
		return mcp.NewToolResultError("failed to get comments: " + err.Error()), nil
	}

	var comments []DocComment
	for _, c := range commentList.Comments {
		// Skip resolved comments unless requested
		if c.Resolved && !args.IncludeResolved {
			continue
		}

		comment := DocComment{
			ID:           c.Id,
			Content:      c.Content,
			CreatedTime:  c.CreatedTime,
			ModifiedTime: c.ModifiedTime,
			Resolved:     c.Resolved,
		}

		if c.Author != nil {
			comment.Author = c.Author.DisplayName
			comment.AuthorEmail = c.Author.EmailAddress
		}

		if c.QuotedFileContent != nil {
			comment.QuotedText = c.QuotedFileContent.Value
		}

		for _, r := range c.Replies {
			reply := CommentReply{
				ID:          r.Id,
				Content:     r.Content,
				CreatedTime: r.CreatedTime,
			}
			if r.Author != nil {
				reply.Author = r.Author.DisplayName
				reply.AuthorEmail = r.Author.EmailAddress
			}
			comment.Replies = append(comment.Replies, reply)
		}

		comments = append(comments, comment)
	}

	response := DocCommentsResponse{
		DocumentID: args.DocumentID,
		Comments:   comments,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
