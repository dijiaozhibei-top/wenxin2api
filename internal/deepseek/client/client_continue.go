package client

import (
	"context"
	"net/http"

	"ds2api/internal/auth"
)

// Stub - Wenxin does not have a continue mechanism.
// Auto-continue was a DeepSeek feature to handle INCOMPLETE/AUTO_CONTINUE status.
func (c *Client) wrapCompletionWithAutoContinue(ctx context.Context, a *auth.RequestAuth, payload map[string]any, powResp string, resp *http.Response) *http.Response {
	return resp
}
