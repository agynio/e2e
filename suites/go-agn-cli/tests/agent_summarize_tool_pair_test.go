//go:build e2e && svc_agn_cli

package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	toolPairPrompt       = "What is the weather in Paris right now please?"
	toolPairTurn1Reply   = "The weather in Paris is currently 18\u00b0C and partly cloudy."
	toolPairFollowup     = "thanks"
	toolPairTurn2Reply   = "You're welcome!"
	toolPairKeepTokens   = 30
	toolPairMaxTokens    = 50
	toolPairThreadID     = "thread-tool-pair"
	toolPairTurn1Model   = "summarize-tool-pair-turn1"
	toolPairTurn2Model   = "summarize-tool-pair-turn2"
	toolPairHistoryModel = "summarize-tool-pair-history"
)

type toolPairTestEnv struct {
	turn1 []string
	turn2 []string
}

func TestSummarizationToolPair(t *testing.T) {
	mcpBinary := buildMCPWeatherServer(t)
	env := newToolPairTestEnv(t, mcpBinary)
	binary := agnBinaryPath(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stdout, stderr := runAgnWithContext(t, ctx, binary, env.turn1, "exec", "--thread-id", toolPairThreadID, toolPairPrompt)
	require.Equal(t, toolPairTurn1Reply, strings.TrimSpace(stdout))
	require.Equal(t, toolPairThreadID, parseThreadID(t, stderr))

	stdout, stderr = runAgnWithContext(t, ctx, binary, env.turn2, "exec", "resume", toolPairThreadID, toolPairFollowup)
	require.Equal(t, toolPairTurn2Reply, strings.TrimSpace(stdout))
	require.Equal(t, toolPairThreadID, parseThreadID(t, stderr))
}

func buildMCPWeatherServer(t *testing.T) string {
	t.Helper()
	buildDir := t.TempDir()
	binary := filepath.Join(buildDir, "mcp-weather-server")
	cmd := exec.Command("go", "build", "-o", binary, "./tests/testdata/mcp_weather_server.go")
	cmd.Dir = suiteRoot(t)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "build mcp weather server: %s", strings.TrimSpace(string(output)))
	return binary
}

func newToolPairTestEnv(t *testing.T, mcpCommand string) toolPairTestEnv {
	t.Helper()
	home := t.TempDir()
	summarization := &summarizationSpec{
		model:      toolPairHistoryModel,
		keepTokens: toolPairKeepTokens,
		maxTokens:  toolPairMaxTokens,
	}

	turn1Path := filepath.Join(home, "config-tool-pair-turn1.yaml")
	writeConfig(t, turn1Path, configSpec{
		model:             toolPairTurn1Model,
		summarization:     summarization,
		tokenCountingAddr: tokenCountingAddress(t),
		mcpCommand:        mcpCommand,
	})

	turn2Path := filepath.Join(home, "config-tool-pair-turn2.yaml")
	writeConfig(t, turn2Path, configSpec{
		model:             toolPairTurn2Model,
		summarization:     summarization,
		tokenCountingAddr: tokenCountingAddress(t),
		mcpCommand:        mcpCommand,
	})

	base := append(os.Environ(), "HOME="+home)
	return toolPairTestEnv{
		turn1: append(append([]string{}, base...), "AGN_CONFIG_PATH="+turn1Path),
		turn2: append(append([]string{}, base...), "AGN_CONFIG_PATH="+turn2Path),
	}
}
