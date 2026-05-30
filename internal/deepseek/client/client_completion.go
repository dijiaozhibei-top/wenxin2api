package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	dsprotocol "ds2api/internal/deepseek/protocol"
	"ds2api/internal/auth"
	"ds2api/internal/config"
	trans "ds2api/internal/deepseek/transport"
)

func (c *Client) CallCompletion(ctx context.Context, a *auth.RequestAuth, payload map[string]any, powResp string, maxAttempts int) (*http.Response, error) {
	_ = maxAttempts
	clients := c.requestClientsForAuth(ctx, a)
	headers := c.authHeaders(a.DeepSeekToken)
	headers["x-chat-message"] = buildChatMessageHeader(payload)
	headers["origin"] = "https://chat.baidu.com"
	headers["referer"] = "https://chat.baidu.com/search?enter_type=chat_url&internal=1"
	headers["landingpageswitch"] = ""
	headers["personifiedswitch"] = "0"
	headers["sec-fetch-dest"] = "empty"
	headers["sec-fetch-mode"] = "cors"
	headers["sec-fetch-site"] = "same-origin"
	headers["sec-ch-ua"] = `"Chromium";v="147", "Not.A/Brand";v="8", "Microsoft Edge";v="147"`
	headers["sec-ch-ua-mobile"] = "?0"
	headers["sec-ch-ua-platform"] = `"Windows"`
	if cookies := strings.TrimSpace(a.Account.Cookies); cookies != "" {
		headers["cookie"] = cookies
	}
	captureSession := c.capture.Start("wenxin_conversation", dsprotocol.WenxinConversationURL, a.AccountID, payload)
	resp, err := c.streamPost(ctx, clients.stream, dsprotocol.WenxinConversationURL, headers, payload)
	if err != nil {
		return nil, err
	}
	if captureSession != nil {
		resp.Body = captureSession.WrapBody(resp.Body, resp.StatusCode)
	}
	return resp, nil
}

func buildChatMessageHeader(payload map[string]any) string {
	msg, _ := payload["message"].(map[string]any)
	if msg == nil {
		return ""
	}
	si, _ := msg["searchInfo"].(map[string]any)
	if si == nil {
		return ""
	}
	enterType, _ := si["enter_type"].(string)
	reRank, _ := si["re_rank"].(string)
	sa, _ := si["sa"].(string)
	queryRaw, _ := msg["query"].([]map[string]any)
	var queryText string
	if len(queryRaw) > 0 {
		if txtData, ok := queryRaw[0]["data"].(map[string]any); ok {
			if txt, ok := txtData["text"].(map[string]any); ok {
				queryText, _ = txt["query"].(string)
			}
		}
	}
	antiExt, _ := msg["anti_ext"].(map[string]any)
	antiExtJSON, _ := json.Marshal(antiExt)
	return "query:" + url.QueryEscape(queryText) + ",anti_ext:" + url.QueryEscape(string(antiExtJSON)) + ",enter_type:" + enterType + ",re_rank:" + reRank + ",modelName:smartMode,sa:" + sa
}

func (c *Client) streamPost(ctx context.Context, doer trans.Doer, url string, headers map[string]string, payload any) (*http.Response, error) {
	return c.streamPostWithFallback(ctx, doer, url, headers, payload, true)
}

func (c *Client) streamPostWithFallback(ctx context.Context, doer trans.Doer, url string, headers map[string]string, payload any, allowFallback bool) (*http.Response, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	headers = c.jsonHeaders(headers)
	clients := c.requestClientsFromContext(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := doer.Do(req)
	if err != nil {
		if allowFallback {
			config.Logger.Warn("[wenxin] fingerprint stream request failed", "url", url, "error", err)
			req2, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
			if req2 != nil {
				for k, v := range headers {
					req2.Header.Set(k, v)
				}
			}
			return clients.fallbackS.Do(req2)
		}
		return nil, err
	}
	return resp, nil
}

type WenxinStreamResult struct {
	Text, Thinking, Lid string
	EndTurn             bool
}

func CollectWenxinStream(resp *http.Response) WenxinStreamResult {
	if resp == nil || resp.Body == nil {
		return WenxinStreamResult{}
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return ParseWenxinSSE(string(bodyBytes))
}

func ParseWenxinSSE(bodyStr string) WenxinStreamResult {
	var result WenxinStreamResult
	var currentEvent string
	scanner := bufio.NewScanner(strings.NewReader(bodyStr))
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var evt map[string]any
		if json.Unmarshal([]byte(dataStr), &evt) != nil {
			continue
		}
		switch currentEvent {
		case "basedata":
			if lid, ok := evt["lid"].(string); ok {
				result.Lid = lid
			}
		case "message":
			if endTurn, ok := evt["endTurn"].(bool); ok && endTurn {
				result.EndTurn = true
			}
			collectWenxinContent(evt, &result)
		}
	}
	return result
}

func collectWenxinContent(evt map[string]any, result *WenxinStreamResult) {
	data, _ := evt["data"].(map[string]any)
	if data == nil {
		return
	}
	message, _ := data["message"].(map[string]any)
	if message == nil {
		return
	}
	content, _ := message["content"].(map[string]any)
	if content == nil {
		return
	}
	gen, _ := content["generator"].(map[string]any)
	if gen == nil {
		return
	}
	genData, _ := gen["data"].(map[string]any)
	if genData == nil {
		return
	}
	switch gen["component"].(string) {
	case "markdown-yiyan":
		if v, ok := genData["value"].(string); ok {
			result.Text += v
		}
	case "thinkingSteps":
		if arr, ok := genData["reasoningContentArr"].([]any); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result.Thinking += s
				}
			}
		}
	}
}

func BuildDeepSeekSSEFromText(text, thinking string) string {
	var buf strings.Builder
	write := func(path, content string) {
		runes := []rune(content)
		for i := 0; i < len(runes); i += 5 {
			end := i + 5
			if end > len(runes) {
				end = len(runes)
			}
			patch, _ := json.Marshal(map[string]any{"p": path, "v": string(runes[i:end])})
			buf.WriteString("data: ")
			buf.Write(patch)
			buf.WriteString("\n")
		}
	}
	if thinking != "" {
		write("response/thinking_content", thinking)
	}
	write("response/content", text)
	finish, _ := json.Marshal(map[string]any{"p": "response/status", "v": "FINISHED"})
	buf.WriteString("data: " + string(finish) + "\n")
	buf.WriteString("data: [DONE]\n")
	return buf.String()
}

// ConvertWenxinToDeepSeekStreamRealtime reads Wenxin SSE events from r
// in real-time, converts to DeepSeek SSE format, and writes to w immediately.
func ConvertWenxinToDeepSeekStreamRealtime(r io.Reader, w *io.PipeWriter) {
	defer w.Close()
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 128*1024)
	scanner.Buffer(buf, 2*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		// Forward data lines directly
		if _, err := io.WriteString(w, line+"\n"); err != nil {
			return
		}
	}
	io.WriteString(w, "data: [DONE]\n")
}
