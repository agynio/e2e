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
