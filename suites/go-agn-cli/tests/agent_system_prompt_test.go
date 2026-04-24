//go:build e2e && svc_agn_cli

package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAgentSystemPrompt(t *testing.T) {
	env := newTestEnv(t, "system-prompt", "You are personal assistant")
	binary := agnBinaryPath(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stdout, _ := runAgnWithContext(t, ctx, binary, env.env, "exec", "hi")
	require.Equal(t, "Hello! I am here to help!", strings.TrimSpace(stdout))
}
