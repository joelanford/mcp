package types

import (
	"context"
	"fmt"
	"os"

	"github.com/slack-go/slack"
)

// Clients holds the Slack API client.
// The client is initialized once and shared across all tools.
type Clients struct {
	api *slack.Client
}

// NewClients creates a Slack API client using a bot token from the environment.
// Returns an error with helpful guidance if SLACK_BOT_TOKEN is not set.
func NewClients(ctx context.Context) (*Clients, error) {
	token := os.Getenv("SLACK_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("Slack bot token not found.\n\n" +
			"Set the SLACK_BOT_TOKEN environment variable with your Slack bot token.\n" +
			"Get your token from https://api.slack.com/apps (create an app if needed).\n\n" +
			"Required scopes:\n" +
			"  - channels:read, groups:read\n" +
			"  - channels:history, groups:history\n" +
			"  - users:read\n" +
			"  - search:read\n" +
			"  - files:read")
	}

	return &Clients{
		api: slack.New(token),
	}, nil
}

// SlackClients provides access to the Slack API client for tools.
type SlackClients struct {
	API *slack.Client
}

// ForSlack returns the client scoped for Slack tools.
func (c *Clients) ForSlack() *SlackClients {
	return &SlackClients{
		API: c.api,
	}
}
