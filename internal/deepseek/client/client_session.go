package client

import (
	"context"
	"errors"

	"ds2api/internal/auth"
)

// Stub - Wenxin uses lid (returned in SSE basedata) for session identification.
// Session management via DeepSeek session APIs is not applicable.

type SessionInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	TitleType string `json:"title_type"`
	Pinned    bool   `json:"pinned"`
	UpdatedAt int64  `json:"updated_at,omitempty"`
}

type SessionStats struct {
	AccountID      string `json:"account_id"`
	FirstPageCount int    `json:"first_page_count"`
	PinnedCount    int    `json:"pinned_count"`
	HasMore        bool   `json:"has_more"`
	Success        bool   `json:"success"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

func (c *Client) GetSessionCount(ctx context.Context, a *auth.RequestAuth, maxAttempts int) (*SessionStats, error) {
	return &SessionStats{}, errors.New("session management not implemented for Wenxin")
}

func (c *Client) GetSessionCountForToken(ctx context.Context, token string) (*SessionStats, error) {
	return &SessionStats{Success: true}, nil
}

func (c *Client) GetSessionCountAll(ctx context.Context) []*SessionStats {
	return nil
}

func (c *Client) FetchSessionPage(ctx context.Context, a *auth.RequestAuth, cursor string) ([]SessionInfo, bool, error) {
	return nil, false, errors.New("session management not implemented for Wenxin")
}
