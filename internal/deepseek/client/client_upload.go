package client

import (
	"context"
	"errors"

	"ds2api/internal/auth"
)

// Stub - File operations not yet implemented for Wenxin.

type UploadFileRequest struct {
	Filename    string
	ContentType string
	Purpose     string
	ModelType   string
	Data        []byte
}

type UploadFileResult struct {
	ID         string
	Filename   string
	Bytes      int64
	Status     string
	Purpose    string
	AccountID  string
	IsImage    bool
}

func (c *Client) UploadFile(ctx context.Context, a *auth.RequestAuth, req UploadFileRequest, maxAttempts int) (*UploadFileResult, error) {
	return nil, errors.New("file upload not implemented for Wenxin")
}
