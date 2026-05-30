package completionruntime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"ds2api/internal/assistantturn"
	"ds2api/internal/auth"
	"ds2api/internal/config"
	dsclient "ds2api/internal/deepseek/client"
	"ds2api/internal/httpapi/openai/history"
	"ds2api/internal/httpapi/openai/shared"
	"ds2api/internal/promptcompat"
	"ds2api/internal/sse"
)

type DeepSeekCaller interface {
	CreateSession(ctx context.Context, a *auth.RequestAuth, maxAttempts int) (string, error)
	GetPow(ctx context.Context, a *auth.RequestAuth, maxAttempts int) (string, error)
	UploadFile(ctx context.Context, a *auth.RequestAuth, req dsclient.UploadFileRequest, maxAttempts int) (*dsclient.UploadFileResult, error)
	CallCompletion(ctx context.Context, a *auth.RequestAuth, payload map[string]any, powResp string, maxAttempts int) (*http.Response, error)
}

type Options struct {
	StripReferenceMarkers bool
	MaxAttempts           int
	RetryEnabled          bool
	RetryMaxAttempts      int
	CurrentInputFile      history.CurrentInputConfigReader
}

type NonStreamResult struct {
	SessionID string
	Payload   map[string]any
	Turn      assistantturn.Turn
	Attempts  int
}

type StartResult struct {
	SessionID string
	Payload   map[string]any
	Pow       string
	Response  *http.Response
	Request   promptcompat.StandardRequest
}

func StartCompletion(ctx context.Context, ds DeepSeekCaller, a *auth.RequestAuth, stdReq promptcompat.StandardRequest, opts Options) (StartResult, *assistantturn.OutputError) {
	payload := stdReq.WenxinCompletionPayload(a.Account)
	resp, err := ds.CallCompletion(ctx, a, payload, "", 1)
	if err != nil {
		return StartResult{Payload: payload, Request: stdReq}, &assistantturn.OutputError{Status: http.StatusInternalServerError, Message: "Failed to get completion.", Code: "error"}
	}
	return StartResult{Payload: payload, Response: resp, Request: stdReq}, nil
}

func prepareCurrentInputFile(ctx context.Context, ds DeepSeekCaller, a *auth.RequestAuth, stdReq promptcompat.StandardRequest, opts Options) (promptcompat.StandardRequest, *assistantturn.OutputError) {
	return stdReq, nil
}

func ExecuteNonStreamWithRetry(ctx context.Context, ds DeepSeekCaller, a *auth.RequestAuth, stdReq promptcompat.StandardRequest, opts Options) (NonStreamResult, *assistantturn.OutputError) {
	start, startErr := StartCompletion(ctx, ds, a, stdReq, opts)
	if startErr != nil {
		return NonStreamResult{}, startErr
	}
	return ExecuteNonStreamStartedWithRetry(ctx, ds, a, start, opts)
}

func ExecuteNonStreamStartedWithRetry(ctx context.Context, ds DeepSeekCaller, a *auth.RequestAuth, start StartResult, opts Options) (NonStreamResult, *assistantturn.OutputError) {
	stdReq := start.Request
	payload := start.Payload

	attempts := 0
	for {
		turn, outErr := collectAttempt(start.Response, stdReq, opts)
		if outErr != nil {
			return NonStreamResult{Payload: payload, Attempts: attempts}, outErr
		}

		retryMax := opts.RetryMaxAttempts
		if retryMax <= 0 {
			retryMax = shared.EmptyOutputRetryMaxAttempts()
		}
		if !opts.RetryEnabled || !assistantturn.ShouldRetryEmptyOutput(turn, attempts, retryMax) {
			return NonStreamResult{Payload: payload, Turn: turn, Attempts: attempts}, turn.Error
		}

		attempts++
		config.Logger.Info("[completion_runtime_empty_retry] attempting retry", "surface", stdReq.Surface, "stream", false, "retry_attempt", attempts)

		chatToken := a.DeepSeekToken
		_ = chatToken
		retryPayload := stdReq.WenxinCompletionPayload(a.Account)
		nextResp, err := ds.CallCompletion(ctx, a, retryPayload, "", 1)
		if err != nil {
			return NonStreamResult{Payload: payload, Turn: turn, Attempts: attempts}, &assistantturn.OutputError{Status: http.StatusInternalServerError, Message: "Failed to get completion.", Code: "error"}
		}
		start.Response = nextResp
		start.Payload = retryPayload
	}
}

func canRetryOnAlternateAccount(ctx context.Context, a *auth.RequestAuth, outErr *assistantturn.OutputError, retryEnabled bool, attempted *bool) bool {
	if outErr == nil || outErr.Status != http.StatusTooManyRequests {
		return false
	}
	if !retryEnabled || attempted == nil || *attempted {
		return false
	}
	if a == nil || !a.UseConfigToken {
		return false
	}
	*attempted = true
	return a.SwitchAccount(ctx)
}

func startStandardCompletionOnAlternateAccount(ctx context.Context, ds DeepSeekCaller, a *auth.RequestAuth, stdReq promptcompat.StandardRequest, opts Options, maxAttempts int) (StartResult, *assistantturn.OutputError) {
	payload := stdReq.WenxinCompletionPayload(a.Account)
	resp, err := ds.CallCompletion(ctx, a, payload, "", maxAttempts)
	if err != nil {
		return StartResult{Payload: payload}, &assistantturn.OutputError{Status: http.StatusInternalServerError, Message: "Failed to get completion.", Code: "error"}
	}
	return StartResult{Payload: payload, Response: resp, Request: stdReq}, nil
}

func reuploadCurrentInputFileForAccount(ctx context.Context, ds DeepSeekCaller, a *auth.RequestAuth, stdReq promptcompat.StandardRequest, opts Options) (promptcompat.StandardRequest, *assistantturn.OutputError) {
	return stdReq, nil
}

func collectAttempt(resp *http.Response, stdReq promptcompat.StandardRequest, opts Options) (assistantturn.Turn, *assistantturn.OutputError) {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			config.Logger.Warn("[completion_runtime] response body close failed", "surface", stdReq.Surface, "error", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return assistantturn.Turn{}, &assistantturn.OutputError{Status: resp.StatusCode, Message: message, Code: "error"}
	}

	wenxinResult := dsclient.CollectWenxinStream(resp)

	turn := assistantturn.BuildTurnFromCollected(sse.CollectResult{
		Text:     wenxinResult.Text,
		Thinking: wenxinResult.Thinking,
	}, buildOptions(stdReq, stdReq.PromptTokenText, opts))
	return turn, nil
}

func buildOptions(stdReq promptcompat.StandardRequest, prompt string, opts Options) assistantturn.BuildOptions {
	return assistantturn.BuildOptions{
		Model:                 stdReq.ResponseModel,
		Prompt:                prompt,
		RefFileTokens:         stdReq.RefFileTokens,
		SearchEnabled:         stdReq.Search,
		StripReferenceMarkers: opts.StripReferenceMarkers,
		ToolNames:             stdReq.ToolNames,
		ToolsRaw:              stdReq.ToolsRaw,
		ToolChoice:            stdReq.ToolChoice,
	}
}

func authOutputError(a *auth.RequestAuth) *assistantturn.OutputError {
	if a != nil && a.UseConfigToken {
		return &assistantturn.OutputError{Status: http.StatusUnauthorized, Message: "Account token is invalid. Please re-login the account in admin.", Code: "error"}
	}
	return &assistantturn.OutputError{Status: http.StatusUnauthorized, Message: "Invalid token. If this should be an API key, add it to config.keys first.", Code: "error"}
}

func Errorf(status int, format string, args ...any) *assistantturn.OutputError {
	return &assistantturn.OutputError{Status: status, Message: fmt.Sprintf(format, args...), Code: "error"}
}
