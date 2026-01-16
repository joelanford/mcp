# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run

```bash
make build                  # Build binary to ./bin/google-workspace-mcp
make clean                  # Remove ./bin directory
./bin/google-workspace-mcp  # Run MCP server over stdio (used by MCP clients)
```

## Architecture

This is an MCP (Model Context Protocol) server that provides read-only access to Google Workspace APIs. It uses the `mcp-go` library and communicates over stdio.

### Package Structure

- **main.go**: Server initialization, registers all tools with the MCP server
- **types/**: Shared types
  - `clients.go`: Google API client initialization with Application Default Credentials
  - `types.go`: Request argument structs for tool handlers
- **tools/**: MCP tool implementations, one file per Google service
  - `docs.go`: Google Docs tools (search, get content as markdown, list in folder, get comments)
  - `calendar.go`: Google Calendar tools (list calendars, get events)

### Adding New Tools

1. Add argument structs to `types/types.go`
2. Create a new `tools/<service>.go` following the pattern:
   - Define a `<Service>Tools` struct holding the API client
   - Add a `New<Service>Tools(clients *types.<Service>Clients)` constructor
   - For each tool: `<Action>Tool()` returns `mcp.Tool`, `<Action>Handler()` handles calls
3. Add the client to `types/Clients` and create a `For<Service>()` accessor
4. Add the new scope to `RequiredScopes()` in `types/clients.go`
5. Register tools in `main.go`

### Authentication

Uses Google Application Default Credentials with read-only scopes. Authenticate with all required scopes:

```bash
gcloud auth application-default login --scopes="https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/calendar.readonly,https://www.googleapis.com/auth/documents.readonly,https://www.googleapis.com/auth/drive.readonly"
```

The `cloud-platform` scope is always required by gcloud. Service-specific scopes are defined in `types.RequiredScopes()`.

## Maintaining This File

Keep this file up to date as the project evolves. When making changes to the codebase, update any affected sections (build commands, architecture, authentication examples, etc.) to reflect the current state. For example, when adding new Google Workspace services, add their scopes to the authentication command above.
