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
- **types/**: Shared configuration and client types
  - `clients.go`: Google API client initialization with Application Default Credentials
  - `config.go`: Output format configuration (`MCP_OUTPUT_FORMAT` env var)
- **tools/**: MCP tool implementations, one file per Google service
  - `docs.go`: Google Docs tools (search, get content as markdown, list in folder, get comments)
  - `calendar.go`: Google Calendar tools (list calendars, get events)
  - `gmail.go`: Gmail tools (search, get message, get thread, list labels, get attachment)

Each tool file contains its own request structs, response types, and `MarshalCompact()` methods for compact output.

### Adding New Tools

1. Create a new `tools/<service>.go` following the pattern:
   - Define request structs (e.g., `<Service><Action>Request`) in the same file
   - Define response structs (e.g., `<Service><Action>Response`) with `MarshalCompact()` methods for compact output
   - Define a `<Service>Tools` struct holding the API client
   - Add a `New<Service>Tools(clients *types.<Service>Clients)` constructor
   - For each tool: `<Action>Tool()` returns `mcp.Tool`, `<Action>Handler()` handles calls
2. Add the client to `types/Clients` and create a `For<Service>()` accessor
3. Add the new scope to `RequiredScopes()` in `types/clients.go`
4. Register tools in `main.go`

### Authentication

Uses Google Application Default Credentials with read-only scopes. Authenticate with all required scopes:

```bash
gcloud auth application-default login --scopes="https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/calendar.readonly,https://www.googleapis.com/auth/documents.readonly,https://www.googleapis.com/auth/drive.readonly,https://www.googleapis.com/auth/gmail.readonly"
```

The `cloud-platform` scope is always required by gcloud. Service-specific scopes are defined in `types.RequiredScopes()`.

## Tool Definitions

Follow the read-only tool definitions as defined in https://github.com/taylorwilsdon/google_workspace_mcp for the Google APIs implemented in this project. To research the upstream project, clone it locally (`git clone https://github.com/taylorwilsdon/google_workspace_mcp /tmp/google_workspace_mcp`) and use local file exploration tools. When implementing a tool with the exact same inputs and outputs as the upstream project, copy the tool descriptions verbatim. The upstream project's LICENSE permits verbatim copying because this project meets its requirements. If adding or removing functionality that differs from the upstream project, adjust the description accordingly.

## Maintaining This File

Keep this file up to date as the project evolves. When making changes to the codebase, update any affected sections (build commands, architecture, authentication examples, etc.) to reflect the current state. For example, when adding new Google Workspace services, add their scopes to the authentication command above.
