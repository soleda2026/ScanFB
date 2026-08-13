package main

import (
	"bytes"
	"encoding/json"
	"os"
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

func TestHelperProcessWatchedGroupsRoundTrip(t *testing.T) {
	helperPath := t.TempDir() + "/scanfb-bridge-helper"
	build := exec.Command("go", "build", "-o", helperPath, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build helper failed: %v; output=%q", err, output)
	}

	command := exec.Command(helperPath)
	command.Stdin = bytes.NewBufferString(`{"schema_version":2,"operation":"watched_groups_list"}`)
	temporaryConfig := t.TempDir()
	command.Env = append(os.Environ(), "HOME="+temporaryConfig, "XDG_CONFIG_HOME="+temporaryConfig)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("helper failed: %v; stderr=%q", err, stderr.String())
	}

	var response struct {
		SchemaVersion int    `json:"schema_version"`
		Operation     string `json:"operation"`
		Status        string `json:"status"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &response); err != nil {
		t.Fatalf("stdout is not JSON response: %v; stdout=%q", err, stdout.String())
	}
	if response.SchemaVersion != 2 || response.Operation != "watched_groups_list" || response.Status != "ok" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestHelperProcessPreparedGroupScanRoundTrip(t *testing.T) {
	helperPath := t.TempDir() + "/scanfb-bridge-helper"
	build := exec.Command("go", "build", "-o", helperPath, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build helper failed: %v; output=%q", err, output)
	}
	temporaryConfig := t.TempDir()
	environment := append(os.Environ(), "HOME="+temporaryConfig, "XDG_CONFIG_HOME="+temporaryConfig)

	add := exec.Command(helperPath)
	add.Env = environment
	add.Stdin = bytes.NewBufferString(`{"schema_version":2,"operation":"watched_groups_add","new_group":{"id":"group-a","name":"Group A","canonical_url":"https://www.facebook.com/groups/group-a","created_at":"2026-08-13T08:00:00+07:00"}}`)
	if output, err := add.CombinedOutput(); err != nil {
		t.Fatalf("add group failed: %v; output=%q", err, output)
	}

	command := exec.Command(helperPath)
	command.Env = environment
	command.Stdin = bytes.NewBufferString(`{"schema_version":1,"operation":"prepared_group_scan","group_id":"group-a","scan_id":"scan-1","attempt_id":"attempt-1","action_at":"2026-08-13T10:00:00+07:00","prepared_snapshot":{"schema_version":1,"posts":[{"post_id":"post-1","post_url":"","author":{"facebook_user_id":"buyer-1","canonical_profile_url":"","username":"","display_name":"Buyer One"},"body":"Can mua MacBook tai HCM","created_at":"2026-08-13T09:00:00+07:00"}]}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("prepared scan failed: %v; stderr=%q", err, stderr.String())
	}

	var response struct {
		SchemaVersion      int    `json:"schema_version"`
		Operation          string `json:"operation"`
		Status             string `json:"status"`
		AttemptStatus      string `json:"attempt_status"`
		CollectedPostCount int    `json:"collected_post_count"`
		IncludedPostCount  int    `json:"included_post_count"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &response); err != nil {
		t.Fatalf("stdout is not JSON response: %v; stdout=%q", err, stdout.String())
	}
	if response.SchemaVersion != 1 || response.Operation != "prepared_group_scan" ||
		response.Status != "ok" || response.AttemptStatus != "succeeded" ||
		response.CollectedPostCount != 1 || response.IncludedPostCount != 1 {
		t.Fatalf("unexpected response: %#v", response)
	}
}
