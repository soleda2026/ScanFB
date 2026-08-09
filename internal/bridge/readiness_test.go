package bridge

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestValidReadinessRequest(t *testing.T) {
	request, err := DecodeRequest(strings.NewReader(`{"schema_version":1,"operation":"core_readiness"}`))
	if err != nil {
		t.Fatalf("DecodeRequest returned error: %v", err)
	}

	response, err := HandleReadiness(request)
	if err != nil {
		t.Fatalf("HandleReadiness returned error: %v", err)
	}

	if response.ReadinessStatus != StatusReady {
		t.Fatalf("ReadinessStatus = %q, want %q", response.ReadinessStatus, StatusReady)
	}
}

func TestExactSchemaVersion(t *testing.T) {
	response := ReadyResponse()

	if response.SchemaVersion != SchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", response.SchemaVersion, SchemaVersion)
	}
}

func TestExactOperation(t *testing.T) {
	request, err := DecodeRequest(strings.NewReader(`{"schema_version":1,"operation":"core_readiness"}`))
	if err != nil {
		t.Fatalf("DecodeRequest returned error: %v", err)
	}

	if request.Operation != OperationReadiness {
		t.Fatalf("Operation = %q, want %q", request.Operation, OperationReadiness)
	}
}

func TestDeterministicReadinessResponse(t *testing.T) {
	first, err := EncodeResponse(ReadyResponse())
	if err != nil {
		t.Fatalf("EncodeResponse returned error: %v", err)
	}
	second, err := EncodeResponse(ReadyResponse())
	if err != nil {
		t.Fatalf("EncodeResponse returned error: %v", err)
	}

	if string(first) != string(second) {
		t.Fatalf("response is not deterministic:\nfirst: %s\nsecond: %s", first, second)
	}
}

func TestCoreIdentityStableValue(t *testing.T) {
	if ReadyResponse().CoreIdentity != CoreIdentity {
		t.Fatalf("CoreIdentity = %q, want %q", ReadyResponse().CoreIdentity, CoreIdentity)
	}
	if CoreIdentity != "scanfb-core" {
		t.Fatalf("CoreIdentity changed to %q", CoreIdentity)
	}
}

func TestMalformedJSONRejected(t *testing.T) {
	_, err := DecodeRequest(strings.NewReader(`{"schema_version":`))

	if !errors.Is(err, ErrMalformedRequest) {
		t.Fatalf("err = %v, want ErrMalformedRequest", err)
	}
}

func TestUnsupportedSchemaRejected(t *testing.T) {
	request := ReadinessRequest{SchemaVersion: 2, Operation: OperationReadiness}
	response, err := HandleReadiness(request)

	if !errors.Is(err, ErrUnsupportedSchemaVersion) {
		t.Fatalf("err = %v, want ErrUnsupportedSchemaVersion", err)
	}
	if response.ReadinessStatus != StatusError {
		t.Fatalf("ReadinessStatus = %q, want %q", response.ReadinessStatus, StatusError)
	}
}

func TestUnsupportedOperationRejected(t *testing.T) {
	request := ReadinessRequest{SchemaVersion: SchemaVersion, Operation: "scan_everything"}
	response, err := HandleReadiness(request)

	if !errors.Is(err, ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if response.ReadinessStatus != StatusError {
		t.Fatalf("ReadinessStatus = %q, want %q", response.ReadinessStatus, StatusError)
	}
}

func TestStdoutContainsResponseOnly(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ServeReadiness(strings.NewReader(`{"schema_version":1,"operation":"core_readiness"}`), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty diagnostics for success", stderr.String())
	}

	var response ReadinessResponse
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &response); err != nil {
		t.Fatalf("stdout is not a response: %v; stdout=%q", err, stdout.String())
	}
	if strings.Contains(stdout.String(), "request rejected") {
		t.Fatalf("stdout contains diagnostic text: %q", stdout.String())
	}
}

func TestNoFacebookDatabaseOrPersistenceAccessInReadinessPath(t *testing.T) {
	source, err := os.ReadFile("readiness.go")
	if err != nil {
		t.Fatalf("ReadFile readiness.go: %v", err)
	}

	for _, forbidden := range []string{
		"internal/facebook",
		"internal/persistence",
		"database/sql",
		"sqlite",
		"modernc.org/sqlite",
	} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("readiness path contains forbidden dependency %q", forbidden)
		}
	}
}
