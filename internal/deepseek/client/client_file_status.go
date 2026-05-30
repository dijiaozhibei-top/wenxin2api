package client

import (
	"context"
	"errors"

	"ds2api/internal/auth"
)

// Stub - File status polling not implemented for Wenxin.

var ErrUploadFileNotFound = errors.New("uploaded file not found")

func (c *Client) waitForUploadedFile(ctx context.Context, a *auth.RequestAuth, result *UploadFileResult) error {
	return nil
}

func (c *Client) FetchUploadedFile(ctx context.Context, a *auth.RequestAuth, fileID string) (*UploadFileResult, error) {
	return nil, ErrUploadFileNotFound
}
