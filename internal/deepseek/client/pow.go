package client

import (
	"context"
	"errors"
)

// Stub - Wenxin has no PoW mechanism.
func ComputePow(ctx context.Context, challenge map[string]any) (int64, error) {
	return 0, errors.New("PoW not implemented for Wenxin")
}

func BuildPowHeader(challenge map[string]any, answer int64) (string, error) {
	return "", nil
}
