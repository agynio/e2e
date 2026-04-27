//go:build e2e

package tests

import (
	"fmt"
	"os"
	"strings"
)

func requireEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("Missing required environment variable %s", key))
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		panic(fmt.Sprintf("Missing required environment variable %s", key))
	}
	return trimmed
}

func envOrDefault(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
