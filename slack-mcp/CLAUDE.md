# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run

```bash
make build                # Build binary to ./bin/slack-mcp
make clean                # Remove ./bin directory
./bin/slack-mcp          # Run MCP server over stdio (used by MCP clients)
```

## Architecture

This is an MCP (Model Context Protocol) server that provides read-only access to Slack workspaces. It uses the `mcp-go` library and communicates over stdio.

### Package Structure

- **main.go**: Server initialization, registers all tools with the MCP server
- **types/**: Shared configuration and client types
  - `clients.go`: Slack API client initialization using bot token
  - `config.go`: Output format configuration (`MCP_OUTPUT_FORMAT` env var)
- **tools/**: MCP tool implementations
  - `slack.go`: All 12 Slack tools for channels, messages, users, and files

The entire tool implementation is in a single file (~1100 lines) since all tools share the same Slack API client.

### Adding New Tools

To add a new Slack tool:

1. **Define Request/Response Structs** in `tools/slack.go`:
   ```go
   type Slack<Action>Request struct {
       Param1 string `json:"param1"`
       Param2 int    `json:"param2"`
   }

   type Slack<Action>Response struct {
       Data []Item `json:"data"`
   }

   // Add MarshalCompact for human-readable output
   func (r Slack<Action>Response) MarshalCompact() string {
       var sb strings.Builder
       // Build compact text representation
       return sb.String()
   }
   ```

2. **Add Tool Definition Method**:
   ```go
   func (t *SlackTools) <Action>Tool() mcp.Tool {
       return mcp.NewTool("slack_<action>",
           mcp.WithDescription("Tool description..."),
           mcp.WithString("param1",
               mcp.Required(),
               mcp.Description("Parameter description")),
       )
   }
   ```

3. **Add Handler Method**:
   ```go
   func (t *SlackTools) <Action>Handler(
       ctx context.Context,
       request mcp.CallToolRequest,
       args Slack<Action>Request,
   ) (*mcp.CallToolResult, error) {
       // Validate inputs
       // Call Slack API via t.api
       // Build response
       data, err := types.MarshalResponse(response)
       if err != nil {
           return mcp.NewToolResultError("failed to marshal: " + err.Error()), nil
       }
       return mcp.NewToolResultText(data), nil
   }
   ```

4. **Register Tool in main.go**:
   ```go
   s.AddTool(slackTools.<Action>Tool(),
       mcp.NewTypedToolHandler(slackTools.<Action>Handler))
   ```

### Authentication

Uses Slack bot tokens via the `SLACK_BOT_TOKEN` environment variable. Unlike OAuth2, this is a simpler token-based authentication:

```bash
export SLACK_BOT_TOKEN="xoxb-your-token-here"
```

**Required Bot Token Scopes:**
- `channels:read` - List/read public channels
- `groups:read` - List/read private channels
- `channels:history` - Read messages in public channels
- `groups:history` - Read messages in private channels
- `users:read` - List/read user profiles
- `search:read` - Search messages and files
- `files:read` - Read file information

To create a bot token:
1. Visit https://api.slack.com/apps
2. Create a new app or select existing
3. Go to "OAuth & Permissions"
4. Add the required Bot Token Scopes
5. Install app to workspace
6. Copy the "Bot User OAuth Token"

### Available Tools

**Channel Tools (3):**
- `slack_list_channels` - List all channels in workspace
- `slack_search_channels` - Search channels by name/keyword
- `slack_get_channel_info` - Get detailed channel information

**Message Tools (3):**
- `slack_search_messages` - Search messages across workspace
- `slack_get_channel_history` - Get message history from a channel
- `slack_get_thread_replies` - Get replies to a message thread

**User Tools (3):**
- `slack_list_users` - List all workspace members
- `slack_search_users` - Search users by name/email
- `slack_get_user_profile` - Get detailed user profile

**File Tools (3):**
- `slack_list_files` - List files in workspace
- `slack_search_files` - Search files by name/type
- `slack_get_file_info` - Get file metadata and download URL

### Output Format

Supports two output modes controlled by `MCP_OUTPUT_FORMAT` environment variable:

- **compact** (default): Human-readable text using `MarshalCompact()` methods
- **json**: Standard JSON output

All response types implement the `CompactMarshaler` interface for token-efficient Claude conversations.

### Error Handling

- All errors returned as `mcp.NewToolResultError(msg)`
- Helpful error messages guide users to fix configuration issues
- No panics - graceful degradation

### Code Patterns

**Consistent Tool Pattern:**
1. Request struct with JSON tags
2. Response struct with `MarshalCompact()` method
3. Tool definition with `mcp.NewTool()`
4. Handler with validation, API call, response formatting

**Default Values:**
Applied in handlers, not structs:
```go
limit := args.Limit
if limit == 0 {
    limit = 100  // default
}
```

**API Error Wrapping:**
```go
if err != nil {
    return mcp.NewToolResultError("failed to <action>: " + err.Error()), nil
}
```

## Dependencies

- **github.com/mark3labs/mcp-go** - MCP server framework
- **github.com/slack-go/slack** - Official Slack Go SDK

## Testing

Manual testing checklist:
1. Set `SLACK_BOT_TOKEN` environment variable
2. Build with `make build`
3. Test via Claude Desktop MCP integration
4. Verify each tool category works:
   - List and search channels
   - Search messages and get history
   - List and search users
   - List and search files
5. Test both compact and JSON output formats
6. Verify error handling with invalid inputs

## Maintaining This File

Keep this file up to date as the project evolves. When making changes to the codebase, update any affected sections (build commands, architecture, authentication examples, available tools, etc.) to reflect the current state.
