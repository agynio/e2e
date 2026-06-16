//go:build e2e && svc_agn_cli

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	agynGatewayURLEnv     = "AGYN_BASE_URL"
	agynAPITokenEnv       = "AGYN_API_TOKEN"
	agynOrganizationIDEnv = "AGYN_ORGANIZATION_ID"
)

type agynEgressRuleOutput struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Matcher        struct {
		DomainPattern string   `json:"domain_pattern"`
		Ports         []int32  `json:"ports"`
		Methods       []string `json:"methods"`
		PathPattern   string   `json:"path_pattern"`
	} `json:"matcher"`
	Effect struct {
		Action  string `json:"action"`
		Headers []struct {
			Name   string `json:"name"`
			Scheme string `json:"scheme"`
			Source string `json:"source"`
		} `json:"headers"`
	} `json:"effect"`
}

type agynEgressAttachmentOutput struct {
	ID      string `json:"id"`
	RuleID  string `json:"rule_id"`
	AgentID string `json:"agent_id"`
}

func TestAgynEgressRuleLifecycle(t *testing.T) {
	binary := agnBinaryPath(t)
	env := newAgynCLIGatewayEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	organizationID := requireAgynCLIEnv(t, agynOrganizationIDEnv)
	ruleName := "e2e-cli-egress-" + uniqueID()
	updatedRuleName := ruleName + "-updated"
	domainPattern := fmt.Sprintf("api-%s.example.com", uniqueID())
	agentID := uniqueID()

	createStdout, _ := runAgnWithContext(t, ctx, binary, env, "egress", "rule", "create",
		"--organization-id", organizationID,
		"--name", ruleName,
		"--description", "E2E CLI egress rule",
		"--domain", domainPattern,
		"--port", "443",
		"--method", "GET",
		"--path", "/repos/**",
		"--action", "allow",
		"--header", "X-E2E-CLI=literal-token",
		"--output", "json",
	)
	rule := decodeAgynEgressRule(t, createStdout)
	require.NotEmpty(t, rule.ID)
	require.Equal(t, organizationID, rule.OrganizationID)
	require.Equal(t, ruleName, rule.Name)
	require.Equal(t, domainPattern, rule.Matcher.DomainPattern)
	require.Equal(t, []int32{443}, rule.Matcher.Ports)
	require.Equal(t, []string{"GET"}, rule.Matcher.Methods)
	require.Equal(t, "/repos/**", rule.Matcher.PathPattern)
	require.Equal(t, "allow", rule.Effect.Action)
	require.Len(t, rule.Effect.Headers, 1)
	require.Equal(t, "X-E2E-CLI", rule.Effect.Headers[0].Name)
	require.Equal(t, "value", rule.Effect.Headers[0].Source)

	ruleID := rule.ID
	deleted := false
	defer func() {
		if !deleted {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			_, _ = runAgnWithContextNoFail(cleanupCtx, binary, env, "egress", "rule", "delete", ruleID)
		}
	}()

	listStdout, _ := runAgnWithContext(t, ctx, binary, env, "egress", "rule", "list", "--organization-id", organizationID, "--output", "json")
	listedRules := decodeAgynEgressRuleList(t, listStdout)
	requireAgynRuleID(t, listedRules, ruleID)

	getStdout, _ := runAgnWithContext(t, ctx, binary, env, "egress", "rule", "get", ruleID, "--output", "json")
	gotRule := decodeAgynEgressRule(t, getStdout)
	require.Equal(t, ruleID, gotRule.ID)
	require.Equal(t, ruleName, gotRule.Name)

	updateStdout, _ := runAgnWithContext(t, ctx, binary, env, "egress", "rule", "update", ruleID,
		"--name", updatedRuleName,
		"--description", "E2E CLI egress rule updated",
		"--domain", domainPattern,
		"--port", "8443",
		"--method", "POST",
		"--path", "/repos/**/issues",
		"--action", "deny",
		"--header", "X-E2E-CLI=updated-literal-token",
		"--output", "json",
	)
	updatedRule := decodeAgynEgressRule(t, updateStdout)
	require.Equal(t, ruleID, updatedRule.ID)
	require.Equal(t, updatedRuleName, updatedRule.Name)
	require.Equal(t, []int32{8443}, updatedRule.Matcher.Ports)
	require.Equal(t, []string{"POST"}, updatedRule.Matcher.Methods)
	require.Equal(t, "/repos/**/issues", updatedRule.Matcher.PathPattern)
	require.Equal(t, "deny", updatedRule.Effect.Action)

	attachStdout, _ := runAgnWithContext(t, ctx, binary, env, "egress", "rule", "attach", ruleID, agentID, "--output", "json")
	attachment := decodeAgynEgressAttachment(t, attachStdout)
	require.NotEmpty(t, attachment.ID)
	require.Equal(t, ruleID, attachment.RuleID)
	require.Equal(t, agentID, attachment.AgentID)

	_, _ = runAgnWithContext(t, ctx, binary, env, "egress", "rule", "detach", attachment.ID)
	_, _ = runAgnWithContext(t, ctx, binary, env, "egress", "rule", "delete", ruleID)
	deleted = true
}

func newAgynCLIGatewayEnv(t *testing.T) []string {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".agyn")
	require.NoError(t, os.MkdirAll(configDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "credentials"), []byte(requireAgynCLIEnv(t, agynAPITokenEnv)+"\n"), 0o600))
	return append(os.Environ(),
		"HOME="+home,
		"AGYN_GATEWAY_URL="+requireAgynCLIEnv(t, agynGatewayURLEnv),
	)
}

func requireAgynCLIEnv(t *testing.T, key string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(key))
	require.NotEmpty(t, value, "%s is required for agyn CLI live egress E2E", key)
	return value
}

func decodeAgynEgressRule(t *testing.T, payload string) agynEgressRuleOutput {
	t.Helper()
	var rule agynEgressRuleOutput
	require.NoError(t, json.Unmarshal([]byte(payload), &rule), "payload: %s", payload)
	return rule
}

func decodeAgynEgressRuleList(t *testing.T, payload string) []agynEgressRuleOutput {
	t.Helper()
	var rules []agynEgressRuleOutput
	require.NoError(t, json.Unmarshal([]byte(payload), &rules), "payload: %s", payload)
	return rules
}

func decodeAgynEgressAttachment(t *testing.T, payload string) agynEgressAttachmentOutput {
	t.Helper()
	var attachment agynEgressAttachmentOutput
	require.NoError(t, json.Unmarshal([]byte(payload), &attachment), "payload: %s", payload)
	return attachment
}

func requireAgynRuleID(t *testing.T, rules []agynEgressRuleOutput, ruleID string) {
	t.Helper()
	for _, rule := range rules {
		if rule.ID == ruleID {
			return
		}
	}
	t.Fatalf("egress rule %s not found in list response: %+v", ruleID, rules)
}

func runAgnWithContextNoFail(ctx context.Context, binary string, env []string, args ...string) (string, string) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	return stdout.String(), stderr.String()
}

func uniqueID() string {
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}
