//go:build e2e && svc_agents_orchestrator

package tests

import (
	"context"
	"strings"
	"testing"
)

func TestAgentSimpleHelloProducesTrace(t *testing.T) {
	requireTracingAvailable(t)
	expectedResponse := "Hi! How are you?"
	result := runFullPipelineMessageResponse(t, testLLMEndpointAgn, agnRuntime, "hi", expectedResponse)

	ctx := context.Background()
	tracingClient := newTracingClient(t)
	traceID := discoverTraceID(t, ctx, tracingClient, result.organizationID, result.threadID, result.startTimeMinNs, result.messageText)
	assertTraceSummary(t, ctx, tracingClient, traceID, map[string]int64{
		"invocation.message": 1,
		"llm.call":           1,
	}, 2, result.threadID)

	// agyn.message.kind is deliberately absent. It was set by agn, on the
	// invocation.message it used to record for itself -- and a turn now carries
	// one such span, the one agynd records for the message it fed the CLI. What
	// that span asserts is what the platform knows about the message.
	assertSpanAttributes(t, ctx, tracingClient, traceID, "invocation.message", map[string]string{
		"agyn.message.text": result.messageText,
		"agyn.message.role": "user",
	})
	llmAttrs := assertSpanAttributes(t, ctx, tracingClient, traceID, "llm.call", map[string]string{
		"gen_ai.system":          "openai",
		"agyn.llm.response_text": expectedResponse,
	})
	modelName, ok := llmAttrs["gen_ai.request.model"]
	if !ok || strings.TrimSpace(modelName) == "" {
		t.Fatal("expected gen_ai.request.model to be set")
	}
}

func TestAgentMCPToolsProducesTrace(t *testing.T) {
	requireTracingAvailable(t)
	result := runMCPToolsE2E(t, testLLMEndpointAgn, agnRuntime)

	ctx := context.Background()
	tracingClient := newTracingClient(t)
	traceID := discoverTraceID(t, ctx, tracingClient, result.organizationID, result.threadID, result.startTimeMinNs, result.messageText)
	// Three model turns for two tools, which is what the scripted conversation
	// asks for: a turn that calls create_entities, a turn that calls
	// list_directory once it has the result, and a turn that answers. Two was
	// the count from before agn's spans joined the cycle's trace, when one of
	// the three was landing somewhere else.
	expectedCounts := map[string]int64{
		"invocation.message": 1,
		"llm.call":           3,
		"tool.execution":     2,
	}
	assertTraceSummary(t, ctx, tracingClient, traceID, expectedCounts, 6, result.threadID)

	spans := traceSpans(t, ctx, tracingClient, traceID)
	foundCreate := false
	foundList := false
	for _, span := range spans {
		if span.GetName() != "tool.execution" {
			continue
		}
		attrs := attributesToMap(span.GetAttributes())
		toolName := attrs["agyn.tool.name"]
		if strings.Contains(toolName, "create_entities") {
			foundCreate = true
		}
		if strings.Contains(toolName, "list_directory") {
			foundList = true
		}
	}
	if !foundCreate {
		t.Fatal("expected tool.execution span for create_entities")
	}
	if !foundList {
		t.Fatal("expected tool.execution span for list_directory")
	}
}
