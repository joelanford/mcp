package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/oceanc80/mcp/slack-mcp/tools"
	"github.com/oceanc80/mcp/slack-mcp/types"
)

func main() {
	ctx := context.Background()

	// Initialize Slack API client
	clients, err := types.NewClients(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	s := server.NewMCPServer(
		"Slack MCP Server",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	// Register Slack tools
	slackTools := tools.NewSlackTools(clients.ForSlack())

	// Channel tools
	s.AddTool(slackTools.ListChannelsTool(), mcp.NewTypedToolHandler(slackTools.ListChannelsHandler))
	s.AddTool(slackTools.SearchChannelsTool(), mcp.NewTypedToolHandler(slackTools.SearchChannelsHandler))
	s.AddTool(slackTools.GetChannelInfoTool(), mcp.NewTypedToolHandler(slackTools.GetChannelInfoHandler))

	// Message tools
	s.AddTool(slackTools.SearchMessagesTool(), mcp.NewTypedToolHandler(slackTools.SearchMessagesHandler))
	s.AddTool(slackTools.GetChannelHistoryTool(), mcp.NewTypedToolHandler(slackTools.GetChannelHistoryHandler))
	s.AddTool(slackTools.GetThreadRepliesTool(), mcp.NewTypedToolHandler(slackTools.GetThreadRepliesHandler))

	// User tools
	s.AddTool(slackTools.ListUsersTool(), mcp.NewTypedToolHandler(slackTools.ListUsersHandler))
	s.AddTool(slackTools.SearchUsersTool(), mcp.NewTypedToolHandler(slackTools.SearchUsersHandler))
	s.AddTool(slackTools.GetUserProfileTool(), mcp.NewTypedToolHandler(slackTools.GetUserProfileHandler))

	// File tools
	s.AddTool(slackTools.ListFilesTool(), mcp.NewTypedToolHandler(slackTools.ListFilesHandler))
	s.AddTool(slackTools.SearchFilesTool(), mcp.NewTypedToolHandler(slackTools.SearchFilesHandler))
	s.AddTool(slackTools.GetFileInfoTool(), mcp.NewTypedToolHandler(slackTools.GetFileInfoHandler))

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
