//go:build e2e && (svc_reminders || svc_gateway)

package tests

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

const (
	remindersPollInterval = 2 * time.Second
	remindersWaitTimeout  = 2 * time.Minute
)

func TestRemindersAgentLoop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)

	orgID := gatewayOrganizationID(t)
	token := gatewayAPIToken(t)
	caller := fetchGatewayIdentity(t, token)
	callerID := caller.IdentityID

	llmConn := dialGRPC(t, envOrDefault("LLM_ADDRESS", "llm:50051"))
	agentsConn := dialGRPC(t, envOrDefault("AGENTS_ADDRESS", "agents:50051"))
	threadsConn := dialGRPC(t, envOrDefault("THREADS_ADDRESS", "threads:50051"))

	llmClient := llmv1.NewLLMServiceClient(llmConn)
	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)

	provider := createRemindersLLMProvider(t, ctx, llmClient, testLLMEndpointAgn, orgID)
	providerID := strings.TrimSpace(provider.GetMeta().GetId())
	if providerID == "" {
		t.Fatal("create llm provider: missing id")
	}

	model := createRemindersModel(t, ctx, llmClient, "e2e-reminders-model-"+uuid.NewString(), providerID, "reminders-agent-loop", orgID)
	modelID := strings.TrimSpace(model.GetMeta().GetId())
	if modelID == "" {
		t.Fatal("create model: missing id")
	}

	initImage := requireEnv("AGN_INIT_IMAGE")
	agent := createRemindersAgent(t, ctx, agentsClient, fmt.Sprintf("e2e-reminders-agent-%s", uuid.NewString()), modelID, orgID, initImage)
	agentID := strings.TrimSpace(agent.GetMeta().GetId())
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		deleteRemindersAgent(t, cleanupCtx, agentsClient, agentID)
	})

	reminderNote := fmt.Sprintf("e2e reminder %s", uuid.NewString())
	createRemindersAgentEnv(t, ctx, agentsClient, agentID, "LLM_API_TOKEN", token)
	createRemindersAgentEnv(t, ctx, agentsClient, agentID, "AGYN_ORGANIZATION_ID", orgID)
	createRemindersAgentEnv(t, ctx, agentsClient, agentID, "REMINDER_NOTE", reminderNote)

	threadsCtx := remindersWithIdentity(ctx, callerID)
	thread := createRemindersThread(t, threadsCtx, threadsClient, orgID, []string{callerID, agentID})
	threadID := strings.TrimSpace(thread.GetId())
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		archiveRemindersThread(t, remindersWithIdentity(cleanupCtx, callerID), threadsClient, threadID)
	})

	sentMessage := sendRemindersMessage(t, threadsCtx, threadsClient, threadID, callerID, "Schedule a short reminder and acknowledge it when it arrives.")
	sentAt := remindersMessageCreatedAt(t, sentMessage)

	expectedScheduled := "Scheduled. I will reply when the reminder arrives."
	expectedReminder := fmt.Sprintf("Reminder: %s", reminderNote)
	expectedAck := "Acknowledged: reminder received."

	pollCtx, pollCancel := context.WithTimeout(threadsCtx, remindersWaitTimeout)
	defer pollCancel()

	var scheduledMsg *threadsv1.Message
	var reminderMsg *threadsv1.Message
	var ackMsg *threadsv1.Message

	err := remindersPollUntil(pollCtx, remindersPollInterval, func(ctx context.Context) error {
		resp, err := threadsClient.GetMessages(ctx, &threadsv1.GetMessagesRequest{
			ThreadId: threadID,
			PageSize: 50,
		})
		if err != nil {
			return fmt.Errorf("get messages: %w", err)
		}
		filtered := make([]*threadsv1.Message, 0, len(resp.GetMessages()))
		for _, msg := range resp.GetMessages() {
			if msg == nil {
				return fmt.Errorf("message is nil")
			}
			createdAt := msg.GetCreatedAt()
			if createdAt == nil {
				return fmt.Errorf("message %s missing created_at", msg.GetId())
			}
			if createdAt.AsTime().Before(sentAt) {
				continue
			}
			filtered = append(filtered, msg)
		}
		sort.Slice(filtered, func(i, j int) bool {
			return remindersMessageCreatedAt(t, filtered[i]).Before(remindersMessageCreatedAt(t, filtered[j]))
		})

		scheduledMsg = nil
		reminderMsg = nil
		ackMsg = nil
		for _, msg := range filtered {
			if scheduledMsg == nil {
				if msg.GetSenderId() == agentID && msg.GetBody() == expectedScheduled {
					scheduledMsg = msg
				}
				continue
			}
			if reminderMsg == nil {
				if msg.GetBody() == expectedReminder {
					reminderMsg = msg
				}
				continue
			}
			if ackMsg == nil {
				if msg.GetSenderId() == agentID && msg.GetBody() == expectedAck {
					ackMsg = msg
				}
				continue
			}
		}
		if scheduledMsg == nil || reminderMsg == nil || ackMsg == nil {
			return fmt.Errorf("waiting for reminders flow messages")
		}
		reminderAt := remindersMessageCreatedAt(t, reminderMsg)
		ackAt := remindersMessageCreatedAt(t, ackMsg)
		if !ackAt.After(reminderAt) {
			return fmt.Errorf("acknowledgment created_at %s not after reminder %s", ackAt, reminderAt)
		}
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, scheduledMsg)
	require.NotNil(t, reminderMsg)
	require.NotNil(t, ackMsg)
}

func remindersPollUntil(ctx context.Context, interval time.Duration, check func(ctx context.Context) error) error {
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

func createRemindersLLMProvider(t *testing.T, ctx context.Context, client llmv1.LLMServiceClient, endpoint, orgID string) *llmv1.LLMProvider {
	t.Helper()
	resp, err := client.CreateLLMProvider(ctx, &llmv1.CreateLLMProviderRequest{
		Endpoint:       endpoint,
		AuthMethod:     llmv1.AuthMethod_AUTH_METHOD_BEARER,
		Token:          "test-token",
		OrganizationId: orgID,
	})
	if err != nil {
		t.Fatalf("create llm provider: %v", err)
	}
	provider := resp.GetProvider()
	if provider == nil || provider.GetMeta() == nil {
		t.Fatal("create llm provider: nil response")
	}
	return provider
}

func createRemindersModel(t *testing.T, ctx context.Context, client llmv1.LLMServiceClient, name, providerID, remoteName, orgID string) *llmv1.Model {
	t.Helper()
	resp, err := client.CreateModel(ctx, &llmv1.CreateModelRequest{
		Name:           name,
		LlmProviderId:  providerID,
		RemoteName:     remoteName,
		OrganizationId: orgID,
	})
	if err != nil {
		t.Fatalf("create model %q: %v", name, err)
	}
	model := resp.GetModel()
	if model == nil || model.GetMeta() == nil {
		t.Fatal("create model: nil response")
	}
	return model
}

func createRemindersAgent(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, name, model, organizationID, initImage string) *agentsv1.Agent {
	t.Helper()
	if strings.TrimSpace(initImage) == "" {
		t.Fatal("create agent: init image is required")
	}
	resp, err := client.CreateAgent(ctx, &agentsv1.CreateAgentRequest{
		Name:           name,
		Role:           "assistant",
		Model:          model,
		Image:          "alpine:3.21",
		InitImage:      initImage,
		OrganizationId: organizationID,
	})
	if err != nil {
		t.Fatalf("create agent %q: %v", name, err)
	}
	agent := resp.GetAgent()
	if agent == nil || agent.GetMeta() == nil {
		t.Fatal("create agent: nil response")
	}
	return agent
}

func deleteRemindersAgent(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, agentID string) {
	t.Helper()
	_, err := client.DeleteAgent(ctx, &agentsv1.DeleteAgentRequest{Id: agentID})
	if err != nil {
		t.Logf("cleanup: delete agent %s: %v", agentID, err)
	}
}

func createRemindersAgentEnv(t *testing.T, ctx context.Context, client agentsv1.AgentsServiceClient, agentID, name, value string) *agentsv1.Env {
	t.Helper()
	resp, err := client.CreateEnv(ctx, &agentsv1.CreateEnvRequest{
		Name:   name,
		Target: &agentsv1.CreateEnvRequest_AgentId{AgentId: agentID},
		Source: &agentsv1.CreateEnvRequest_Value{Value: value},
	})
	if err != nil {
		t.Fatalf("create agent env %q: %v", name, err)
	}
	env := resp.GetEnv()
	if env == nil || env.GetMeta() == nil {
		t.Fatal("create agent env: nil response")
	}
	return env
}

func createRemindersThread(t *testing.T, ctx context.Context, client threadsv1.ThreadsServiceClient, organizationID string, participantIDs []string) *threadsv1.Thread {
	t.Helper()
	if organizationID == "" {
		t.Fatal("create thread: missing organization id")
	}
	resp, err := client.CreateThread(ctx, &threadsv1.CreateThreadRequest{
		ParticipantIds: participantIDs,
		OrganizationId: &organizationID,
	})
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	thread := resp.GetThread()
	if thread == nil {
		t.Fatal("create thread: nil response")
	}
	return thread
}

func archiveRemindersThread(t *testing.T, ctx context.Context, client threadsv1.ThreadsServiceClient, threadID string) {
	t.Helper()
	_, err := client.ArchiveThread(ctx, &threadsv1.ArchiveThreadRequest{ThreadId: threadID})
	if err != nil {
		t.Logf("cleanup: archive thread %s: %v", threadID, err)
	}
}

func sendRemindersMessage(t *testing.T, ctx context.Context, client threadsv1.ThreadsServiceClient, threadID, senderID, body string) *threadsv1.Message {
	t.Helper()
	resp, err := client.SendMessage(ctx, &threadsv1.SendMessageRequest{
		ThreadId: threadID,
		SenderId: senderID,
		Body:     body,
	})
	if err != nil {
		t.Fatalf("send message on thread %s: %v", threadID, err)
	}
	msg := resp.GetMessage()
	if msg == nil {
		t.Fatal("send message: nil response")
	}
	return msg
}

func remindersMessageCreatedAt(t *testing.T, msg *threadsv1.Message) time.Time {
	t.Helper()
	if msg == nil {
		t.Fatal("message is nil")
	}
	createdAt := msg.GetCreatedAt()
	if createdAt == nil {
		t.Fatal("message created_at is nil")
	}
	return createdAt.AsTime()
}

func remindersWithIdentity(ctx context.Context, identityID string) context.Context {
	md := metadata.New(map[string]string{"x-identity-id": identityID})
	return metadata.NewOutgoingContext(ctx, md)
}
