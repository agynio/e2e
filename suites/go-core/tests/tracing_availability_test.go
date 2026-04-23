//go:build e2e && svc_agents_orchestrator

package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	tracingv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/tracing/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var (
	tracingAvailable         bool
	tracingUnavailableReason string
)

type tracingAvailability struct {
	available         bool
	unavailableReason string
}

func TestMain(m *testing.M) {
	availability := checkTracingAvailability()
	tracingAvailable = availability.available
	tracingUnavailableReason = availability.unavailableReason
	if !tracingAvailable {
		reason := strings.TrimSpace(tracingUnavailableReason)
		if reason == "" {
			reason = "unknown reason"
		}
		log.Printf("tracing e2e unavailable: %s", reason)
	}
	os.Exit(m.Run())
}

func checkTracingAvailability() tracingAvailability {
	addr := strings.TrimSpace(tracingAddr)
	if addr == "" {
		return tracingAvailability{available: false, unavailableReason: "TRACING_ADDRESS is empty"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	traceID, err := runTraceCanary(ctx, addr)
	if err != nil {
		if isCanaryAuthFailure(err) {
			availability := checkTracingQueryAvailability(ctx, addr)
			if availability.available {
				return availability
			}
			reason := canaryAuthFailureReason(err)
			if availability.unavailableReason != "" {
				reason = fmt.Sprintf("%s; %s", reason, availability.unavailableReason)
			}
			return tracingAvailability{available: false, unavailableReason: reason}
		}
		reason := canaryFailureReason(err)
		if reason == "" {
			reason = err.Error()
		}
		return tracingAvailability{available: false, unavailableReason: fmt.Sprintf("trace canary failed: %s", reason)}
	}

	conn, err := dialGRPCForCheck(ctx, addr)
	if err != nil {
		return tracingAvailability{available: false, unavailableReason: fmt.Sprintf("dial tracing %s: %v", addr, err)}
	}
	defer conn.Close()

	queryClient := tracingv1.NewTracingServiceClient(conn)
	pollCtx, cancelPoll := context.WithTimeout(ctx, 20*time.Second)
	defer cancelPoll()

	err = pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
		resp, err := queryClient.GetTrace(ctx, &tracingv1.GetTraceRequest{TraceId: traceID})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return fmt.Errorf("trace not found")
			}
			return fmt.Errorf("get trace: %w", err)
		}
		if len(flattenSpans(resp.GetResourceSpans())) == 0 {
			return fmt.Errorf("trace has no spans")
		}
		return nil
	})
	if err != nil {
		return tracingAvailability{available: false, unavailableReason: fmt.Sprintf("tracing ingest check failed: %v", err)}
	}

	return tracingAvailability{available: true}
}

func checkTracingQueryAvailability(ctx context.Context, addr string) tracingAvailability {
	conn, err := dialGRPCForCheck(ctx, addr)
	if err != nil {
		return tracingAvailability{available: false, unavailableReason: fmt.Sprintf("dial tracing %s: %v", addr, err)}
	}
	defer conn.Close()

	queryClient := tracingv1.NewTracingServiceClient(conn)
	traceID, err := randomTraceID()
	if err != nil {
		return tracingAvailability{available: false, unavailableReason: fmt.Sprintf("generate trace id: %v", err)}
	}

	_, err = queryClient.GetTrace(ctx, &tracingv1.GetTraceRequest{TraceId: traceID})
	if err == nil {
		return tracingAvailability{available: false, unavailableReason: "tracing query check returned unexpected trace"}
	}
	if status.Code(err) == codes.NotFound {
		return tracingAvailability{available: true}
	}
	return tracingAvailability{available: false, unavailableReason: fmt.Sprintf("tracing query check failed: %v", err)}
}

func randomTraceID() ([]byte, error) {
	traceID := make([]byte, 16)
	if _, err := rand.Read(traceID); err != nil {
		return nil, err
	}
	return traceID, nil
}

func dialGRPCForCheck(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
}

func runTraceCanary(ctx context.Context, addr string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", "run", "./tracecanary")
	cmd.Env = append(os.Environ(), fmt.Sprintf("TRACING_ADDRESS=%s", addr))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		reason := strings.TrimSpace(stderr.String())
		if reason == "" {
			reason = err.Error()
		}
		return nil, fmt.Errorf("trace canary failed: %s", reason)
	}
	traceHex := strings.TrimSpace(string(output))
	if traceHex == "" {
		return nil, fmt.Errorf("trace canary returned empty trace id")
	}
	traceID, err := hex.DecodeString(traceHex)
	if err != nil {
		return nil, fmt.Errorf("decode trace id %q: %w", traceHex, err)
	}
	return traceID, nil
}

func isCanaryAuthFailure(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "source identity missing") || strings.Contains(message, "unauthenticated")
}

func canaryFailureReason(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return ""
	}
	return strings.TrimPrefix(message, "trace canary failed: ")
}

func canaryAuthFailureReason(err error) string {
	reason := strings.TrimSpace(canaryFailureReason(err))
	if reason == "" {
		reason = "source identity missing"
	}
	return fmt.Sprintf("trace canary unauthenticated: %s", reason)
}
