package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestBreadcrumbLoggerReportsTestDurations(t *testing.T) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	now := time.Date(2026, 5, 28, 14, 0, 0, 0, time.UTC)
	logger := newBreadcrumbLogger(&out, &stderr, func() time.Time { return now })

	logger.handle(testEvent{
		Time:    now,
		Action:  "run",
		Package: "github.com/agynio/e2e/suites/go-core/tests",
		Test:    "TestExample",
	})
	now = now.Add(3 * time.Second)
	logger.handle(testEvent{
		Time:    now,
		Action:  "pass",
		Package: "github.com/agynio/e2e/suites/go-core/tests",
		Test:    "TestExample",
		Elapsed: 3,
	})

	logs := stderr.String()
	for _, expected := range []string{
		"package_start package=github.com/agynio/e2e/suites/go-core/tests",
		"test_start package=github.com/agynio/e2e/suites/go-core/tests test=TestExample",
		"test_pass package=github.com/agynio/e2e/suites/go-core/tests test=TestExample elapsed=3s",
	} {
		if !strings.Contains(logs, expected) {
			t.Fatalf("missing log %q in:\n%s", expected, logs)
		}
	}
}

func TestScanEventsPassesThroughPlainOutput(t *testing.T) {
	input := strings.NewReader("plain line\n{\"Action\":\"run\",\"Package\":\"pkg\",\"Test\":\"TestOne\"}\n")
	var out bytes.Buffer
	events := make(chan testEvent, 1)

	if err := scanEvents(input, events, &out); err != nil {
		t.Fatalf("scan events: %v", err)
	}
	close(events)

	if got := out.String(); got != "plain line\n" {
		t.Fatalf("plain output mismatch: got %q", got)
	}
	event := <-events
	if event.Package != "pkg" || event.Test != "TestOne" || event.Action != "run" {
		t.Fatalf("event mismatch: %#v", event)
	}
}
