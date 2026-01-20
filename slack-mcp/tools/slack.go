package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/slack-go/slack"

	"github.com/oceanc80/mcp/slack-mcp/types"
)

// ========== Request/Response Structs ==========

// Channel tools requests
type SlackListChannelsRequest struct {
	Limit           int    `json:"limit"`
	ExcludeArchived bool   `json:"exclude_archived"`
	Types           string `json:"types"`
}

type SlackSearchChannelsRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type SlackGetChannelInfoRequest struct {
	ChannelID string `json:"channel_id"`
}

// Message tools requests
type SlackSearchMessagesRequest struct {
	Query string `json:"query"`
	Count int    `json:"count"`
	Sort  string `json:"sort"`
}

type SlackGetChannelHistoryRequest struct {
	ChannelID string `json:"channel_id"`
	Limit     int    `json:"limit"`
	Oldest    string `json:"oldest"`
	Latest    string `json:"latest"`
}

type SlackGetThreadRepliesRequest struct {
	ChannelID string `json:"channel_id"`
	ThreadTS  string `json:"thread_ts"`
}

// User tools requests
type SlackListUsersRequest struct {
	Limit       int  `json:"limit"`
	IncludeBots bool `json:"include_bots"`
}

type SlackSearchUsersRequest struct {
	Query string `json:"query"`
}

type SlackGetUserProfileRequest struct {
	UserID string `json:"user_id"`
}

// File tools requests
type SlackListFilesRequest struct {
	Count   int    `json:"count"`
	Types   string `json:"types"`
	Channel string `json:"channel"`
}

type SlackSearchFilesRequest struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

type SlackGetFileInfoRequest struct {
	FileID string `json:"file_id"`
}

// ========== Response Structs ==========

// Channel responses
type ChannelInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsPrivate  bool   `json:"is_private"`
	IsMember   bool   `json:"is_member"`
	NumMembers int    `json:"num_members"`
	Topic      string `json:"topic,omitempty"`
	Purpose    string `json:"purpose,omitempty"`
}

type SlackListChannelsResponse struct {
	Channels []ChannelInfo `json:"channels"`
}

func (r SlackListChannelsResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Channels (%d):\n", len(r.Channels)))
	for _, ch := range r.Channels {
		sb.WriteString("  ")
		if ch.IsPrivate {
			sb.WriteString("🔒 ")
		} else {
			sb.WriteString("# ")
		}
		sb.WriteString(ch.Name)
		sb.WriteString(" (")
		sb.WriteString(ch.ID)
		sb.WriteString(")")
		if ch.IsMember {
			sb.WriteString(" [member]")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

type SlackSearchChannelsResponse struct {
	Channels []ChannelInfo `json:"channels"`
}

func (r SlackSearchChannelsResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d channels:\n", len(r.Channels)))
	for _, ch := range r.Channels {
		sb.WriteString("  # ")
		sb.WriteString(ch.Name)
		sb.WriteString(" (")
		sb.WriteString(ch.ID)
		sb.WriteString(")")
		if ch.Topic != "" {
			sb.WriteString(" - ")
			sb.WriteString(ch.Topic)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

type SlackGetChannelInfoResponse struct {
	Channel ChannelInfo `json:"channel"`
}

func (r SlackGetChannelInfoResponse) MarshalCompact() string {
	var sb strings.Builder
	ch := r.Channel
	sb.WriteString("Channel Info:\n")
	sb.WriteString("  Name: ")
	if ch.IsPrivate {
		sb.WriteString("🔒 ")
	} else {
		sb.WriteString("# ")
	}
	sb.WriteString(ch.Name)
	sb.WriteString("\n")
	sb.WriteString("  ID: ")
	sb.WriteString(ch.ID)
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Members: %d\n", ch.NumMembers))
	if ch.Topic != "" {
		sb.WriteString("  Topic: ")
		sb.WriteString(ch.Topic)
		sb.WriteString("\n")
	}
	if ch.Purpose != "" {
		sb.WriteString("  Purpose: ")
		sb.WriteString(ch.Purpose)
		sb.WriteString("\n")
	}
	return sb.String()
}

// Message responses
type MessageInfo struct {
	Type      string `json:"type"`
	User      string `json:"user"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
	Channel   string `json:"channel,omitempty"`
}

type SlackSearchMessagesResponse struct {
	Messages []MessageInfo `json:"messages"`
	Total    int           `json:"total"`
}

func (r SlackSearchMessagesResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d messages:\n", r.Total))
	for i, msg := range r.Messages {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("  ... and %d more\n", r.Total-10))
			break
		}
		sb.WriteString("  ")
		sb.WriteString(msg.User)
		sb.WriteString(" at ")
		sb.WriteString(msg.Timestamp)
		sb.WriteString(": ")
		text := msg.Text
		if len(text) > 100 {
			text = text[:97] + "..."
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return sb.String()
}

type SlackGetChannelHistoryResponse struct {
	Messages []MessageInfo `json:"messages"`
	HasMore  bool          `json:"has_more"`
}

func (r SlackGetChannelHistoryResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Channel History (%d messages):\n", len(r.Messages)))
	for _, msg := range r.Messages {
		sb.WriteString("  [")
		sb.WriteString(msg.Timestamp)
		sb.WriteString("] ")
		sb.WriteString(msg.User)
		sb.WriteString(": ")
		sb.WriteString(msg.Text)
		sb.WriteString("\n")
	}
	if r.HasMore {
		sb.WriteString("  ... more messages available\n")
	}
	return sb.String()
}

type SlackGetThreadRepliesResponse struct {
	Messages []MessageInfo `json:"messages"`
}

func (r SlackGetThreadRepliesResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Thread (%d messages):\n", len(r.Messages)))
	for _, msg := range r.Messages {
		sb.WriteString("  [")
		sb.WriteString(msg.Timestamp)
		sb.WriteString("] ")
		sb.WriteString(msg.User)
		sb.WriteString(": ")
		sb.WriteString(msg.Text)
		sb.WriteString("\n")
	}
	return sb.String()
}

// User responses
type UserInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	Email    string `json:"email,omitempty"`
	IsBot    bool   `json:"is_bot"`
	IsAdmin  bool   `json:"is_admin,omitempty"`
}

type SlackListUsersResponse struct {
	Users []UserInfo `json:"users"`
}

func (r SlackListUsersResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Users (%d):\n", len(r.Users)))
	for _, u := range r.Users {
		sb.WriteString("  ")
		if u.IsBot {
			sb.WriteString("🤖 ")
		} else {
			sb.WriteString("👤 ")
		}
		sb.WriteString(u.Name)
		if u.RealName != "" && u.RealName != u.Name {
			sb.WriteString(" (")
			sb.WriteString(u.RealName)
			sb.WriteString(")")
		}
		sb.WriteString(" - ")
		sb.WriteString(u.ID)
		sb.WriteString("\n")
	}
	return sb.String()
}

type SlackSearchUsersResponse struct {
	Users []UserInfo `json:"users"`
}

func (r SlackSearchUsersResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d users:\n", len(r.Users)))
	for _, u := range r.Users {
		sb.WriteString("  ")
		sb.WriteString(u.Name)
		if u.RealName != "" {
			sb.WriteString(" (")
			sb.WriteString(u.RealName)
			sb.WriteString(")")
		}
		if u.Email != "" {
			sb.WriteString(" - ")
			sb.WriteString(u.Email)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

type SlackGetUserProfileResponse struct {
	User UserInfo `json:"user"`
}

func (r SlackGetUserProfileResponse) MarshalCompact() string {
	var sb strings.Builder
	u := r.User
	sb.WriteString("User Profile:\n")
	sb.WriteString("  Name: ")
	sb.WriteString(u.Name)
	sb.WriteString("\n")
	if u.RealName != "" {
		sb.WriteString("  Real Name: ")
		sb.WriteString(u.RealName)
		sb.WriteString("\n")
	}
	sb.WriteString("  ID: ")
	sb.WriteString(u.ID)
	sb.WriteString("\n")
	if u.Email != "" {
		sb.WriteString("  Email: ")
		sb.WriteString(u.Email)
		sb.WriteString("\n")
	}
	if u.IsBot {
		sb.WriteString("  Type: Bot\n")
	}
	if u.IsAdmin {
		sb.WriteString("  Admin: Yes\n")
	}
	return sb.String()
}

// File responses
type FileInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Title     string `json:"title,omitempty"`
	Mimetype  string `json:"mimetype"`
	Size      int    `json:"size"`
	URL       string `json:"url_private,omitempty"`
	User      string `json:"user"`
	Timestamp string `json:"timestamp"`
}

type SlackListFilesResponse struct {
	Files []FileInfo `json:"files"`
}

func (r SlackListFilesResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Files (%d):\n", len(r.Files)))
	for _, f := range r.Files {
		sb.WriteString("  📄 ")
		sb.WriteString(f.Name)
		if f.Title != "" && f.Title != f.Name {
			sb.WriteString(" (")
			sb.WriteString(f.Title)
			sb.WriteString(")")
		}
		sb.WriteString(fmt.Sprintf(" - %s", formatFileSize(f.Size)))
		sb.WriteString("\n")
	}
	return sb.String()
}

type SlackSearchFilesResponse struct {
	Files []FileInfo `json:"files"`
	Total int        `json:"total"`
}

func (r SlackSearchFilesResponse) MarshalCompact() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d files:\n", r.Total))
	for _, f := range r.Files {
		sb.WriteString("  📄 ")
		sb.WriteString(f.Name)
		sb.WriteString(" - ")
		sb.WriteString(f.Mimetype)
		sb.WriteString(fmt.Sprintf(" (%s)", formatFileSize(f.Size)))
		sb.WriteString("\n")
	}
	return sb.String()
}

type SlackGetFileInfoResponse struct {
	File FileInfo `json:"file"`
}

func (r SlackGetFileInfoResponse) MarshalCompact() string {
	var sb strings.Builder
	f := r.File
	sb.WriteString("File Info:\n")
	sb.WriteString("  Name: ")
	sb.WriteString(f.Name)
	sb.WriteString("\n")
	if f.Title != "" {
		sb.WriteString("  Title: ")
		sb.WriteString(f.Title)
		sb.WriteString("\n")
	}
	sb.WriteString("  ID: ")
	sb.WriteString(f.ID)
	sb.WriteString("\n")
	sb.WriteString("  Type: ")
	sb.WriteString(f.Mimetype)
	sb.WriteString("\n")
	sb.WriteString("  Size: ")
	sb.WriteString(formatFileSize(f.Size))
	sb.WriteString("\n")
	if f.URL != "" {
		sb.WriteString("  Download URL: ")
		sb.WriteString(f.URL)
		sb.WriteString("\n")
	}
	return sb.String()
}

// Helper function to format file sizes
func formatFileSize(size int) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}

// ========== SlackTools Struct ==========

// SlackTools provides Slack API tools.
type SlackTools struct {
	api *slack.Client
}

// NewSlackTools creates a new SlackTools instance from the provided clients.
func NewSlackTools(clients *types.SlackClients) *SlackTools {
	return &SlackTools{
		api: clients.API,
	}
}

// ========== Channel Tools ==========

// ListChannelsTool returns the tool definition for listing channels.
func (t *SlackTools) ListChannelsTool() mcp.Tool {
	return mcp.NewTool("slack_list_channels",
		mcp.WithDescription(`Lists channels in the Slack workspace.

Returns a list of channels with basic information including name, ID, privacy status, and member count.

Parameters:
- limit: Maximum number of channels to return (default: 100)
- exclude_archived: Exclude archived channels (default: true)
- types: Comma-separated list of channel types (public_channel, private_channel, mpim, im)`),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of channels to return (default: 100)")),
		mcp.WithBoolean("exclude_archived",
			mcp.Description("Exclude archived channels (default: true)")),
		mcp.WithString("types",
			mcp.Description("Comma-separated channel types: public_channel, private_channel, mpim, im")),
	)
}

// ListChannelsHandler handles slack_list_channels tool calls.
func (t *SlackTools) ListChannelsHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackListChannelsRequest,
) (*mcp.CallToolResult, error) {
	limit := args.Limit
	if limit == 0 {
		limit = 100
	}

	channelTypes := args.Types
	if channelTypes == "" {
		channelTypes = "public_channel,private_channel"
	}

	channels, _, err := t.api.GetConversationsContext(ctx, &slack.GetConversationsParameters{
		ExcludeArchived: args.ExcludeArchived,
		Limit:           limit,
		Types:           strings.Split(channelTypes, ","),
	})
	if err != nil {
		return mcp.NewToolResultError("failed to list channels: " + err.Error()), nil
	}

	response := SlackListChannelsResponse{
		Channels: make([]ChannelInfo, 0, len(channels)),
	}
	for _, ch := range channels {
		response.Channels = append(response.Channels, ChannelInfo{
			ID:         ch.ID,
			Name:       ch.Name,
			IsPrivate:  ch.IsPrivate,
			IsMember:   ch.IsMember,
			NumMembers: ch.NumMembers,
			Topic:      ch.Topic.Value,
			Purpose:    ch.Purpose.Value,
		})
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// SearchChannelsTool returns the tool definition for searching channels.
func (t *SlackTools) SearchChannelsTool() mcp.Tool {
	return mcp.NewTool("slack_search_channels",
		mcp.WithDescription(`Searches for channels by name or keyword.

Returns channels matching the search query.

Parameters:
- query: Search term to match against channel names
- limit: Maximum number of results to return (default: 20)`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search term for channel names")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results (default: 20)")),
	)
}

// SearchChannelsHandler handles slack_search_channels tool calls.
func (t *SlackTools) SearchChannelsHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackSearchChannelsRequest,
) (*mcp.CallToolResult, error) {
	if args.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	limit := args.Limit
	if limit == 0 {
		limit = 20
	}

	// Get all channels and filter by query
	channels, _, err := t.api.GetConversationsContext(ctx, &slack.GetConversationsParameters{
		ExcludeArchived: true,
		Limit:           1000,
		Types:           []string{"public_channel", "private_channel"},
	})
	if err != nil {
		return mcp.NewToolResultError("failed to search channels: " + err.Error()), nil
	}

	queryLower := strings.ToLower(args.Query)
	var matched []ChannelInfo
	for _, ch := range channels {
		if strings.Contains(strings.ToLower(ch.Name), queryLower) ||
			strings.Contains(strings.ToLower(ch.Topic.Value), queryLower) {
			matched = append(matched, ChannelInfo{
				ID:         ch.ID,
				Name:       ch.Name,
				IsPrivate:  ch.IsPrivate,
				IsMember:   ch.IsMember,
				NumMembers: ch.NumMembers,
				Topic:      ch.Topic.Value,
				Purpose:    ch.Purpose.Value,
			})
			if len(matched) >= limit {
				break
			}
		}
	}

	response := SlackSearchChannelsResponse{
		Channels: matched,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// GetChannelInfoTool returns the tool definition for getting channel info.
func (t *SlackTools) GetChannelInfoTool() mcp.Tool {
	return mcp.NewTool("slack_get_channel_info",
		mcp.WithDescription(`Gets detailed information about a specific channel.

Returns comprehensive channel details including topic, purpose, and member count.

Parameters:
- channel_id: The ID of the channel`),
		mcp.WithString("channel_id",
			mcp.Required(),
			mcp.Description("Channel ID (e.g., C1234567890)")),
	)
}

// GetChannelInfoHandler handles slack_get_channel_info tool calls.
func (t *SlackTools) GetChannelInfoHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackGetChannelInfoRequest,
) (*mcp.CallToolResult, error) {
	if args.ChannelID == "" {
		return mcp.NewToolResultError("channel_id parameter is required"), nil
	}

	channel, err := t.api.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
		ChannelID: args.ChannelID,
	})
	if err != nil {
		return mcp.NewToolResultError("failed to get channel info: " + err.Error()), nil
	}

	response := SlackGetChannelInfoResponse{
		Channel: ChannelInfo{
			ID:         channel.ID,
			Name:       channel.Name,
			IsPrivate:  channel.IsPrivate,
			IsMember:   channel.IsMember,
			NumMembers: channel.NumMembers,
			Topic:      channel.Topic.Value,
			Purpose:    channel.Purpose.Value,
		},
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// ========== Message Tools ==========

// SearchMessagesTool returns the tool definition for searching messages.
func (t *SlackTools) SearchMessagesTool() mcp.Tool {
	return mcp.NewTool("slack_search_messages",
		mcp.WithDescription(`Searches for messages across the Slack workspace.

Returns messages matching the search query.

Parameters:
- query: Search query string
- count: Number of results to return (default: 20, max: 100)
- sort: Sort order (score, timestamp) (default: score)`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query for messages")),
		mcp.WithNumber("count",
			mcp.Description("Number of results (default: 20, max: 100)")),
		mcp.WithString("sort",
			mcp.Description("Sort order: score or timestamp (default: score)")),
	)
}

// SearchMessagesHandler handles slack_search_messages tool calls.
func (t *SlackTools) SearchMessagesHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackSearchMessagesRequest,
) (*mcp.CallToolResult, error) {
	if args.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	count := args.Count
	if count == 0 {
		count = 20
	}
	if count > 100 {
		count = 100
	}

	sort := args.Sort
	if sort == "" {
		sort = "score"
	}

	searchParams := slack.NewSearchParameters()
	searchParams.Count = count
	searchParams.Sort = sort

	result, err := t.api.SearchMessagesContext(ctx, args.Query, searchParams)
	if err != nil {
		return mcp.NewToolResultError("failed to search messages: " + err.Error()), nil
	}

	messages := make([]MessageInfo, 0, len(result.Matches))
	for _, match := range result.Matches {
		messages = append(messages, MessageInfo{
			Type:      match.Type,
			User:      match.User,
			Text:      match.Text,
			Timestamp: match.Timestamp,
			Channel:   match.Channel.ID,
		})
	}

	response := SlackSearchMessagesResponse{
		Messages: messages,
		Total:    result.Total,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// GetChannelHistoryTool returns the tool definition for getting channel history.
func (t *SlackTools) GetChannelHistoryTool() mcp.Tool {
	return mcp.NewTool("slack_get_channel_history",
		mcp.WithDescription(`Retrieves message history from a channel.

Returns recent messages from the specified channel.

Parameters:
- channel_id: The ID of the channel
- limit: Maximum number of messages to return (default: 100)
- oldest: Only messages after this Unix timestamp
- latest: Only messages before this Unix timestamp`),
		mcp.WithString("channel_id",
			mcp.Required(),
			mcp.Description("Channel ID")),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of messages (default: 100)")),
		mcp.WithString("oldest",
			mcp.Description("Unix timestamp - only messages after this time")),
		mcp.WithString("latest",
			mcp.Description("Unix timestamp - only messages before this time")),
	)
}

// GetChannelHistoryHandler handles slack_get_channel_history tool calls.
func (t *SlackTools) GetChannelHistoryHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackGetChannelHistoryRequest,
) (*mcp.CallToolResult, error) {
	if args.ChannelID == "" {
		return mcp.NewToolResultError("channel_id parameter is required"), nil
	}

	limit := args.Limit
	if limit == 0 {
		limit = 100
	}

	params := &slack.GetConversationHistoryParameters{
		ChannelID: args.ChannelID,
		Limit:     limit,
	}
	if args.Oldest != "" {
		params.Oldest = args.Oldest
	}
	if args.Latest != "" {
		params.Latest = args.Latest
	}

	history, err := t.api.GetConversationHistoryContext(ctx, params)
	if err != nil {
		return mcp.NewToolResultError("failed to get channel history: " + err.Error()), nil
	}

	messages := make([]MessageInfo, 0, len(history.Messages))
	for _, msg := range history.Messages {
		messages = append(messages, MessageInfo{
			Type:      msg.Type,
			User:      msg.User,
			Text:      msg.Text,
			Timestamp: msg.Timestamp,
		})
	}

	response := SlackGetChannelHistoryResponse{
		Messages: messages,
		HasMore:  history.HasMore,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// GetThreadRepliesTool returns the tool definition for getting thread replies.
func (t *SlackTools) GetThreadRepliesTool() mcp.Tool {
	return mcp.NewTool("slack_get_thread_replies",
		mcp.WithDescription(`Retrieves all replies in a message thread.

Returns all messages in a thread, including the parent message.

Parameters:
- channel_id: The ID of the channel containing the thread
- thread_ts: The timestamp of the parent message`),
		mcp.WithString("channel_id",
			mcp.Required(),
			mcp.Description("Channel ID")),
		mcp.WithString("thread_ts",
			mcp.Required(),
			mcp.Description("Thread parent message timestamp")),
	)
}

// GetThreadRepliesHandler handles slack_get_thread_replies tool calls.
func (t *SlackTools) GetThreadRepliesHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackGetThreadRepliesRequest,
) (*mcp.CallToolResult, error) {
	if args.ChannelID == "" {
		return mcp.NewToolResultError("channel_id parameter is required"), nil
	}
	if args.ThreadTS == "" {
		return mcp.NewToolResultError("thread_ts parameter is required"), nil
	}

	msgs, _, _, err := t.api.GetConversationRepliesContext(ctx, &slack.GetConversationRepliesParameters{
		ChannelID: args.ChannelID,
		Timestamp: args.ThreadTS,
	})
	if err != nil {
		return mcp.NewToolResultError("failed to get thread replies: " + err.Error()), nil
	}

	messages := make([]MessageInfo, 0, len(msgs))
	for _, msg := range msgs {
		messages = append(messages, MessageInfo{
			Type:      msg.Type,
			User:      msg.User,
			Text:      msg.Text,
			Timestamp: msg.Timestamp,
		})
	}

	response := SlackGetThreadRepliesResponse{
		Messages: messages,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// ========== User Tools ==========

// ListUsersTool returns the tool definition for listing users.
func (t *SlackTools) ListUsersTool() mcp.Tool {
	return mcp.NewTool("slack_list_users",
		mcp.WithDescription(`Lists all users in the Slack workspace.

Returns a list of workspace members with basic profile information.

Parameters:
- limit: Maximum number of users to return (default: 100)
- include_bots: Include bot users in the results (default: false)`),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of users (default: 100)")),
		mcp.WithBoolean("include_bots",
			mcp.Description("Include bot users (default: false)")),
	)
}

// ListUsersHandler handles slack_list_users tool calls.
func (t *SlackTools) ListUsersHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackListUsersRequest,
) (*mcp.CallToolResult, error) {
	limit := args.Limit
	if limit == 0 {
		limit = 100
	}

	users, err := t.api.GetUsersContext(ctx)
	if err != nil {
		return mcp.NewToolResultError("failed to list users: " + err.Error()), nil
	}

	response := SlackListUsersResponse{
		Users: make([]UserInfo, 0),
	}

	count := 0
	for _, u := range users {
		if !args.IncludeBots && u.IsBot {
			continue
		}
		if u.Deleted {
			continue
		}

		response.Users = append(response.Users, UserInfo{
			ID:       u.ID,
			Name:     u.Name,
			RealName: u.RealName,
			Email:    u.Profile.Email,
			IsBot:    u.IsBot,
			IsAdmin:  u.IsAdmin,
		})

		count++
		if count >= limit {
			break
		}
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// SearchUsersTool returns the tool definition for searching users.
func (t *SlackTools) SearchUsersTool() mcp.Tool {
	return mcp.NewTool("slack_search_users",
		mcp.WithDescription(`Searches for users by name or email.

Returns users matching the search query.

Parameters:
- query: Search term to match against user names and emails`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search term for user names or emails")),
	)
}

// SearchUsersHandler handles slack_search_users tool calls.
func (t *SlackTools) SearchUsersHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackSearchUsersRequest,
) (*mcp.CallToolResult, error) {
	if args.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	users, err := t.api.GetUsersContext(ctx)
	if err != nil {
		return mcp.NewToolResultError("failed to search users: " + err.Error()), nil
	}

	queryLower := strings.ToLower(args.Query)
	var matched []UserInfo
	for _, u := range users {
		if u.Deleted {
			continue
		}
		if strings.Contains(strings.ToLower(u.Name), queryLower) ||
			strings.Contains(strings.ToLower(u.RealName), queryLower) ||
			strings.Contains(strings.ToLower(u.Profile.Email), queryLower) {
			matched = append(matched, UserInfo{
				ID:       u.ID,
				Name:     u.Name,
				RealName: u.RealName,
				Email:    u.Profile.Email,
				IsBot:    u.IsBot,
				IsAdmin:  u.IsAdmin,
			})
		}
	}

	response := SlackSearchUsersResponse{
		Users: matched,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// GetUserProfileTool returns the tool definition for getting user profile.
func (t *SlackTools) GetUserProfileTool() mcp.Tool {
	return mcp.NewTool("slack_get_user_profile",
		mcp.WithDescription(`Gets detailed profile information for a specific user.

Returns comprehensive user details including name, email, and role information.

Parameters:
- user_id: The ID of the user`),
		mcp.WithString("user_id",
			mcp.Required(),
			mcp.Description("User ID (e.g., U1234567890)")),
	)
}

// GetUserProfileHandler handles slack_get_user_profile tool calls.
func (t *SlackTools) GetUserProfileHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackGetUserProfileRequest,
) (*mcp.CallToolResult, error) {
	if args.UserID == "" {
		return mcp.NewToolResultError("user_id parameter is required"), nil
	}

	user, err := t.api.GetUserInfoContext(ctx, args.UserID)
	if err != nil {
		return mcp.NewToolResultError("failed to get user profile: " + err.Error()), nil
	}

	response := SlackGetUserProfileResponse{
		User: UserInfo{
			ID:       user.ID,
			Name:     user.Name,
			RealName: user.RealName,
			Email:    user.Profile.Email,
			IsBot:    user.IsBot,
			IsAdmin:  user.IsAdmin,
		},
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// ========== File Tools ==========

// ListFilesTool returns the tool definition for listing files.
func (t *SlackTools) ListFilesTool() mcp.Tool {
	return mcp.NewTool("slack_list_files",
		mcp.WithDescription(`Lists files in the Slack workspace.

Returns a list of files with metadata.

Parameters:
- count: Number of files to return (default: 20, max: 100)
- types: Comma-separated file types (e.g., images, pdfs, zips)
- channel: Filter by channel ID`),
		mcp.WithNumber("count",
			mcp.Description("Number of files (default: 20, max: 100)")),
		mcp.WithString("types",
			mcp.Description("File types filter (e.g., images, pdfs)")),
		mcp.WithString("channel",
			mcp.Description("Filter by channel ID")),
	)
}

// ListFilesHandler handles slack_list_files tool calls.
func (t *SlackTools) ListFilesHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackListFilesRequest,
) (*mcp.CallToolResult, error) {
	count := args.Count
	if count == 0 {
		count = 20
	}
	if count > 100 {
		count = 100
	}

	params := slack.GetFilesParameters{
		Count: count,
	}
	if args.Types != "" {
		params.Types = args.Types
	}
	if args.Channel != "" {
		params.Channel = args.Channel
	}

	files, _, err := t.api.GetFilesContext(ctx, params)
	if err != nil {
		return mcp.NewToolResultError("failed to list files: " + err.Error()), nil
	}

	fileList := make([]FileInfo, 0, len(files))
	for _, f := range files {
		fileList = append(fileList, FileInfo{
			ID:        f.ID,
			Name:      f.Name,
			Title:     f.Title,
			Mimetype:  f.Mimetype,
			Size:      f.Size,
			URL:       f.URLPrivate,
			User:      f.User,
			Timestamp: fmt.Sprintf("%d", f.Timestamp),
		})
	}

	response := SlackListFilesResponse{
		Files: fileList,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// SearchFilesTool returns the tool definition for searching files.
func (t *SlackTools) SearchFilesTool() mcp.Tool {
	return mcp.NewTool("slack_search_files",
		mcp.WithDescription(`Searches for files by name or content.

Returns files matching the search query.

Parameters:
- query: Search query string
- count: Number of results (default: 20, max: 100)`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query for files")),
		mcp.WithNumber("count",
			mcp.Description("Number of results (default: 20, max: 100)")),
	)
}

// SearchFilesHandler handles slack_search_files tool calls.
func (t *SlackTools) SearchFilesHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackSearchFilesRequest,
) (*mcp.CallToolResult, error) {
	if args.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	count := args.Count
	if count == 0 {
		count = 20
	}
	if count > 100 {
		count = 100
	}

	searchParams := slack.NewSearchParameters()
	searchParams.Count = count

	result, err := t.api.SearchFilesContext(ctx, args.Query, searchParams)
	if err != nil {
		return mcp.NewToolResultError("failed to search files: " + err.Error()), nil
	}

	fileList := make([]FileInfo, 0, len(result.Matches))
	for _, f := range result.Matches {
		fileList = append(fileList, FileInfo{
			ID:        f.ID,
			Name:      f.Name,
			Title:     f.Title,
			Mimetype:  f.Mimetype,
			Size:      f.Size,
			URL:       f.URLPrivate,
			User:      f.User,
			Timestamp: fmt.Sprintf("%d", f.Timestamp),
		})
	}

	response := SlackSearchFilesResponse{
		Files: fileList,
		Total: result.Total,
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}

// GetFileInfoTool returns the tool definition for getting file info.
func (t *SlackTools) GetFileInfoTool() mcp.Tool {
	return mcp.NewTool("slack_get_file_info",
		mcp.WithDescription(`Gets detailed information about a specific file.

Returns comprehensive file metadata including download URL.

Parameters:
- file_id: The ID of the file`),
		mcp.WithString("file_id",
			mcp.Required(),
			mcp.Description("File ID")),
	)
}

// GetFileInfoHandler handles slack_get_file_info tool calls.
func (t *SlackTools) GetFileInfoHandler(
	ctx context.Context,
	request mcp.CallToolRequest,
	args SlackGetFileInfoRequest,
) (*mcp.CallToolResult, error) {
	if args.FileID == "" {
		return mcp.NewToolResultError("file_id parameter is required"), nil
	}

	file, _, _, err := t.api.GetFileInfoContext(ctx, args.FileID, 0, 0)
	if err != nil {
		return mcp.NewToolResultError("failed to get file info: " + err.Error()), nil
	}

	response := SlackGetFileInfoResponse{
		File: FileInfo{
			ID:        file.ID,
			Name:      file.Name,
			Title:     file.Title,
			Mimetype:  file.Mimetype,
			Size:      file.Size,
			URL:       file.URLPrivate,
			User:      file.User,
			Timestamp: fmt.Sprintf("%d", file.Timestamp),
		},
	}

	data, err := types.MarshalResponse(response)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal response: " + err.Error()), nil
	}
	return mcp.NewToolResultText(data), nil
}
