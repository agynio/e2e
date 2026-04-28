//go:build e2e

package tests

import "testing"

const (
	zitiManagementAddrEnvKey  = "ZITI_MANAGEMENT_ADDRESS"
	defaultZitiManagementAddr = "ziti-management:50051"
)

func zitiManagementAddr(t *testing.T) string {
	t.Helper()
	return envOrDefault(zitiManagementAddrEnvKey, defaultZitiManagementAddr)
}
