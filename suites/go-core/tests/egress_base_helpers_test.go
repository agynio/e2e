//go:build e2e && (((svc_egress || svc_egress_gateway) && !(svc_agents_orchestrator || svc_runners || svc_metering || svc_k8s_runner || svc_organizations || svc_llm || svc_llm_proxy || smoke)) || (svc_egress && svc_egress_gateway))

package tests

import (
	"context"
	"fmt"
	"time"
)

const (
	pollInterval = 2 * time.Second
	testTimeout  = 120 * time.Second
)

var (
	orgsAddr    = envOrDefault("ORGANIZATIONS_ADDRESS", "organizations:50051")
	runnerAddr  = envOrDefault("RUNNER_ADDRESS", "k8s-runner:50051")
	secretsAddr = envOrDefault("SECRETS_ADDRESS", "secrets:50051")
	usersAddr   = envOrDefault("USERS_ADDRESS", "users:50051")
)

func pollUntil(ctx context.Context, interval time.Duration, check func(ctx context.Context) error) error {
	lastErr := check(ctx)
	if lastErr == nil {
		return nil
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("poll timed out: last error: %w", lastErr)
		case <-ticker.C:
			if err := check(ctx); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
	}
}
