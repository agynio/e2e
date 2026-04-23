//go:build e2e && (svc_metering || smoke)

package tests

import (
	"testing"

	meteringv1 "github.com/agynio/e2e/suites/go-core/.gen/go/agynio/api/metering/v1"
)

func newMeteringClient(t *testing.T) meteringv1.MeteringServiceClient {
	t.Helper()
	conn := dialGRPC(t, meteringAddr)
	return meteringv1.NewMeteringServiceClient(conn)
}
