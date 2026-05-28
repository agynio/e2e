package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

type testEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Elapsed float64   `json:"Elapsed"`
	Output  string    `json:"Output"`
}

type activeTest struct {
	Package string
	Test    string
	Started time.Time
}

type breadcrumbLogger struct {
	now          func() time.Time
	started      time.Time
	active       map[string]activeTest
	seenPackages map[string]struct{}
	out          io.Writer
	err          io.Writer
}

func newBreadcrumbLogger(out, err io.Writer, now func() time.Time) *breadcrumbLogger {
	started := now()
	return &breadcrumbLogger{
		now:          now,
		started:      started,
		active:       make(map[string]activeTest),
		seenPackages: make(map[string]struct{}),
		out:          out,
		err:          err,
	}
}

func main() {
	interval := flag.Duration("interval", 30*time.Second, "active-test breadcrumb interval")
	goTestTimeout := flag.String("timeout", "", "go test package timeout for log context")
	flag.Parse()

	logger := newBreadcrumbLogger(os.Stdout, os.Stderr, time.Now)
	logger.logSuiteStart(*goTestTimeout, *interval)

	if err := logger.run(os.Stdin, *interval); err != nil {
		fmt.Fprintf(os.Stderr, "[go-test-timing] error=%q\n", err.Error())
		os.Exit(1)
	}
}

func (l *breadcrumbLogger) run(input io.Reader, interval time.Duration) error {
	events := make(chan testEvent)
	done := make(chan error, 1)

	go func() {
		defer close(events)
		done <- scanEvents(input, events, l.out)
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				l.logActive("finish")
				return <-done
			}
			l.handle(event)
		case <-ticker.C:
			l.logActive("running")
		}
	}
}

func scanEvents(input io.Reader, events chan<- testEvent, out io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		var event testEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			fmt.Fprintln(out, line)
			continue
		}
		events <- event
	}
	return scanner.Err()
}

func (l *breadcrumbLogger) handle(event testEvent) {
	if event.Output != "" {
		fmt.Fprint(l.out, event.Output)
	}
	if event.Package != "" {
		if _, ok := l.seenPackages[event.Package]; !ok {
			l.seenPackages[event.Package] = struct{}{}
			l.logf("package_start package=%s suite_elapsed=%s", event.Package, l.sinceStart())
		}
	}
	if event.Test == "" {
		return
	}

	switch event.Action {
	case "run", "cont":
		l.active[eventKey(event.Package, event.Test)] = activeTest{
			Package: event.Package,
			Test:    event.Test,
			Started: l.eventTime(event),
		}
		l.logf("test_start package=%s test=%s suite_elapsed=%s", event.Package, event.Test, l.sinceStart())
	case "pass", "fail", "skip":
		key := eventKey(event.Package, event.Test)
		elapsed := l.elapsedFor(event, key)
		delete(l.active, key)
		l.logf("test_%s package=%s test=%s elapsed=%s suite_elapsed=%s", event.Action, event.Package, event.Test, elapsed, l.sinceStart())
	}
}

func (l *breadcrumbLogger) logSuiteStart(timeout string, interval time.Duration) {
	if strings.TrimSpace(timeout) == "" {
		timeout = "go-default"
	}
	l.logf("suite_start timeout=%s interval=%s", timeout, interval)
}

func (l *breadcrumbLogger) logActive(state string) {
	if len(l.active) == 0 {
		l.logf("suite_%s active=0 suite_elapsed=%s", state, l.sinceStart())
		return
	}

	entries := make([]activeTest, 0, len(l.active))
	for _, entry := range l.active {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Package == entries[j].Package {
			return entries[i].Test < entries[j].Test
		}
		return entries[i].Package < entries[j].Package
	})
	for _, entry := range entries {
		l.logf(
			"suite_%s active=%d package=%s test=%s elapsed=%s suite_elapsed=%s",
			state,
			len(entries),
			entry.Package,
			entry.Test,
			l.now().Sub(entry.Started).Round(time.Millisecond),
			l.sinceStart(),
		)
	}
}

func (l *breadcrumbLogger) elapsedFor(event testEvent, key string) time.Duration {
	if event.Elapsed > 0 {
		return time.Duration(event.Elapsed * float64(time.Second)).Round(time.Millisecond)
	}
	active, ok := l.active[key]
	if !ok {
		return 0
	}
	return l.eventTime(event).Sub(active.Started).Round(time.Millisecond)
}

func (l *breadcrumbLogger) eventTime(event testEvent) time.Time {
	if !event.Time.IsZero() {
		return event.Time
	}
	return l.now()
}

func (l *breadcrumbLogger) sinceStart() time.Duration {
	return l.now().Sub(l.started).Round(time.Millisecond)
}

func (l *breadcrumbLogger) logf(format string, args ...any) {
	fmt.Fprintf(l.err, "[go-test-timing] "+format+"\n", args...)
}

func eventKey(packageName, testName string) string {
	return packageName + "\x00" + testName
}
