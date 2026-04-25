//go:build e2e && svc_agn_cli

package tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	summarizePrompt = "Tell me about the history of computing in detail"
	summarizeReply  = "Computing began with Charles Babbage who designed the Analytical Engine in the 1830s. Ada Lovelace wrote the first algorithm. Alan Turing formalized computation in 1936. ENIAC was built in 1945. The transistor was invented in 1947 at Bell Labs. Integrated circuits followed in the late 1950s."
	followupPrompt  = "What came next?"
	followupReply   = "After integrated circuits came microprocessors and personal computers."
)

type summarizationTestEnv struct {
	turn1 []string
	turn2 []string
}

func TestSummarization(t *testing.T) {
	env := newSummarizationTestEnv(t)
	binary := agnBinaryPath(t)
	threadID := "thread-summarize"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stdout, stderr := runAgnWithContext(t, ctx, binary, env.turn1, "exec", "--thread-id", threadID, summarizePrompt)
	require.Equal(t, summarizeReply, strings.TrimSpace(stdout))
	require.Equal(t, threadID, parseThreadID(t, stderr))

	stdout, stderr = runAgnWithContext(t, ctx, binary, env.turn2, "exec", "resume", threadID, followupPrompt)
	require.Equal(t, followupReply, strings.TrimSpace(stdout))
	require.Equal(t, threadID, parseThreadID(t, stderr))
}

func newSummarizationTestEnv(t *testing.T) summarizationTestEnv {
	t.Helper()
	home := t.TempDir()
	summarization := &summarizationSpec{
		model:      "summarize-history",
		keepTokens: 4,
		maxTokens:  90,
	}

	turn1Path := filepath.Join(home, "config-turn1.yaml")
	writeConfig(t, turn1Path, configSpec{
		model:         "summarize-agent-turn1",
		summarization: summarization,
	})

	turn2Path := filepath.Join(home, "config-turn2.yaml")
	writeConfig(t, turn2Path, configSpec{
		model:         "summarize-agent-turn2",
		summarization: summarization,
	})

	base := append(os.Environ(), "HOME="+home)
	return summarizationTestEnv{
		turn1: append(append([]string{}, base...), "AGN_CONFIG_PATH="+turn1Path),
		turn2: append(append([]string{}, base...), "AGN_CONFIG_PATH="+turn2Path),
	}
}
