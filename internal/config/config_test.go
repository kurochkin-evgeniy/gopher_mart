package config

import (
	"os"
	"testing"
)

func TestValueFromFlagOrEnv(t *testing.T) {
	t.Setenv("RUN_ADDRESS", "env:9000")

	if got := valueFromFlagOrEnv("flag:8080", "RUN_ADDRESS", "default"); got != "flag:8080" {
		t.Fatalf("flag priority: got %q", got)
	}
	if got := valueFromFlagOrEnv("", "RUN_ADDRESS", "default"); got != "env:9000" {
		t.Fatalf("env fallback: got %q", got)
	}
	if got := valueFromFlagOrEnv("", "UNSET_VAR", "default"); got != "default" {
		t.Fatalf("default: got %q", got)
	}
}

func TestValueFromFlagOrEnvEmptyEnv(t *testing.T) {
	_ = os.Unsetenv("RUN_ADDRESS")
	if got := valueFromFlagOrEnv("", "RUN_ADDRESS", "localhost:8080"); got != "localhost:8080" {
		t.Fatalf("got %q", got)
	}
}
