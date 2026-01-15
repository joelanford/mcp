package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"

	"github.com/joelanford/mcp/google-workspace/types"
)

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

// GetContentTool returns the tool definition for fetching document content.
func (d *DocsTools) GetContentTool() mcp.Tool {
	return mcp.NewTool("docs_get_content",
		mcp.WithDescription(`Retrieves content of a Google Doc identified by document_id.

Returns:
    str: The document content with metadata header.`),
		mcp.WithString("document_id",
			mcp.Required(),
			mcp.Description("The document ID"),
		),
	)
}

// GetContentHandler handles docs_get_content tool calls.
func (d *DocsTools) GetContentHandler(ctx context.Context, request mcp.CallToolRequest, args types.DocsGetContentArgs) (*mcp.CallToolResult, error) {
	if args.DocumentID == "" {
		return mcp.NewToolResultError("document_id is required"), nil
	}

	doc, err := d.docsService.Documents.Get(args.DocumentID).Context(ctx).Do()
	if err != nil {
		return mcp.NewToolResultError("failed to get document: " + err.Error()), nil
	}

	// Extract text content from the document
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s\n\n", doc.Title))

	// Process document body
	if doc.Body != nil {
		extractContent(&content, doc.Body.Content)
	}

	return mcp.NewToolResultText(content.String()), nil
}

// extractContent extracts text from document structural elements.
func extractContent(sb *strings.Builder, elements []*docs.StructuralElement) {
	for _, elem := range elements {
		if elem.Paragraph != nil {
			for _, e := range elem.Paragraph.Elements {
				if e.TextRun != nil {
					sb.WriteString(e.TextRun.Content)
				}
			}
		}
		if elem.Table != nil {
			for _, row := range elem.Table.TableRows {
				for i, cell := range row.TableCells {
					if i > 0 {
						sb.WriteString("\t")
					}
					extractContent(sb, cell.Content)
				}
				sb.WriteString("\n")
			}
		}
		if elem.SectionBreak != nil {
			sb.WriteString("\n---\n")
		}
	}
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
