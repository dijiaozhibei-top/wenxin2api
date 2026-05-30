package chat

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"

	"ds2api/internal/sse"
)

func (h *Handler) consumeWenxinStreamAttempt(r *http.Request, resp *http.Response, rt *chatStreamRuntime, allowDeferEmpty bool) (bool, bool) {
	defer resp.Body.Close()
	if resp.Body == nil {
		return false, false
	}
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 128*1024)
	scanner.Buffer(buf, 2*1024*1024)

	var currentEvent string
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") || currentEvent != "message" {
			continue
		}
		dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var evt map[string]any
		if json.Unmarshal([]byte(dataStr), &evt) != nil {
			continue
		}
		data, _ := evt["data"].(map[string]any)
		if data == nil {
			continue
		}
		message, _ := data["message"].(map[string]any)
		if message == nil {
			continue
		}
		content, _ := message["content"].(map[string]any)
		if content == nil {
			continue
		}
		gen, _ := content["generator"].(map[string]any)
		if gen == nil {
			continue
		}
		genData, _ := gen["data"].(map[string]any)
		if genData == nil {
			continue
		}

		var lr sse.LineResult
		lr.Parsed = true
		switch gen["component"].(string) {
		case "markdown-yiyan":
			if v, ok := genData["value"].(string); ok && v != "" {
				lr.Parts = []sse.ContentPart{{Text: v, Type: "text"}}
				lr.NextType = "text"
			} else {
				continue
			}
		case "thinkingSteps":
			if arr, ok := genData["reasoningContentArr"].([]any); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok && s != "" {
						lr.Parts = append(lr.Parts, sse.ContentPart{Text: s, Type: "thinking"})
					}
				}
				if len(lr.Parts) > 0 {
					lr.NextType = "thinking"
				} else {
					continue
				}
			} else {
				continue
			}
		default:
			continue
		}
		if decision := rt.onParsed(lr); decision.Stop {
			return true, false
		}
	}
	if lineCount == 0 {
		return false, false
	}
	// Let Finalize hook handle history save + finish chunk
	return false, false
}
