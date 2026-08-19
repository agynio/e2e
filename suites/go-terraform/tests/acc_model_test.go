//go:build e2e && (svc_gateway || tf_provider_agyn)

package tests

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// A model is organization configuration, not something a release ships, so a
// freshly installed platform has none. The CI action used to create one and
// hand it over in AGYN_MODEL_ID; that step ran for every caller, needed a URL
// the runner could reach, and failed the whole job when the Gateway answered
// 503 in the seconds after a rollout. The suite that wants a model makes one,
// the way go-core already does.
//
// AGYN_MODEL_ID still wins when a caller has one.
var (
	modelOnce sync.Once
	modelID   string
	modelErr  error
)

func requireModelID(t *testing.T) string {
	t.Helper()
	if value := strings.TrimSpace(os.Getenv("AGYN_MODEL_ID")); value != "" {
		return value
	}
	modelOnce.Do(func() { modelID, modelErr = resolveOrCreateModel(t) })
	if modelErr != nil {
		t.Fatalf("resolve a model for the suite: %v", modelErr)
	}
	return modelID
}

func resolveOrCreateModel(t *testing.T) (string, error) {
	t.Helper()
	organizationID := requireOrganizationID(t)

	var models struct {
		Models []struct {
			ID   string `json:"id"`
			Meta struct {
				ID string `json:"id"`
			} `json:"meta"`
		} `json:"models"`
	}
	if err := gatewayCall(t, "LLMGateway/ListModels", map[string]any{
		"organizationId": organizationID,
		"llmProviderId":  "",
		"pageSize":       200,
		"pageToken":      "",
	}, &models); err != nil {
		return "", err
	}
	for _, model := range models.Models {
		if id := firstNonEmpty(model.Meta.ID, model.ID); id != "" {
			return id, nil
		}
	}

	providerID, err := resolveOrCreateProvider(t, organizationID)
	if err != nil {
		return "", err
	}

	var created struct {
		Model struct {
			ID   string `json:"id"`
			Meta struct {
				ID string `json:"id"`
			} `json:"meta"`
		} `json:"model"`
	}
	stamp := time.Now().UnixNano()
	if err := gatewayCall(t, "LLMGateway/CreateModel", map[string]any{
		"name":           fmt.Sprintf("TF Acc Model %d", stamp),
		"llmProviderId":  providerID,
		"remoteName":     "gpt-4o-mini",
		"organizationId": organizationID,
	}, &created); err != nil {
		return "", err
	}
	id := firstNonEmpty(created.Model.Meta.ID, created.Model.ID)
	if id == "" {
		return "", fmt.Errorf("CreateModel returned no model id")
	}
	return id, nil
}

func resolveOrCreateProvider(t *testing.T, organizationID string) (string, error) {
	t.Helper()
	var providers struct {
		LLMProviders []struct {
			ID   string `json:"id"`
			Meta struct {
				ID string `json:"id"`
			} `json:"meta"`
		} `json:"llmProviders"`
	}
	if err := gatewayCall(t, "LLMGateway/ListLLMProviders", map[string]any{
		"organizationId": organizationID,
		"pageSize":       200,
		"pageToken":      "",
	}, &providers); err != nil {
		return "", err
	}
	for _, provider := range providers.LLMProviders {
		if id := firstNonEmpty(provider.Meta.ID, provider.ID); id != "" {
			return id, nil
		}
	}

	var created struct {
		LLMProvider struct {
			ID   string `json:"id"`
			Meta struct {
				ID string `json:"id"`
			} `json:"meta"`
		} `json:"llmProvider"`
	}
	stamp := time.Now().UnixNano()
	if err := gatewayCall(t, "LLMGateway/CreateLLMProvider", map[string]any{
		"endpoint":       fmt.Sprintf("https://llm.e2e.agyn.dev/%d", stamp),
		"authMethod":     "AUTH_METHOD_BEARER",
		"token":          fmt.Sprintf("tf-acc-token-%d", stamp),
		"organizationId": organizationID,
	}, &created); err != nil {
		return "", err
	}
	id := firstNonEmpty(created.LLMProvider.Meta.ID, created.LLMProvider.ID)
	if id == "" {
		return "", fmt.Errorf("CreateLLMProvider returned no provider id")
	}
	return id, nil
}

// The Gateway is answering by the time a suite runs, but the LLM service behind
// it can still be rolling: a 503 here is a moment, not a verdict.
func gatewayCall(t *testing.T, method string, payload map[string]any, out any) error {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := strings.TrimSuffix(requireEnv(t, "AGYN_BASE_URL"), "/") + "/agynio.api.gateway.v1." + method
	token := requireAPIToken(t)
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}

	deadline := time.Now().Add(2 * time.Minute)
	for {
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Connect-Protocol-Version", "1")
		request.Header.Set("Authorization", "Bearer "+token)

		response, err := client.Do(request)
		if err == nil {
			raw, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr != nil {
				return readErr
			}
			if response.StatusCode < 500 {
				if response.StatusCode != http.StatusOK {
					return fmt.Errorf("%s returned %d: %s", method, response.StatusCode, strings.TrimSpace(string(raw)))
				}
				return json.Unmarshal(raw, out)
			}
			err = fmt.Errorf("%s returned %d: %s", method, response.StatusCode, strings.TrimSpace(string(raw)))
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(5 * time.Second)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
