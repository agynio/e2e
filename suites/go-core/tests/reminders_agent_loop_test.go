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
)

const (
	remindersWaitTimeout = 2 * time.Minute
)

func TestRemindersAgentLoop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)

	orgID := gatewayOrganizationID(t)
	token := gatewayAPIToken(t)
	caller := fetchGatewayIdentity(t, token)
	callerID := caller.IdentityID
	remindersAppIdentity := remindersAppIdentityID(t)

	llmConn := dialGRPC(t, llmAddr)
	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)

	llmClient := llmv1.NewLLMServiceClient(llmConn)
	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)

	provider := createLLMProvider(t, ctx, llmClient, testLLMEndpointAgn, orgID)
	providerID := strings.TrimSpace(provider.GetMeta().GetId())
	if providerID == "" {
		t.Fatal("create llm provider: missing id")
	}

	model := createModel(t, ctx, llmClient, "e2e-reminders-model-"+uuid.NewString(), providerID, "reminders-agent-loop", orgID)
	modelID := strings.TrimSpace(model.GetMeta().GetId())
	if modelID == "" {
		t.Fatal("create model: missing id")
	}

	agent := createAgent(t, ctx, agentsClient, fmt.Sprintf("e2e-reminders-agent-%s", uuid.NewString()), modelID, orgID, agnInitImage)
	agentID := strings.TrimSpace(agent.GetMeta().GetId())
	if agentID == "" {
		t.Fatal("create agent: missing id")
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		deleteAgent(t, cleanupCtx, agentsClient, agentID)
	})

	reminderNote := fmt.Sprintf("e2e reminder %s", uuid.NewString())
	createAgentEnv(t, ctx, agentsClient, agentID, "LLM_API_TOKEN", token)
	createAgentEnv(t, ctx, agentsClient, agentID, "AGYN_API_TOKEN", token)
	createAgentEnv(t, ctx, agentsClient, agentID, "AGYN_BASE_URL", zitiGatewayBaseURL)
	createAgentEnv(t, ctx, agentsClient, agentID, "AGYN_ORGANIZATION_ID", orgID)
	createAgentEnv(t, ctx, agentsClient, agentID, "REMINDER_NOTE", reminderNote)

	threadsCtx := remindersIdentityContext(ctx, callerID)
	thread := createThread(t, threadsCtx, threadsClient, orgID, []string{callerID, agentID})
	threadID := strings.TrimSpace(thread.GetId())
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		archiveThread(t, remindersIdentityContext(cleanupCtx, callerID), threadsClient, threadID)
	})
	t.Cleanup(func() {
		cancelPendingRemindersBestEffort(t, threadID)
	})

	sentMessage := sendMessage(t, threadsCtx, threadsClient, threadID, callerID, "Schedule a short reminder and acknowledge it when it arrives.")
	sentAt := messageCreatedAt(t, sentMessage)

	expectedScheduled := "Scheduled. I will reply when the reminder arrives."
	expectedReminder := fmt.Sprintf("Reminder: %s", reminderNote)
	expectedAck := "Acknowledged: reminder received."

	pollCtx, pollCancel := context.WithTimeout(threadsCtx, remindersWaitTimeout)
	defer pollCancel()

	var scheduledMsg *threadsv1.Message
	var reminderMsg *threadsv1.Message
	var ackMsg *threadsv1.Message

	err := pollUntil(pollCtx, pollInterval, func(ctx context.Context) error {
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
			return messageCreatedAt(t, filtered[i]).Before(messageCreatedAt(t, filtered[j]))
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
				if msg.GetSenderId() == remindersAppIdentity && msg.GetBody() == expectedReminder {
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
		reminderAt := messageCreatedAt(t, reminderMsg)
		ackAt := messageCreatedAt(t, ackMsg)
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
