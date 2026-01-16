# Google Workspace MCP Server

An MCP (Model Context Protocol) server that provides read-only access to Google Workspace APIs. Built with the [mcp-go](https://github.com/mark3labs/mcp-go) library and communicates over stdio.

## Features

- **Google Docs**: Search documents, get content as markdown, list documents in folders, get comments
- **Google Calendar**: List calendars, get events with attendees and attachments
- **Gmail**: Search messages, get message content, get threads, list labels, download attachments

## Installation

```bash
go install github.com/joelanford/mcp/google-workspace@latest
```

Or build from source:

```bash
make build
./bin/google-workspace-mcp
```

## Authentication

This server uses Google Application Default Credentials. Authenticate with all required scopes:

```bash
gcloud auth application-default login --scopes="https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/calendar.readonly,https://www.googleapis.com/auth/documents.readonly,https://www.googleapis.com/auth/drive.readonly,https://www.googleapis.com/auth/gmail.readonly"
```

## Configuration

### Output Format

By default, the server returns compact text output to reduce token usage. Set the `MCP_OUTPUT_FORMAT` environment variable to change the format:

- `compact` (default): Human-readable text format
- `json`: Full JSON output

```bash
MCP_OUTPUT_FORMAT=json ./bin/google-workspace-mcp
```

## Available Tools

### Google Docs

| Tool | Description |
|------|-------------|
| `docs_search` | Search for Google Docs by name |
| `docs_get_content` | Get document content as markdown (supports multi-tab documents) |
| `docs_list_in_folder` | List Google Docs in a specific folder |
| `docs_get_comments` | Get comments and replies from a document |

### Google Calendar

| Tool | Description |
|------|-------------|
| `calendar_list` | List all accessible calendars |
| `calendar_get_events` | Get events from a calendar (supports time ranges, search, single event lookup) |

### Gmail

| Tool | Description |
|------|-------------|
| `gmail_search` | Search for emails using Gmail query syntax |
| `gmail_get_message` | Get full message content including headers, body, and attachments |
| `gmail_get_thread` | Get all messages in a thread |
| `gmail_list_labels` | List all Gmail labels (system and user-created) |
| `gmail_get_attachment` | Download an attachment by ID |

## Usage with Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "google-workspace": {
      "command": "/path/to/google-workspace-mcp"
    }
  }
}
```

## Project Structure

```
.
├── main.go              # Server initialization, tool registration
├── types/
│   ├── clients.go       # Google API client initialization
│   └── config.go        # Output format configuration
└── tools/
    ├── docs.go          # Google Docs tools
    ├── calendar.go      # Google Calendar tools
    └── gmail.go         # Gmail tools
```

## License

See LICENSE file.
