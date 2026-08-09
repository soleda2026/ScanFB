package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"
)

func TestHelperProcessReadinessRoundTrip(t *testing.T) {
	command := exec.Command("go", "run", ".")
	command.Stdin = bytes.NewBufferString(`{"schema_version":1,"operation":"core_readiness"}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("helper failed: %v; stderr=%q", err, stderr.String())
	}

	var response struct {
		SchemaVersion   int    `json:"schema_version"`
		ReadinessStatus string `json:"readiness_status"`
		CoreIdentity    string `json:"core_identity"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &response); err != nil {
		t.Fatalf("stdout is not JSON response: %v; stdout=%q", err, stdout.String())
	}

	if response.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", response.SchemaVersion)
	}
	if response.ReadinessStatus != "ready" {
		t.Fatalf("ReadinessStatus = %q, want ready", response.ReadinessStatus)
	}
	if response.CoreIdentity != "scanfb-core" {
		t.Fatalf("CoreIdentity = %q, want scanfb-core", response.CoreIdentity)
	}
}
