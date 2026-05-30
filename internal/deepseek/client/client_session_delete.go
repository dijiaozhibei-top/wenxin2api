package client

import (
	"context"
	"errors"

	"ds2api/internal/auth"
)

// Stub - Session deletion not implemented for Wenxin.

type DeleteSessionResult struct {
	SessionID    string
	Success      bool
	ErrorMessage string
}

func (c *Client) DeleteSession(ctx context.Context, a *auth.RequestAuth, sessionID string, maxAttempts int) (*DeleteSessionResult, error) {
	return &DeleteSessionResult{SessionID: sessionID}, errors.New("session deletion not implemented for Wenxin")
}

func (c *Client) DeleteSessionForToken(ctx context.Context, token string, sessionID string) (*DeleteSessionResult, error) {
	return &DeleteSessionResult{SessionID: sessionID}, errors.New("session deletion not implemented for Wenxin")
}

func (c *Client) DeleteAllSessions(ctx context.Context, a *auth.RequestAuth) error {
	return nil
}

func (c *Client) DeleteAllSessionsForToken(ctx context.Context, token string) error {
	return nil
}
