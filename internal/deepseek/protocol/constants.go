package protocol

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

const (
	WenxinHost                 = "chat.baidu.com"
	WenxinConversationURL      = "https://chat.baidu.com/aichat/api/conversation"
	WenxinHistoryURL           = "https://chat.baidu.com/csaitab/history/list"
	WenxinMessagesURL          = "https://chat.baidu.com/aichat/api/messages/list"
	WenxinMessagesConfURL      = "https://chat.baidu.com/aichat/api/messages/conf"
	WenxinSugURL               = "https://chat.baidu.com/aichat/api/aitabserver"
	WenxinLongTaskListURL      = "https://chat.baidu.com/aichat/longtask/list"
	WenxinLongTaskQueryURL     = "https://chat.baidu.com/aichat/longtask/query"
	WenxinImpushURL            = "https://chat.baidu.com/aichat/api/impush"
	WenxinSidebarURL           = "https://chat.baidu.com/csaitab/sidebar/list"
)

var defaultStaticBaseHeaders = map[string]string{
	"Host":         "chat.baidu.com",
	"Accept":       "text/event-stream",
	"Content-Type": "application/json",
}

var defaultSkipContainsPatterns = []string{}

var defaultSkipExactPaths = []string{}

var ClientVersion string
var BaseHeaders = map[string]string{}
var SkipContainsPatterns = cloneStringSlice(defaultSkipContainsPatterns)
var SkipExactPathSet = toStringSet(defaultSkipExactPaths)

type clientConstants struct {
	Name            string `json:"name"`
	Platform        string `json:"platform"`
	Version         string `json:"version"`
	AndroidAPILevel string `json:"android_api_level"`
	Locale          string `json:"locale"`
}

type sharedConstants struct {
	Client              clientConstants   `json:"client"`
	BaseHeaders         map[string]string `json:"base_headers"`
	SkipContainsPattern []string          `json:"skip_contains_patterns"`
	SkipExactPaths      []string          `json:"skip_exact_paths"`
}

//go:embed constants_shared.json
var sharedConstantsJSON []byte

func init() {
	cfg := sharedConstants{}
	if err := json.Unmarshal(sharedConstantsJSON, &cfg); err != nil {
		panic(fmt.Errorf("load Wenxin shared constants: %w", err))
	}
	applySharedConstants(cfg)
}

func applySharedConstants(cfg sharedConstants) {
	client := normalizeClientConstants(cfg.Client)
	ClientVersion = client.Version
	BaseHeaders = buildBaseHeaders(client, cfg.BaseHeaders)
	SkipContainsPatterns = cloneStringSlice(defaultSkipContainsPatterns)
	if len(cfg.SkipContainsPattern) > 0 {
		SkipContainsPatterns = cloneStringSlice(cfg.SkipContainsPattern)
	}
	SkipExactPathSet = toStringSet(defaultSkipExactPaths)
	if len(cfg.SkipExactPaths) > 0 {
		SkipExactPathSet = toStringSet(cfg.SkipExactPaths)
	}
}

func normalizeClientConstants(in clientConstants) clientConstants {
	if in.Name == "" {
		in.Name = "BaiduWenxin_PC"
	}
	if in.Platform == "" {
		in.Platform = "web"
	}
	if in.Locale == "" {
		in.Locale = "zh_CN"
	}
	return in
}

func buildBaseHeaders(client clientConstants, overrides map[string]string) map[string]string {
	out := cloneStringMap(defaultStaticBaseHeaders)
	for k, v := range overrides {
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	if client.Name != "" && client.Version != "" {
		userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"
		out["User-Agent"] = userAgent
	}
	if client.Platform != "" {
		out["x-client-platform"] = client.Platform
	}
	if client.Version != "" {
		out["x-client-version"] = client.Version
	}
	if client.Locale != "" {
		out["accept-language"] = client.Locale
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringSlice(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func toStringSet(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		out[v] = struct{}{}
	}
	return out
}

const (
	KeepAliveTimeout  = 5
	StreamIdleTimeout = 300
	MaxKeepaliveCount = 40
)
