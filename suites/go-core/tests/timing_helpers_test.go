//go:build e2e

package tests

import (
	"testing"
	"time"
)

func startTimingBreadcrumbs(t *testing.T) func(string) {
	t.Helper()
	started := time.Now()
	last := started
	t.Logf("timing: test=%s step=start total=0s", t.Name())
	t.Cleanup(func() {
		t.Logf("timing: test=%s step=finish total=%s", t.Name(), time.Since(started).Round(time.Millisecond))
	})
	return func(step string) {
		t.Helper()
		now := time.Now()
		t.Logf(
			"timing: test=%s step=%s step_elapsed=%s total=%s",
			t.Name(),
			step,
			now.Sub(last).Round(time.Millisecond),
			now.Sub(started).Round(time.Millisecond),
		)
		last = now
	}
}
