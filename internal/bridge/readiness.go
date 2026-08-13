package bridge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/soleda2026/ScanFB/internal/orchestration"
)

const (
	SchemaVersion      = 1
	OperationReadiness = "core_readiness"
	StatusReady        = "ready"
	StatusError        = "error"
	CoreIdentity       = "scanfb-core"

	MaxRequestBytes    = 1024
	MaxResponseBytes   = 1024
	MaxDiagnosticBytes = 512
)

var (
	ErrMalformedRequest         = errors.New("malformed request")
	ErrUnsupportedSchemaVersion = errors.New("unsupported schema version")
	ErrUnsupportedOperation     = errors.New("unsupported operation")
	ErrResponseTooLarge         = errors.New("response too large")
)

type ReadinessRequest struct {
	SchemaVersion int    `json:"schema_version"`
	Operation     string `json:"operation"`
}

type ReadinessResponse struct {
	SchemaVersion   int    `json:"schema_version"`
	ReadinessStatus string `json:"readiness_status"`
	CoreIdentity    string `json:"core_identity"`
}

type requestEnvelope struct {
	Operation string `json:"operation"`
}

func ReadyResponse() ReadinessResponse {
	return ReadinessResponse{
		SchemaVersion:   SchemaVersion,
		ReadinessStatus: StatusReady,
		CoreIdentity:    CoreIdentity,
	}
}

func ErrorResponse() ReadinessResponse {
	return ReadinessResponse{
		SchemaVersion:   SchemaVersion,
		ReadinessStatus: StatusError,
		CoreIdentity:    CoreIdentity,
	}
}

func DecodeRequest(reader io.Reader) (ReadinessRequest, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, MaxRequestBytes+1))
	if err != nil {
		return ReadinessRequest{}, fmt.Errorf("%w: read failed", ErrMalformedRequest)
	}
	if len(payload) == 0 || len(payload) > MaxRequestBytes {
		return ReadinessRequest{}, ErrMalformedRequest
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var request ReadinessRequest
	if err := decoder.Decode(&request); err != nil {
		return ReadinessRequest{}, ErrMalformedRequest
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ReadinessRequest{}, ErrMalformedRequest
	}

	return request, nil
}

func HandleReadiness(request ReadinessRequest) (ReadinessResponse, error) {
	if request.SchemaVersion != SchemaVersion {
		return ErrorResponse(), ErrUnsupportedSchemaVersion
	}
	if request.Operation != OperationReadiness {
		return ErrorResponse(), ErrUnsupportedOperation
	}

	return ReadyResponse(), nil
}

func EncodeResponse(response ReadinessResponse) ([]byte, error) {
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	if len(payload)+1 > MaxResponseBytes {
		return nil, ErrResponseTooLarge
	}

	return append(payload, '\n'), nil
}

func ServeReadiness(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	request, err := DecodeRequest(stdin)
	if err != nil {
		writeDiagnostic(stderr, "request rejected")
		writeResponse(stdout, ErrorResponse())
		return 2
	}

	response, err := HandleReadiness(request)
	if err != nil {
		writeDiagnostic(stderr, "request rejected")
		writeResponse(stdout, response)
		return 2
	}

	if err := writeResponse(stdout, response); err != nil {
		writeDiagnostic(stderr, "response write failed")
		return 3
	}

	return 0
}

// Serve dispatches one bounded helper request to an explicitly supported operation.
func Serve(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return ServeWithWatchedGroupRepositoryFactory(stdin, stdout, stderr, productionWatchedGroupRepositoryFactory)
}

func ServeWithWatchedGroupRepositoryFactory(stdin io.Reader, stdout io.Writer, stderr io.Writer, factory WatchedGroupRepositoryFactory) int {
	payload, err := io.ReadAll(io.LimitReader(stdin, MaxBridgeDispatchRequestBytes+1))
	if err != nil || len(payload) == 0 || len(payload) > MaxBridgeDispatchRequestBytes {
		writeDiagnostic(stderr, "request rejected")
		return 2
	}

	var envelope requestEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		writeDiagnostic(stderr, "request rejected")
		return 2
	}
	if isWatchedGroupsOperation(envelope.Operation) {
		return ServeWatchedGroups(bytes.NewReader(payload), stdout, stderr, factory)
	}
	if envelope.Operation == OperationPreparedGroupScan {
		return ServePreparedGroupScan(bytes.NewReader(payload), stdout, stderr, factory, orchestration.RunOneGroupScan)
	}
	return ServeReadiness(bytes.NewReader(payload), stdout, stderr)
}

func writeResponse(writer io.Writer, response ReadinessResponse) error {
	payload, err := EncodeResponse(response)
	if err != nil {
		return err
	}
	_, err = writer.Write(payload)
	return err
}

func writeDiagnostic(writer io.Writer, message string) {
	sanitized := boundedDiagnostic(message)
	if sanitized == "" {
		return
	}
	_, _ = fmt.Fprintln(writer, sanitized)
}

func boundedDiagnostic(message string) string {
	trimmed := strings.TrimSpace(message)
	if len(trimmed) <= MaxDiagnosticBytes {
		return trimmed
	}
	return trimmed[:MaxDiagnosticBytes]
}
