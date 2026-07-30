//go:build e2e && (svc_metering || smoke)

package tests

import (
	"context"
	"testing"
	"time"

	meteringv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/metering/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRecordAndQueryUsage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := newMeteringClient(t)

	orgID := uuid.NewString()
	resourceID := uuid.NewString()
	identityID := uuid.NewString()
	threadID := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Second)

	records := []*meteringv1.UsageRecord{
		{
			OrgId:          orgID,
			IdempotencyKey: uuid.NewString() + "-input",
			Producer:       "e2e",
			Timestamp:      timestamppb.New(now),
			Labels: map[string]string{
				"resource_id":   resourceID,
				"resource":      "model",
				"identity_id":   identityID,
				"identity_type": "user",
				"thread_id":     threadID,
				"kind":          "input",
				"status":        "success",
			},
			Unit:  meteringv1.Unit_UNIT_TOKENS,
			Value: 1_500_000,
		},
		{
			OrgId:          orgID,
			IdempotencyKey: uuid.NewString() + "-output",
			Producer:       "e2e",
			Timestamp:      timestamppb.New(now),
			Labels: map[string]string{
				"resource_id":   resourceID,
				"resource":      "model",
				"identity_id":   identityID,
				"identity_type": "user",
				"thread_id":     threadID,
				"kind":          "output",
				"status":        "success",
			},
			Unit:  meteringv1.Unit_UNIT_TOKENS,
			Value: 2_500_000,
		},
	}

	if _, err := client.Record(ctx, &meteringv1.RecordRequest{Records: records}); err != nil {
		t.Fatalf("record usage: %v", err)
	}

	start := timestamppb.New(now.Add(-time.Minute))
	end := timestamppb.New(now.Add(time.Minute))

	grouped, err := client.QueryUsage(ctx, &meteringv1.QueryUsageRequest{
		OrgId:       orgID,
		Start:       start,
		End:         end,
		Unit:        meteringv1.Unit_UNIT_TOKENS,
		GroupBy:     "kind",
		Granularity: meteringv1.Granularity_GRANULARITY_TOTAL,
	})
	if err != nil {
		t.Fatalf("query grouped usage: %v", err)
	}
	if len(grouped.Buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(grouped.Buckets))
	}

	values := map[string]int64{}
	for _, bucket := range grouped.Buckets {
		values[bucket.GetGroupValue()] = bucket.GetValue()
	}
	if values["input"] != records[0].Value {
		t.Fatalf("expected input value %d, got %d", records[0].Value, values["input"])
	}
	if values["output"] != records[1].Value {
		t.Fatalf("expected output value %d, got %d", records[1].Value, values["output"])
	}

	filtered, err := client.QueryUsage(ctx, &meteringv1.QueryUsageRequest{
		OrgId:        orgID,
		Start:        start,
		End:          end,
		Unit:         meteringv1.Unit_UNIT_TOKENS,
		LabelFilters: map[string]string{"kind": "input"},
		Granularity:  meteringv1.Granularity_GRANULARITY_TOTAL,
	})
	if err != nil {
		t.Fatalf("query filtered usage: %v", err)
	}
	if len(filtered.Buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(filtered.Buckets))
	}
	if filtered.Buckets[0].Value != records[0].Value {
		t.Fatalf("expected filtered value %d, got %d", records[0].Value, filtered.Buckets[0].Value)
	}
}

// Compute is billed per flavor, and UsageEvent stores labels in dedicated
// columns rather than generically — a label the service does not recognise is
// accepted on the wire and then dropped. That failure is silent: Record
// succeeds, the row lands with a NULL, and the usage simply goes missing. Only
// a round trip through the real service catches it.
func TestRecordAndQueryFlavorUsage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := newMeteringClient(t)

	orgID := uuid.NewString()
	runnerID := uuid.NewString()
	workloadID := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Second)

	records := []*meteringv1.UsageRecord{
		{
			OrgId:          orgID,
			IdempotencyKey: uuid.NewString() + "-small",
			Producer:       "e2e",
			Timestamp:      timestamppb.New(now),
			Labels: map[string]string{
				"resource":    "workload",
				"resource_id": workloadID,
				"runner_id":   runnerID,
				"flavor":      "cpu-1x",
			},
			Unit:  meteringv1.Unit_UNIT_FLAVOR_SECONDS,
			Value: 60_000_000,
		},
		{
			OrgId:          orgID,
			IdempotencyKey: uuid.NewString() + "-large",
			Producer:       "e2e",
			Timestamp:      timestamppb.New(now),
			Labels: map[string]string{
				"resource":    "workload",
				"resource_id": uuid.NewString(),
				"runner_id":   runnerID,
				"flavor":      "cpu-4x",
			},
			Unit:  meteringv1.Unit_UNIT_FLAVOR_SECONDS,
			Value: 120_000_000,
		},
	}

	if _, err := client.Record(ctx, &meteringv1.RecordRequest{Records: records}); err != nil {
		t.Fatalf("record flavor usage: %v", err)
	}

	start := timestamppb.New(now.Add(-time.Minute))
	end := timestamppb.New(now.Add(time.Minute))

	// Grouping by flavor is what billing reads. If the label were dropped the
	// buckets would collapse into a single empty group value.
	grouped, err := client.QueryUsage(ctx, &meteringv1.QueryUsageRequest{
		OrgId:       orgID,
		Start:       start,
		End:         end,
		Unit:        meteringv1.Unit_UNIT_FLAVOR_SECONDS,
		GroupBy:     "flavor",
		Granularity: meteringv1.Granularity_GRANULARITY_TOTAL,
	})
	if err != nil {
		t.Fatalf("query usage by flavor: %v", err)
	}
	byFlavor := map[string]int64{}
	for _, bucket := range grouped.Buckets {
		byFlavor[bucket.GetGroupValue()] = bucket.GetValue()
	}
	if byFlavor["cpu-1x"] != records[0].Value {
		t.Fatalf("expected cpu-1x value %d, got %d (buckets=%v)", records[0].Value, byFlavor["cpu-1x"], byFlavor)
	}
	if byFlavor["cpu-4x"] != records[1].Value {
		t.Fatalf("expected cpu-4x value %d, got %d (buckets=%v)", records[1].Value, byFlavor["cpu-4x"], byFlavor)
	}

	// runner_id is the other new column, and it has to filter as well as store.
	filtered, err := client.QueryUsage(ctx, &meteringv1.QueryUsageRequest{
		OrgId:        orgID,
		Start:        start,
		End:          end,
		Unit:         meteringv1.Unit_UNIT_FLAVOR_SECONDS,
		LabelFilters: map[string]string{"runner_id": runnerID, "flavor": "cpu-4x"},
		Granularity:  meteringv1.Granularity_GRANULARITY_TOTAL,
	})
	if err != nil {
		t.Fatalf("query usage filtered by runner and flavor: %v", err)
	}
	if len(filtered.Buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(filtered.Buckets))
	}
	if filtered.Buckets[0].GetValue() != records[1].Value {
		t.Fatalf("expected filtered value %d, got %d", records[1].Value, filtered.Buckets[0].GetValue())
	}
}
