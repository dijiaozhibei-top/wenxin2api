package promptcompat

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"ds2api/internal/config"
)

type StandardRequest struct {
	Surface                 string
	RequestedModel          string
	ResolvedModel           string
	ResponseModel           string
	Messages                []any
	HistoryText             string
	PromptTokenText         string
	CurrentInputFileApplied bool
	CurrentInputFileID      string
	CurrentToolsFileID      string
	ToolsRaw                any
	FinalPrompt             string
	ToolNames               []string
	ToolChoice              ToolChoicePolicy
	Stream                  bool
	Thinking                bool
	Search                  bool
	RefFileIDs              []string
	RefFileTokens           int
	PassThrough             map[string]any
}

type ToolChoiceMode string

const (
	ToolChoiceAuto     ToolChoiceMode = "auto"
	ToolChoiceNone     ToolChoiceMode = "none"
	ToolChoiceRequired ToolChoiceMode = "required"
	ToolChoiceForced   ToolChoiceMode = "forced"
)

type ToolChoicePolicy struct {
	Mode       ToolChoiceMode
	ForcedName string
	Allowed    map[string]struct{}
}

func DefaultToolChoicePolicy() ToolChoicePolicy {
	return ToolChoicePolicy{Mode: ToolChoiceAuto}
}

func (p ToolChoicePolicy) IsNone() bool {
	return p.Mode == ToolChoiceNone
}

func (p ToolChoicePolicy) IsRequired() bool {
	return p.Mode == ToolChoiceRequired || p.Mode == ToolChoiceForced
}

func (p ToolChoicePolicy) Allows(name string) bool {
	if len(p.Allowed) == 0 {
		return true
	}
	_, ok := p.Allowed[name]
	return ok
}

func (r StandardRequest) CompletionPayload(sessionID string) map[string]any {
	modelID := r.ResolvedModel
	if modelID == "" {
		modelID = r.RequestedModel
	}
	modelType := "default"
	if resolvedType, ok := config.GetModelType(modelID); ok {
		modelType = resolvedType
	}
	refFileIDs := make([]any, 0, len(r.RefFileIDs))
	for _, fileID := range r.RefFileIDs {
		if fileID == "" {
			continue
		}
		refFileIDs = append(refFileIDs, fileID)
	}
	payload := map[string]any{
		"chat_session_id":   sessionID,
		"model_type":        modelType,
		"parent_message_id": nil,
		"prompt":            r.FinalPrompt,
		"ref_file_ids":      refFileIDs,
		"thinking_enabled":  r.Thinking,
		"search_enabled":    r.Search,
	}
	for k, v := range r.PassThrough {
		payload[k] = v
	}
	return payload
}

// ResolveWenxinModelName maps internal model IDs to Wenxin API model names.
func ResolveWenxinModelName(modelID string) string {
	switch modelID {
	case "ernie-4.5-turbo":
		return "smartMode"
	case "deepseek-v4-pro":
		return "DeepSeek-V4"
	case "deepseek-r1", "smartmode-thinking":
		return "DeepSeek-R1"
	case "smartmode":
		return "smartMode"
	default:
		return "smartMode"
	}
}

func (r StandardRequest) WenxinCompletionPayload(account config.Account) map[string]any {
	modelID := r.ResolvedModel
	if modelID == "" {
		modelID = r.RequestedModel
	}
	wenxinModelName := ResolveWenxinModelName(modelID)

	searchEnabled := "0"
	if r.Search {
		searchEnabled = "1"
	}

	// Generate token dynamically: session_hash = md5(query)
	token := account.ChatToken
	if account.UserHash != "" && account.UserID != "" {
		sessionHash := fmt.Sprintf("%x", md5.Sum([]byte(r.FinalPrompt)))
		timestampMs := time.Now().UnixMilli()
		payload := fmt.Sprintf("%s|%s|%d|%s", account.UserHash, sessionHash, timestampMs, account.UserID)
		encoded := base64.StdEncoding.EncodeToString([]byte(payload))
		token = fmt.Sprintf("%s-%s-3", encoded, account.UserID)
	}

	// Add strong tool call nudge right before <|Assistant|> if tools are present
	prompt := r.FinalPrompt
	if len(r.ToolNames) > 0 {
		prompt = strings.Replace(prompt, "<|Assistant|>", "[IMPORTANT: If using tools, output ONLY the <|DSML|tool_calls> XML block, no explanations.]<|Assistant|>", 1)
	}

	return map[string]any{
		"message": map[string]any{
			"inputMethod": "chat_search",
			"isRebuild":   false,
			"content": map[string]any{
				"query": "",
				"agentInfo": map[string]any{
					"agent_id": []string{""},
					"params":   `{"agt_rk":4,"agt_sess_cnt":1}`,
				},
				"agentInfoList": []any{},
				"qtype":         0,
				"extData":       map[string]any{},
			},
			"searchInfo": map[string]any{
				"srcid":   "",
				"order":   "",
				"tplname": "",
				"dqaKey":  "",
				"re_rank": "4",
				"ori_lid": "",
				"sa":      "bkb",
				"enter_type": "chat_url",
				"chatParams": map[string]any{
					"setype":       "csaitab",
					"chat_samples": "WISE_NEW_CSAITAB",
					"chat_token":   token,
					"scene":        "",
				},
				"isPrivateChat":  false,
				"usedModel": map[string]any{
					"modelName":     wenxinModelName,
					"showModelName": wenxinModelName,
					"modelFunction": map[string]any{
						"deepSearch":     "0",
						"internetSearch": searchEnabled,
					},
				},
				"landingPageSwitch": "",
				"landingPage":       "aitab",
				"ecomFrom":          "",
				"hasLocPermission":  "",
				"isInnovate":        2,
				"applid":            "",
				"a_lid":             "",
				"showMindMap":       false,
				"deepDecisionInfo": map[string]any{
					"isDeepDecision": 0,
				},
			},
			"from":   "",
			"source": "pc_csaitab",
			"query": []map[string]any{
				{
					"type": "TEXT",
					"data": map[string]any{
						"text": map[string]any{
							"query":    r.FinalPrompt,
							"extData":  "{}",
							"text_type": "",
						},
					},
				},
			},
			"anti_ext": map[string]any{
				"inputT": nil,
			},
		},
		"sa":     "bkb",
		"setype": "csaitab",
		"rank":   4,
	}
}
