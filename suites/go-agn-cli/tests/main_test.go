//go:build e2e && svc_agn_cli

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testLLMEndpoint = "https://testllm.dev/v1/org/agynio/suite/agn"

type testEnv struct {
	home string
	env  []string
}

type summarizationSpec struct {
	model      string
	keepTokens int
	maxTokens  int
}

type configSpec struct {
	model             string
	systemPrompt      string
	tokenCountingAddr string
	summarization     *summarizationSpec
	mcpCommand        string
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestAgnExecHello(t *testing.T) {
	env := newTestEnv(t, "simple-hello", "")
	binary := agnBinaryPath(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stdout, stderr := runAgnWithContext(t, ctx, binary, env.env, "exec", "hi")
	require.Equal(t, "Hi! How are you?", strings.TrimSpace(stdout))
	_ = parseThreadID(t, stderr)
}

func TestExecStatePersistence(t *testing.T) {
	env := newTestEnv(t, "simple-state", "")
	binary := agnBinaryPath(t)
	threadID := "thread-test"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stdout, stderr := runAgnWithContext(t, ctx, binary, env.env, "exec", "--thread-id", threadID, "hi")
	require.Equal(t, "Hi! How are you?", strings.TrimSpace(stdout))
	require.Equal(t, threadID, parseThreadID(t, stderr))

	stdout, stderr = runAgnWithContext(t, ctx, binary, env.env, "exec", "--thread-id", threadID, "fine")
	require.Equal(t, "How can I help you?", strings.TrimSpace(stdout))
	require.Equal(t, threadID, parseThreadID(t, stderr))

	statePath := filepath.Join(env.home, ".agyn", "agn", "threads", threadID+".json")
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var persisted struct {
		Messages []json.RawMessage `json:"messages"`
	}
	require.NoError(t, json.Unmarshal(data, &persisted))
	require.Len(t, persisted.Messages, 4)
}

func TestExecResume(t *testing.T) {
	env := newTestEnv(t, "simple-state", "")
	binary := agnBinaryPath(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stdout, stderr := runAgnWithContext(t, ctx, binary, env.env, "exec", "hi")
	require.Equal(t, "Hi! How are you?", strings.TrimSpace(stdout))
	threadID := parseThreadID(t, stderr)

	stdout, stderr = runAgnWithContext(t, ctx, binary, env.env, "exec", "resume", threadID, "fine")
	require.Equal(t, "How can I help you?", strings.TrimSpace(stdout))
	require.Equal(t, threadID, parseThreadID(t, stderr))
}

func newTestEnv(t *testing.T, model string, systemPrompt string) testEnv {
	t.Helper()
	home := t.TempDir()
	configPath := filepath.Join(home, "config.yaml")
	configData := configSpec{
		model:             model,
		systemPrompt:      systemPrompt,
		tokenCountingAddr: tokenCountingAddress(t),
	}
	writeConfig(t, configPath, configData)
	env := append(os.Environ(), "HOME="+home, "AGN_CONFIG_PATH="+configPath)
	return testEnv{home: home, env: env}
}

func writeConfig(t *testing.T, path string, spec configSpec) {
	t.Helper()
	payload := buildConfigYAML(spec)
	require.NoError(t, os.WriteFile(path, []byte(payload), 0o600))
}

func buildConfigYAML(spec configSpec) string {
	var builder strings.Builder
	builder.WriteString("llm:\n")
	writeKeyValue(&builder, 1, "endpoint", testLLMEndpoint)
	builder.WriteString("  auth:\n")
	writeKeyValue(&builder, 2, "api_key", "dummy")
	writeKeyValue(&builder, 1, "model", spec.model)
	if strings.TrimSpace(spec.systemPrompt) != "" {
		writeKeyValue(&builder, 0, "system_prompt", spec.systemPrompt)
	}
	builder.WriteString("token_counting:\n")
	writeKeyValue(&builder, 1, "address", spec.tokenCountingAddr)
	if spec.summarization != nil {
		builder.WriteString("summarization:\n")
		builder.WriteString("  llm:\n")
		writeKeyValue(&builder, 2, "endpoint", testLLMEndpoint)
		builder.WriteString("    auth:\n")
		writeKeyValue(&builder, 3, "api_key", "dummy")
		writeKeyValue(&builder, 2, "model", spec.summarization.model)
		builder.WriteString(fmt.Sprintf("  keep_tokens: %d\n", spec.summarization.keepTokens))
		builder.WriteString(fmt.Sprintf("  max_tokens: %d\n", spec.summarization.maxTokens))
	}
	if strings.TrimSpace(spec.mcpCommand) != "" {
		builder.WriteString("mcp:\n")
		builder.WriteString("  servers:\n")
		builder.WriteString("    weather:\n")
		writeKeyValue(&builder, 3, "command", spec.mcpCommand)
	}
	return builder.String()
}

func writeKeyValue(builder *strings.Builder, indent int, key string, value string) {
	spaces := strings.Repeat("  ", indent)
	quoted := strconv.Quote(value)
	builder.WriteString(fmt.Sprintf("%s%s: %s\n", spaces, key, quoted))
}

func agnBinaryPath(t *testing.T) string {
	t.Helper()
	path := strings.TrimSpace(os.Getenv("AGN_BINARY"))
	if path == "" {
		path = filepath.Join("bin", "agn")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(suiteRoot(t), path)
	}
	info, err := os.Stat(path)
	require.NoError(t, err)
	if info.Mode()&0o111 == 0 {
		t.Fatalf("agn binary is not executable: %s", path)
	}
	return path
}

func suiteRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, ".."))
}

func runAgnWithContext(t *testing.T, ctx context.Context, binary string, env []string, args ...string) (string, string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run agn %v: %v\nstdout: %s\nstderr: %s", args, err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String()
}

func parseThreadID(t *testing.T, stderr string) string {
	t.Helper()
	for _, line := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(line, "thread_id:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "thread_id:"))
		}
	}
	t.Fatalf("thread_id not found in stderr: %q", stderr)
	return ""
}
