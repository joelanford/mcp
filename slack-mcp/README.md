# Slack MCP Server

A Model Context Protocol (MCP) server that provides read-only access to Slack workspaces. This server enables AI assistants like Claude to interact with Slack channels, messages, users, and files.

## Features

### Channel Tools
- **slack_list_channels**: List all channels in the workspace
- **slack_search_channels**: Search for channels by name or keyword
- **slack_get_channel_info**: Get detailed information about a specific channel

### Message Tools
- **slack_search_messages**: Search for messages across the workspace
- **slack_get_channel_history**: Retrieve message history from a channel
- **slack_get_thread_replies**: Get all replies in a message thread

### User Tools
- **slack_list_users**: List all workspace members
- **slack_search_users**: Search for users by name or email
- **slack_get_user_profile**: Get detailed profile information for a user

### File Tools
- **slack_list_files**: List files in the workspace
- **slack_search_files**: Search for files by name or content
- **slack_get_file_info**: Get detailed metadata for a specific file

## Installation

### Prerequisites
- Go 1.24.6 or later
- A Slack workspace with admin access to create apps
- A Slack bot token with appropriate scopes

### Build from Source

```bash
# Clone the repository
git clone https://github.com/oceanc80/mcp.git
cd mcp/slack-mcp

# Build the binary
make build

# Or install to $GOPATH/bin
make install
```

The built binary will be available at `./bin/slack-mcp`.

## Authentication Setup

### 1. Create a Slack App

1. Go to [https://api.slack.com/apps](https://api.slack.com/apps)
2. Click "Create New App" → "From scratch"
3. Give your app a name (e.g., "MCP Server") and select your workspace
4. Click "Create App"

### 2. Configure OAuth Scopes

Navigate to "OAuth & Permissions" in the sidebar and add these Bot Token Scopes:

**Required scopes:**
- `channels:read` - View public channels
- `groups:read` - View private channels
- `channels:history` - View messages in public channels
- `groups:history` - View messages in private channels
- `users:read` - View workspace users
- `search:read` - Search messages and files
- `files:read` - View files

### 3. Install the App

1. Click "Install to Workspace" at the top of the OAuth & Permissions page
2. Review the permissions and click "Allow"
3. Copy the "Bot User OAuth Token" (starts with `xoxb-`)

### 4. Set Environment Variable

```bash
export SLACK_BOT_TOKEN="xoxb-your-token-here"
```

For permanent configuration, add this to your `~/.bashrc`, `~/.zshrc`, or equivalent:

```bash
echo 'export SLACK_BOT_TOKEN="xoxb-your-token-here"' >> ~/.bashrc
source ~/.bashrc
```

## Configuration

### Output Format

The server supports two output formats:

- **Compact** (default): Human-readable text format optimized for Claude conversations
- **JSON**: Standard JSON format for programmatic access

Set the output format using the `MCP_OUTPUT_FORMAT` environment variable:

```bash
export MCP_OUTPUT_FORMAT="compact"  # or "json"
```

## Usage with Claude Desktop

Add the following to your Claude Desktop MCP configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "slack": {
      "command": "/path/to/slack-mcp/bin/slack-mcp",
      "env": {
        "SLACK_BOT_TOKEN": "xoxb-your-token-here"
      }
    }
  }
}
```

Replace `/path/to/slack-mcp/bin/slack-mcp` with the actual path to your built binary.

After updating the configuration, restart Claude Desktop.

## Usage Examples

Once configured, you can ask Claude to interact with your Slack workspace:

- "List all channels in my Slack workspace"
- "Search for messages about the product launch"
- "Show me the conversation history in #general"
- "Find users with 'engineering' in their name"
- "List all PDF files shared in the workspace"

## Project Structure

```
slack-mcp/
├── main.go              # Server entry point and tool registration
├── Makefile            # Build automation
├── README.md           # User documentation (this file)
├── CLAUDE.md           # Developer documentation
├── go.mod              # Go module definition
├── types/
│   ├── clients.go      # Slack API client initialization
│   └── config.go       # Output format configuration
└── tools/
    └── slack.go        # All Slack tool implementations
```

## Development

### Running Locally

```bash
# Build the server
make build

# Run the server (communicates over stdio)
./bin/slack-mcp
```

The server uses stdio for communication with MCP clients, so it won't produce output when run directly. Use it through an MCP client like Claude Desktop.

### Adding New Tools

See [CLAUDE.md](CLAUDE.md) for developer documentation on adding new tools.

## Troubleshooting

### "Slack bot token not found" Error

Make sure you've set the `SLACK_BOT_TOKEN` environment variable correctly:

```bash
echo $SLACK_BOT_TOKEN  # Should display your token
```

### Permission Errors

Verify that your Slack app has all the required scopes listed in the Authentication Setup section. You may need to reinstall the app after adding scopes.

### Messages Not Appearing

The bot can only see messages in:
- Public channels
- Private channels where the bot has been explicitly invited

To invite the bot to a private channel, use `/invite @your-bot-name` in the channel.

## Security Considerations

- **Read-Only Access**: This server only reads data from Slack and cannot post messages, create channels, or modify any data
- **Token Security**: Keep your `SLACK_BOT_TOKEN` secure and never commit it to version control
- **Scope Minimization**: The server requests only read scopes necessary for its functionality

## License

See [LICENSE](../LICENSE) file in the repository root.

## Support

For issues and questions:
- Report bugs at [https://github.com/oceanc80/mcp/issues](https://github.com/oceanc80/mcp/issues)
- Review the developer documentation in [CLAUDE.md](CLAUDE.md)

## Acknowledgments

This server follows the patterns established in the [google-workspace-mcp](../google-workspace-mcp) server and uses the [mcp-go](https://github.com/mark3labs/mcp-go) framework.
