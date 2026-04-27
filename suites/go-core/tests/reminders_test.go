//go:build e2e && (svc_reminders || svc_gateway)

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	appsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/apps/v1"
	threadsv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/threads/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

const remindersAppSlug = "reminders"

type reminderResponse struct {
	ID          string  `json:"id"`
	ThreadID    string  `json:"thread_id"`
	IdentityID  string  `json:"identity_id"`
	Note        string  `json:"note"`
	Status      string  `json:"status"`
	At          string  `json:"at"`
	CreatedAt   string  `json:"created_at"`
	CompletedAt *string `json:"completed_at"`
	CancelledAt *string `json:"cancelled_at"`
}

type singleReminderResponse struct {
	Reminder reminderResponse `json:"reminder"`
}

type listRemindersResponse struct {
	Reminders []reminderResponse `json:"reminders"`
}

func TestRemindersGatewayHealthz(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	request, err := newRemindersRequest(t, ctx, http.MethodGet, "healthz", nil)
	require.NoError(t, err)

	response, err := remindersGatewayClient(t).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("healthz returned %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
}

func TestRemindersCreateGetRoundtrip(t *testing.T) {
	callerIdentity := fetchGatewayIdentity(t, gatewayAPIToken(t)).IdentityID
	threadID := uuid.NewString()
	note := fmt.Sprintf("roundtrip %s", uuid.NewString())

	created := createReminder(t, threadID, 60, note)
	t.Cleanup(func() { cancelReminderBestEffort(t, created.ID) })

	require.NotEmpty(t, created.ID)
	require.Equal(t, threadID, created.ThreadID)
	require.Equal(t, callerIdentity, created.IdentityID)
	require.Equal(t, note, created.Note)
	require.Equal(t, "pending", created.Status)
	require.NotEmpty(t, created.At)
	require.NotEmpty(t, created.CreatedAt)
	require.Nil(t, created.CompletedAt)
	require.Nil(t, created.CancelledAt)

	fetched := getReminder(t, created.ID)
	require.Equal(t, created, fetched)
}

func TestRemindersListFilters(t *testing.T) {
	threadID := uuid.NewString()
	pending := createReminder(t, threadID, 3600, "pending "+uuid.NewString())
	cancelled := createReminder(t, threadID, 3600, "cancelled "+uuid.NewString())
	t.Cleanup(func() { cancelReminderBestEffort(t, pending.ID) })
	t.Cleanup(func() { cancelReminderBestEffort(t, cancelled.ID) })

	cancelReminder := cancelReminderStrict(t, cancelled.ID)
	require.Equal(t, "cancelled", cancelReminder.Status)

	defaultList := listReminders(t, threadID, "")
	require.ElementsMatch(t, []string{pending.ID}, reminderIDs(defaultList.Reminders))

	pendingList := listReminders(t, threadID, "pending")
	require.ElementsMatch(t, []string{pending.ID}, reminderIDs(pendingList.Reminders))

	cancelledList := listReminders(t, threadID, "cancelled")
	require.ElementsMatch(t, []string{cancelled.ID}, reminderIDs(cancelledList.Reminders))

	allList := listReminders(t, threadID, "all")
	require.ElementsMatch(t, []string{pending.ID, cancelled.ID}, reminderIDs(allList.Reminders))
}

func TestRemindersCancelSemantics(t *testing.T) {
	threadID := uuid.NewString()
	reminder := createReminder(t, threadID, 3600, "cancel "+uuid.NewString())
	t.Cleanup(func() { cancelReminderBestEffort(t, reminder.ID) })

	cancelled := cancelReminderStrict(t, reminder.ID)
	require.Equal(t, "cancelled", cancelled.Status)
	require.NotNil(t, cancelled.CancelledAt)
	require.Nil(t, cancelled.CompletedAt)

	secondResp := postRemindersJSON(t, "cancel-reminder", map[string]string{"reminder_id": reminder.ID})
	require.Equal(t, http.StatusConflict, secondResp.StatusCode)
	secondResp.Body.Close()

	notFoundResp := postRemindersJSON(t, "cancel-reminder", map[string]string{"reminder_id": uuid.NewString()})
	require.Equal(t, http.StatusNotFound, notFoundResp.StatusCode)
	notFoundResp.Body.Close()
}

func TestRemindersDeliveryHappyPath(t *testing.T) {
	callerIdentity := fetchGatewayIdentity(t, gatewayAPIToken(t)).IdentityID
	threadsClient := newThreadsClient(t)
	appIdentity := remindersAppIdentityID(t)

	threadID := createThreadWithAppParticipant(t, threadsClient, callerIdentity, appIdentity)
	t.Cleanup(func() { archiveThreadBestEffort(t, threadsClient, callerIdentity, threadID) })

	note := "delivery " + uuid.NewString()
	created := createReminder(t, threadID, 0, note)
	t.Cleanup(func() { cancelReminderBestEffort(t, created.ID) })

	completed := pollReminderStatus(t, created.ID, "completed", 30*time.Second)
	require.NotNil(t, completed.CompletedAt)

	messages := pollThreadMessages(t, threadsClient, callerIdentity, threadID, 1, 30*time.Second)
	require.Len(t, messages, 1)
	require.Equal(t, "Reminder: "+note, messages[0].GetBody())
	require.Equal(t, appIdentity, messages[0].GetSenderId())
}

func TestRemindersDeliveryFailurePending(t *testing.T) {
	threadID := uuid.NewString()
	created := createReminder(t, threadID, 0, "missing thread "+uuid.NewString())
	t.Cleanup(func() { cancelReminderBestEffort(t, created.ID) })

	time.Sleep(10 * time.Second)

	reminder := getReminder(t, created.ID)
	require.Equal(t, "pending", reminder.Status)
	require.Nil(t, reminder.CompletedAt)
}

func remindersAppsAddr() string {
	return envOrDefault("APPS_ADDRESS", "apps:50051")
}

func remindersThreadsAddr() string {
	return envOrDefault("THREADS_ADDRESS", "threads:50051")
}

func remindersGatewayClient(t *testing.T) *http.Client {
	t.Helper()
	return newGatewayAuthenticatedClient(t, gatewayAPIToken(t))
}

func remindersGatewayEndpoint(t *testing.T, path string) string {
	t.Helper()
	trimmed := strings.TrimPrefix(path, "/")
	if trimmed == "" {
		return gatewayEndpoint(t, "apps/reminders")
	}
	return gatewayEndpoint(t, "apps/reminders/"+trimmed)
}

func newRemindersRequest(t *testing.T, ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, method, remindersGatewayEndpoint(t, path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-organization-id", gatewayOrganizationID(t))
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func postRemindersJSON(t *testing.T, path string, payload any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), gatewayRequestTimeout)
	defer cancel()

	req, err := newRemindersRequest(t, ctx, http.MethodPost, path, bytes.NewReader(body))
	require.NoError(t, err)

	resp, err := remindersGatewayClient(t).Do(req)
	require.NoError(t, err)
	return resp
}

func decodeGatewayResponse[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var payload T
	require.NoError(t, decoder.Decode(&payload))
	return payload
}

func createReminder(t *testing.T, threadID string, delaySeconds int64, note string) reminderResponse {
	t.Helper()
	resp := postRemindersJSON(t, "create-reminder", map[string]any{
		"thread_id":     threadID,
		"delay_seconds": delaySeconds,
		"note":          note,
	})
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("create reminder failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	created := decodeGatewayResponse[singleReminderResponse](t, resp)
	return created.Reminder
}

func getReminder(t *testing.T, reminderID string) reminderResponse {
	t.Helper()
	resp := postRemindersJSON(t, "get-reminder", map[string]string{"reminder_id": reminderID})
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("get reminder failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	payload := decodeGatewayResponse[singleReminderResponse](t, resp)
	return payload.Reminder
}

func listReminders(t *testing.T, threadID, status string) listRemindersResponse {
	t.Helper()
	body := map[string]any{"thread_id": threadID}
	if status != "" {
		body["status"] = status
	}
	resp := postRemindersJSON(t, "list-reminders", body)
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("list reminders failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	payload := decodeGatewayResponse[listRemindersResponse](t, resp)
	return payload
}

func cancelReminderStrict(t *testing.T, reminderID string) reminderResponse {
	t.Helper()
	resp := postRemindersJSON(t, "cancel-reminder", map[string]string{"reminder_id": reminderID})
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("cancel reminder failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	payload := decodeGatewayResponse[singleReminderResponse](t, resp)
	return payload.Reminder
}

func cancelReminderBestEffort(t *testing.T, reminderID string) {
	t.Helper()
	resp := postRemindersJSON(t, "cancel-reminder", map[string]string{"reminder_id": reminderID})
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK, http.StatusConflict, http.StatusNotFound:
		return
	default:
		body, _ := io.ReadAll(resp.Body)
		t.Logf("cleanup: cancel reminder %s: status %d: %s", reminderID, resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

func reminderIDs(reminders []reminderResponse) []string {
	ids := make([]string, 0, len(reminders))
	for _, reminder := range reminders {
		ids = append(ids, reminder.ID)
	}
	return ids
}

func newThreadsClient(t *testing.T) threadsv1.ThreadsServiceClient {
	t.Helper()
	conn := dialGRPC(t, remindersThreadsAddr())
	return threadsv1.NewThreadsServiceClient(conn)
}

func newAppsClient(t *testing.T) appsv1.AppsServiceClient {
	t.Helper()
	conn := dialGRPC(t, remindersAppsAddr())
	return appsv1.NewAppsServiceClient(conn)
}

func remindersAppIdentityID(t *testing.T) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := newAppsClient(t).GetAppBySlug(ctx, &appsv1.GetAppBySlugRequest{Slug: remindersAppSlug})
	require.NoError(t, err)
	if resp == nil || resp.GetApp() == nil {
		t.Fatal("get app by slug: missing app")
	}
	identityID := strings.TrimSpace(resp.GetApp().GetIdentityId())
	if identityID == "" {
		t.Fatal("get app by slug: missing identity id")
	}
	return identityID
}

func createThreadWithAppParticipant(
	t *testing.T,
	client threadsv1.ThreadsServiceClient,
	callerIdentityID string,
	appIdentityID string,
) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	callCtx := remindersIdentityContext(ctx, callerIdentityID)
	resp, err := client.CreateThread(callCtx, &threadsv1.CreateThreadRequest{
		ParticipantIds: []string{callerIdentityID, appIdentityID},
	})
	require.NoError(t, err)
	thread := resp.GetThread()
	if thread == nil {
		t.Fatal("create thread: missing thread")
	}
	threadID := strings.TrimSpace(thread.GetId())
	if threadID == "" {
		t.Fatal("create thread: missing id")
	}
	return threadID
}

func archiveThreadBestEffort(t *testing.T, client threadsv1.ThreadsServiceClient, identityID, threadID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	callCtx := remindersIdentityContext(ctx, identityID)
	_, err := client.ArchiveThread(callCtx, &threadsv1.ArchiveThreadRequest{ThreadId: threadID})
	if err != nil {
		t.Logf("cleanup: archive thread %s: %v", threadID, err)
	}
}

func pollReminderStatus(t *testing.T, reminderID, targetStatus string, timeout time.Duration) reminderResponse {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		reminder := getReminder(t, reminderID)
		if reminder.Status == targetStatus {
			return reminder
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("reminder %s did not reach status %q within %s", reminderID, targetStatus, timeout)
	return reminderResponse{}
}

func pollThreadMessages(
	t *testing.T,
	client threadsv1.ThreadsServiceClient,
	identityID string,
	threadID string,
	expectedCount int,
	timeout time.Duration,
) []*threadsv1.Message {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		messages := getThreadMessages(t, client, identityID, threadID)
		if len(messages) >= expectedCount {
			return messages
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("expected %d messages for thread %s within %s", expectedCount, threadID, timeout)
	return nil
}

func getThreadMessages(
	t *testing.T,
	client threadsv1.ThreadsServiceClient,
	identityID string,
	threadID string,
) []*threadsv1.Message {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	callCtx := remindersIdentityContext(ctx, identityID)
	var all []*threadsv1.Message
	pageToken := ""
	for {
		resp, err := client.GetMessages(callCtx, &threadsv1.GetMessagesRequest{
			ThreadId:  threadID,
			PageSize:  100,
			PageToken: pageToken,
		})
		require.NoError(t, err)
		all = append(all, resp.GetMessages()...)
		pageToken = resp.GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	return all
}

func remindersIdentityContext(ctx context.Context, identityID string) context.Context {
	md := metadata.New(map[string]string{"x-identity-id": identityID})
	return metadata.NewOutgoingContext(ctx, md)
}
