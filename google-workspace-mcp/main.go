package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/joelanford/mcp/google-workspace/tools"
	"github.com/joelanford/mcp/google-workspace/types"
)

func main() {
	ctx := context.Background()

	// Initialize all Google API clients
	clients, err := types.NewClients(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	s := server.NewMCPServer(
		"Google Workspace MCP Server",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	// Register Docs tools
	docsTools := tools.NewDocsTools(clients.ForDocs())
	s.AddTool(docsTools.SearchTool(), mcp.NewTypedToolHandler(docsTools.SearchHandler))
	s.AddTool(docsTools.GetContentTool(), mcp.NewTypedToolHandler(docsTools.GetContentHandler))
	s.AddTool(docsTools.GetCommentsTool(), mcp.NewTypedToolHandler(docsTools.GetCommentsHandler))
	s.AddTool(docsTools.ListInFolderTool(), mcp.NewTypedToolHandler(docsTools.ListInFolderHandler))

	// Register Calendar tools
	calendarTools := tools.NewCalendarTools(clients.ForCalendar())
	s.AddTool(calendarTools.ListCalendarsTool(), mcp.NewTypedToolHandler(calendarTools.ListCalendarsHandler))
	s.AddTool(calendarTools.GetEventsTool(), mcp.NewTypedToolHandler(calendarTools.GetEventsHandler))

	// TODO: Implement additional Google Workspace tools:
	// - Gmail
	// - Sheets
	// - Slides
	// - Tasks

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
