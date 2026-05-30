package client

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode"

	"ds2api/internal/auth"
	"ds2api/internal/config"
	dsprotocol "ds2api/internal/deepseek/protocol"
)

func (c *Client) Login(ctx context.Context, acc config.Account) (string, error) {
	token := strings.TrimSpace(acc.ChatToken)
	if token == "" {
		token = strings.TrimSpace(acc.Token)
	}
	if token == "" {
		return "", errors.New("missing chat_token in account config")
	}
	return token, nil
}

func (c *Client) CreateSession(ctx context.Context, a *auth.RequestAuth, maxAttempts int) (string, error) {
	return "", nil
}

func (c *Client) GetPow(ctx context.Context, a *auth.RequestAuth, maxAttempts int) (string, error) {
	return "", nil
}

func (c *Client) GetPowForTarget(ctx context.Context, a *auth.RequestAuth, targetPath string, maxAttempts int) (string, error) {
	return "", nil
}

func (c *Client) authHeaders(token string) map[string]string {
	headers := make(map[string]string, len(dsprotocol.BaseHeaders)+1)
	for k, v := range dsprotocol.BaseHeaders {
		headers[k] = v
	}
	return headers
}

func isTokenInvalid(status int, code int, bizCode int, msg string, bizMsg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg) + " " + strings.TrimSpace(bizMsg))
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return true
	}
	return strings.Contains(msg, "token") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "expired") ||
		strings.Contains(msg, "not login") ||
		strings.Contains(msg, "login required")
}

func shouldAttemptRefresh(status int, code int, bizCode int, msg string, bizMsg string) bool {
	if isTokenInvalid(status, code, bizCode, msg, bizMsg) {
		return true
	}
	return status == http.StatusOK &&
		code == 0 &&
		bizCode != 0 &&
		isAuthIndicativeBizFailure(msg, bizMsg)
}

func isAuthIndicativeBizFailure(msg string, bizMsg string) bool {
	combined := strings.ToLower(strings.TrimSpace(msg) + " " + strings.TrimSpace(bizMsg))
	authKeywords := []string{
		"auth",
		"authorization",
		"credential",
		"expired",
		"invalid jwt",
		"jwt",
		"login",
		"not login",
		"session expired",
		"token",
		"unauthorized",
		"登录",
		"未登录",
		"认证",
		"凭证",
		"会话过期",
		"令牌",
	}
	for _, keyword := range authKeywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}
	return false
}

func authFailureKind(useConfigToken bool) FailureKind {
	if useConfigToken {
		return FailureManagedUnauthorized
	}
	return FailureDirectUnauthorized
}

func failureMessage(msg string, bizMsg string, fallback string) string {
	if trimmed := strings.TrimSpace(bizMsg); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(msg); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}

func normalizeMobileForLogin(raw string) (mobile string, areaCode any) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	hasPlus := strings.HasPrefix(s, "+")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if digits == "" {
		return "", nil
	}
	if (hasPlus || strings.HasPrefix(digits, "86")) && strings.HasPrefix(digits, "86") && len(digits) == 13 {
		return digits[2:], nil
	}
	return digits, nil
}
