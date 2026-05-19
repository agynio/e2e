//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	agentsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/agents/v1"
	llmv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/llm/v1"
	organizationsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/organizations/v1"
	runnerv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/runner/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	usersv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/users/v1"
	"github.com/google/uuid"
)

const (
	agynWaitAgentAResponse = "Agent B replied successfully via agyn --wait."
	agynWaitAgentBResponse = "Agent B received the agyn wait check-in."
)

func TestAgentAgynCLIWaitToAnotherAgent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	t.Cleanup(cancel)

	agentsConn := dialGRPC(t, agentsAddr)
	threadsConn := dialGRPC(t, threadsAddr)
	runnerConn := dialRunnerGRPC(t, runnerAddr)
	usersConn := dialGRPC(t, usersAddr)
	orgsConn := dialGRPC(t, orgsAddr)
	llmConn := dialGRPC(t, llmAddr)

	agentsClient := agentsv1.NewAgentsServiceClient(agentsConn)
	threadsClient := threadsv1.NewThreadsServiceClient(threadsConn)
	runnerClient := runnerv1.NewRunnerServiceClient(runnerConn)
	usersClient := usersv1.NewUsersServiceClient(usersConn)
	orgsClient := organizationsv1.NewOrganizationsServiceClient(orgsConn)
	llmClient := llmv1.NewLLMServiceClient(llmConn)

	identityID := resolveOrCreateUser(t, ctx, usersClient)
	threadsCtx := withIdentity(ctx, identityID)
	orgID := createTestOrganization(t, ctx, orgsClient, identityID)

	provider := createLLMProvider(t, ctx, llmClient, testLLMEndpointAgn, orgID)
	providerID := provider.GetMeta().GetId()
	if providerID == "" {
		t.Fatal("create llm provider: missing id")
	}

	agentAModel := createModel(t, ctx, llmClient, "e2e-agyn-wait-agent-a-model-"+uuid.NewString(), providerID, "shell-agyn-thread-create-wait", orgID)
	agentBModel := createModel(t, ctx, llmClient, "e2e-agyn-wait-agent-b-model-"+uuid.NewString(), providerID, "agyn-wait-agent-b-reply", orgID)

	agentBNickname := "e2e-agyn-wait-b-fixed"
	agentB := createAgentWithNickname(t, threadsCtx, agentsClient, "e2e-agyn-wait-agent-b-"+uuid.NewString(), agentBNickname, agentBModel.GetMeta().GetId(), orgID, agnInitImage)
	agentBID := agentB.GetMeta().GetId()
	if agentBID == "" {
		t.Fatal("create agent B: missing id")
	}
	t.Cleanup(func() { deleteAgent(t, threadsCtx, agentsClient, agentBID) })

	agentA := createAgent(t, threadsCtx, agentsClient, "e2e-agyn-wait-agent-a-"+uuid.NewString(), agentAModel.GetMeta().GetId(), orgID, agnInitImage)
	agentAID := agentA.GetMeta().GetId()
	if agentAID == "" {
		t.Fatal("create agent A: missing id")
	}
	t.Cleanup(func() { deleteAgent(t, threadsCtx, agentsClient, agentAID) })

	threadA := createThread(t, threadsCtx, threadsClient, orgID, []string{identityID, agentAID})
	threadAID := threadA.GetId()
	if threadAID == "" {
		t.Fatal("create agent A thread: missing id")
	}
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadAID) })

	uniqueRef := "e2e-agyn-wait-fixed"
	sentinel := "e2e-agyn-wait-sentinel-fixed"
	prompt := fmt.Sprintf(
		"Use agyn CLI to create a new thread with @%s using ref %s, send the exact text %q, wait for the reply, then tell me whether it worked.",
		agentBNickname,
		uniqueRef,
		"Please reply with "+sentinel,
	)
	sentMessage := sendMessage(t, threadsCtx, threadsClient, threadAID, identityID, prompt)
	sentMessageTime := messageCreatedAt(t, sentMessage)
	labels := map[string]string{
		labelManagedBy: managedByValue,
		labelAgentID:   agentAID,
		labelThreadID:  threadAID,
	}
	t.Cleanup(func() {
		ids, err := findWorkloadsByLabels(ctx, runnerClient, labels)
		if err != nil {
			t.Logf("cleanup: find workloads: %v", err)
			return
		}
		for _, workloadID := range ids {
			cleanupWorkload(t, ctx, runnerClient, workloadID)
		}
	})

	pollCtx, pollCancel := context.WithTimeout(threadsCtx, 5*time.Minute)
	defer pollCancel()
	agentABody, err := pollForAgentResponse(t, pollCtx, threadsClient, runnerClient, threadAID, agentAID, labels, sentMessageTime, agynWaitAgentAResponse)
	if err != nil {
		logAgynWaitDiagnostics(t, threadsCtx, threadsClient, orgID, threadAID, agentAID, agentBID)
		t.Fatalf("wait for agent A agyn --wait response: %v", err)
	}
	if strings.Contains(agentABody, "notification stream closed") {
		t.Fatalf("agent A response contains notification stream failure: %q", agentABody)
	}

	threadB, messagesB, err := findThreadWithParticipantsAndMessage(threadsCtx, threadsClient, orgID, agentAID, agentBID, sentinel)
	if err != nil {
		logAgynWaitDiagnostics(t, threadsCtx, threadsClient, orgID, threadAID, agentAID, agentBID)
		t.Fatalf("find agent A to agent B thread: %v", err)
	}
	t.Cleanup(func() { archiveThread(t, threadsCtx, threadsClient, threadB.GetId()) })
	if !messagesContainSenderBody(messagesB, agentBID, agynWaitAgentBResponse) {
		t.Fatalf("agent B reply %q not found in thread %s; messages=%s", agynWaitAgentBResponse, threadB.GetId(), describeThreadMessages(messagesB))
	}
}

func findThreadWithParticipantsAndMessage(ctx context.Context, client threadsv1.ThreadsServiceClient, orgID, participantA, participantB, bodySubstring string) (*threadsv1.Thread, []*threadsv1.Message, error) {
	var foundThread *threadsv1.Thread
	var foundMessages []*threadsv1.Message
	err := pollUntil(ctx, pollInterval, func(ctx context.Context) error {
		resp, err := client.ListOrganizationThreads(ctx, &threadsv1.ListOrganizationThreadsRequest{
			OrganizationId: orgID,
			Filter: &threadsv1.ListOrganizationThreadsFilter{
				ParticipantIdIn: []string{participantA, participantB},
				StatusIn:        []threadsv1.ThreadStatus{threadsv1.ThreadStatus_THREAD_STATUS_ACTIVE},
			},
			Sort: &threadsv1.ListOrganizationThreadsSort{
				Field:     threadsv1.ListOrganizationThreadsSortField_LIST_ORGANIZATION_THREADS_SORT_FIELD_UPDATED,
				Direction: threadsv1.SortDirection_SORT_DIRECTION_DESC,
			},
			PageSize: 25,
		})
		if err != nil {
			return fmt.Errorf("list organization threads: %w", err)
		}
		for _, thread := range resp.GetThreads() {
			if thread == nil || thread.GetId() == "" || !threadHasParticipants(thread, participantA, participantB) {
				continue
			}
			messagesResp, err := client.GetMessages(ctx, &threadsv1.GetMessagesRequest{ThreadId: thread.GetId(), PageSize: 50})
			if err != nil {
				return fmt.Errorf("get messages for %s: %w", thread.GetId(), err)
			}
			messages := messagesResp.GetMessages()
			if messagesContainBodySubstring(messages, bodySubstring) {
				foundThread = thread
				foundMessages = messages
				return nil
			}
		}
		return fmt.Errorf("thread with participants %s/%s and message containing %q not found", participantA, participantB, bodySubstring)
	})
	if err != nil {
		return nil, nil, err
	}
	return foundThread, foundMessages, nil
}

func threadHasParticipants(thread *threadsv1.Thread, participantA, participantB string) bool {
	foundA := false
	foundB := false
	for _, participant := range thread.GetParticipants() {
		switch participant.GetId() {
		case participantA:
			foundA = true
		case participantB:
			foundB = true
		}
	}
	return foundA && foundB
}

func messagesContainBodySubstring(messages []*threadsv1.Message, substring string) bool {
	for _, msg := range messages {
		if strings.Contains(msg.GetBody(), substring) {
			return true
		}
	}
	return false
}

func messagesContainSenderBody(messages []*threadsv1.Message, senderID, body string) bool {
	for _, msg := range messages {
		if msg.GetSenderId() == senderID && msg.GetBody() == body {
			return true
		}
	}
	return false
}

func describeThreadMessages(messages []*threadsv1.Message) string {
	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		parts = append(parts, fmt.Sprintf("sender=%s body=%q", msg.GetSenderId(), msg.GetBody()))
	}
	return strings.Join(parts, "; ")
}

func logAgynWaitDiagnostics(t *testing.T, ctx context.Context, threadsClient threadsv1.ThreadsServiceClient, orgID, threadAID, agentAID, agentBID string) {
	t.Helper()
	if resp, err := threadsClient.GetMessages(ctx, &threadsv1.GetMessagesRequest{ThreadId: threadAID, PageSize: 50}); err == nil {
		t.Logf("diagnostics: agent A thread messages: %s", describeThreadMessages(resp.GetMessages()))
	} else {
		t.Logf("diagnostics: get agent A thread messages: %v", err)
	}
	if resp, err := threadsClient.ListOrganizationThreads(ctx, &threadsv1.ListOrganizationThreadsRequest{
		OrganizationId: orgID,
		Filter:         &threadsv1.ListOrganizationThreadsFilter{ParticipantIdIn: []string{agentAID, agentBID}},
		PageSize:       25,
	}); err == nil {
		for _, thread := range resp.GetThreads() {
			if thread == nil || thread.GetId() == "" || !threadHasParticipants(thread, agentAID, agentBID) {
				continue
			}
			messagesResp, err := threadsClient.GetMessages(ctx, &threadsv1.GetMessagesRequest{ThreadId: thread.GetId(), PageSize: 50})
			if err != nil {
				t.Logf("diagnostics: get messages for candidate thread %s: %v", thread.GetId(), err)
				continue
			}
			t.Logf("diagnostics: candidate agent A/B thread %s messages: %s", thread.GetId(), describeThreadMessages(messagesResp.GetMessages()))
		}
	} else {
		t.Logf("diagnostics: list candidate agent A/B threads: %v", err)
	}
}
