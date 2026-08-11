package facebook

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseSafariRenderedDOMResponsePreservesSnapshotExactly(t *testing.T) {
	capturedAt := phase10B1CapturedAt()
	wire := phase10B2dWireResponse()
	wire.PageTitle = "Synthetic rendered page"
	wire.RenderedDOM = "<html><body><main>Rendered</main></body></html>"

	snapshot, err := parseSafariRenderedDOMResponse(phase10B2dMarshalResponse(t, wire), capturedAt)
	if err != nil {
		t.Fatalf("parseSafariRenderedDOMResponse() error = %v", err)
	}
	if snapshot.PageURL != wire.PageURL {
		t.Fatalf("PageURL = %q, want %q", snapshot.PageURL, wire.PageURL)
	}
	if snapshot.PageTitle != wire.PageTitle {
		t.Fatalf("PageTitle = %q, want %q", snapshot.PageTitle, wire.PageTitle)
	}
	if snapshot.RenderedDOM != wire.RenderedDOM {
		t.Fatalf("RenderedDOM = %q, want %q", snapshot.RenderedDOM, wire.RenderedDOM)
	}
	if !snapshot.CapturedAt.Equal(capturedAt) {
		t.Fatalf("CapturedAt = %v, want %v", snapshot.CapturedAt, capturedAt)
	}
}

func TestParseSafariRenderedDOMResponseFailsClosed(t *testing.T) {
	tests := []struct {
		name       string
		response   func(t *testing.T) []byte
		capturedAt time.Time
		wantErr    error
	}{
		{
			name:       "empty response",
			response:   func(t *testing.T) []byte { return nil },
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrMalformedSafariRenderedDOMResponse,
		},
		{
			name:       "invalid JSON",
			response:   func(t *testing.T) []byte { return []byte("not-json") },
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrMalformedSafariRenderedDOMResponse,
		},
		{
			name: "unknown field",
			response: func(t *testing.T) []byte {
				return []byte(`{"schema_version":1,"page_url":"https://example.test","page_title":"","rendered_dom":"<html></html>","extra":true}`)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrMalformedSafariRenderedDOMResponse,
		},
		{
			name: "trailing response",
			response: func(t *testing.T) []byte {
				return append(phase10B2dValidResponse(t), []byte(` {}`)...)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrMalformedSafariRenderedDOMResponse,
		},
		{
			name: "unsupported response schema",
			response: func(t *testing.T) []byte {
				wire := phase10B2dWireResponse()
				wire.SchemaVersion++
				return phase10B2dMarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrUnsupportedSafariRenderedDOMVersion,
		},
		{
			name: "empty rendered DOM",
			response: func(t *testing.T) []byte {
				wire := phase10B2dWireResponse()
				wire.RenderedDOM = " \n\t "
				return phase10B2dMarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrEmptySafariRenderedDOM,
		},
		{
			name: "non HTTPS URL",
			response: func(t *testing.T) []byte {
				wire := phase10B2dWireResponse()
				wire.PageURL = "http://example.test/page"
				return phase10B2dMarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrUnsupportedSafariActiveTabURL,
		},
		{
			name: "userinfo URL",
			response: func(t *testing.T) []byte {
				wire := phase10B2dWireResponse()
				wire.PageURL = "https://user:secret@example.test/page"
				return phase10B2dMarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrUnsupportedSafariActiveTabURL,
		},
		{
			name: "oversized rendered DOM",
			response: func(t *testing.T) []byte {
				wire := phase10B2dWireResponse()
				wire.RenderedDOM = strings.Repeat("x", SafariRenderedDOMMaxBytes+1)
				return phase10B2dMarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrOversizedSafariRenderedDOM,
		},
		{
			name: "oversized transport response",
			response: func(t *testing.T) []byte {
				return []byte(strings.Repeat("x", safariRenderedDOMResponseMaxBytes+1))
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrOversizedSafariRenderedDOM,
		},
		{
			name:       "missing captured at",
			response:   phase10B2dValidResponse,
			capturedAt: time.Time{},
			wantErr:    ErrInvalidSafariCapturedAt,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snapshot, err := parseSafariRenderedDOMResponse(tc.response(t), tc.capturedAt)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("parseSafariRenderedDOMResponse() error = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(snapshot, SafariRenderedDOMSnapshot{}) {
				t.Fatalf("invalid response returned snapshot %#v", snapshot)
			}
		})
	}
}

func TestParseSafariRenderedDOMResponseAcceptsExactDecodedBoundary(t *testing.T) {
	wire := phase10B2dWireResponse()
	wire.RenderedDOM = strings.Repeat("<", SafariRenderedDOMMaxBytes)
	response := phase10B2dMarshalResponse(t, wire)
	if len(response) <= SafariRenderedDOMMaxBytes {
		t.Fatalf("fixture did not exercise JSON escaping: encoded bytes = %d", len(response))
	}

	snapshot, err := parseSafariRenderedDOMResponse(response, phase10B1CapturedAt())
	if err != nil {
		t.Fatalf("parseSafariRenderedDOMResponse() boundary error = %v", err)
	}
	if len([]byte(snapshot.RenderedDOM)) != SafariRenderedDOMMaxBytes {
		t.Fatalf("decoded rendered DOM bytes = %d", len([]byte(snapshot.RenderedDOM)))
	}
}

func TestSafariRenderedDOMResponseEnvelopeIsFiniteAndConservative(t *testing.T) {
	const wantResponseMaxBytes = 50_397_184
	if SafariRenderedDOMMaxBytes != 8_388_608 {
		t.Fatalf("decoded rendered DOM limit = %d, want 8 MiB", SafariRenderedDOMMaxBytes)
	}
	if safariRenderedDOMResponseMaxBytes != wantResponseMaxBytes {
		t.Fatalf("transport envelope = %d, want %d", safariRenderedDOMResponseMaxBytes, wantResponseMaxBytes)
	}

	wire := phase10B2dWireResponse()
	wire.RenderedDOM = strings.Repeat("<", SafariRenderedDOMMaxBytes)
	response := phase10B2dMarshalResponse(t, wire)
	if len(response) > safariRenderedDOMResponseMaxBytes {
		t.Fatalf("worst-case escaped fixture bytes = %d, envelope = %d", len(response), safariRenderedDOMResponseMaxBytes)
	}

	stdoutMaxBytes, stderrMaxBytes := (osaScriptCommandRunner{stdoutMaxBytes: safariRenderedDOMResponseMaxBytes}).outputLimits()
	if stdoutMaxBytes != safariRenderedDOMResponseMaxBytes || stderrMaxBytes != safariDiagnosticMaxBytes {
		t.Fatalf("runner limits = (%d, %d)", stdoutMaxBytes, stderrMaxBytes)
	}
}

func TestParseSafariRenderedDOMResponseIsDeterministic(t *testing.T) {
	response := phase10B2dValidResponse(t)
	capturedAt := phase10B1CapturedAt()

	first, err := parseSafariRenderedDOMResponse(response, capturedAt)
	if err != nil {
		t.Fatalf("first parse error = %v", err)
	}
	second, err := parseSafariRenderedDOMResponse(response, capturedAt)
	if err != nil {
		t.Fatalf("second parse error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same response changed: first=%#v second=%#v", first, second)
	}
}

func TestAcquireSafariActiveTabRenderedDOMUsesFixedDirectCommand(t *testing.T) {
	var executable string
	var arguments []string
	runner := safariCommandRunnerFunc(func(_ context.Context, path string, args []string) (safariCommandResult, error) {
		executable = path
		arguments = append([]string(nil), args...)
		return safariCommandResult{
			stdout: phase10B2dValidResponse(t),
			stderr: []byte("bounded diagnostic that must not corrupt stdout"),
		}, nil
	})

	snapshot, err := acquireSafariActiveTabRenderedDOM(context.Background(), phase10B1CapturedAt(), time.Second, runner)
	if err != nil {
		t.Fatalf("acquireSafariActiveTabRenderedDOM() error = %v", err)
	}
	if executable != "/usr/bin/osascript" {
		t.Fatalf("executable = %q", executable)
	}
	wantArguments := []string{"-l", "JavaScript", "-e", safariRenderedDOMJXAScript}
	if !reflect.DeepEqual(arguments, wantArguments) {
		t.Fatalf("arguments = %#v, want fixed JXA arguments", arguments)
	}
	if snapshot.RenderedDOM != "<html><body>synthetic</body></html>" {
		t.Fatalf("RenderedDOM = %q", snapshot.RenderedDOM)
	}

	var publicAPI func(context.Context, time.Time) (SafariRenderedDOMSnapshot, error) = AcquireSafariActiveTabRenderedDOM
	if publicAPI == nil {
		t.Fatal("AcquireSafariActiveTabRenderedDOM is nil")
	}
}

func TestSafariRenderedDOMJXAOwnsOneCurrentTabAndFixedPageScript(t *testing.T) {
	required := []string{
		`Application("Safari")`,
		"safari.running()",
		"safari.windows()",
		"windows[0].currentTab()",
		"tab.url()",
		"tab.name()",
		"safari.doJavaScript('" + safariRenderedDOMPageScript + "', {in: tab})",
		"JSON.stringify",
	}
	for _, fragment := range required {
		if !strings.Contains(safariRenderedDOMJXAScript, fragment) {
			t.Errorf("JXA script missing %q", fragment)
		}
	}
	if strings.Count(safariRenderedDOMJXAScript, "doJavaScript(") != 1 {
		t.Fatalf("doJavaScript call count = %d, want 1", strings.Count(safariRenderedDOMJXAScript, "doJavaScript("))
	}
	if safariRenderedDOMPageScript != `document.documentElement ? document.documentElement.outerHTML : ""` {
		t.Fatalf("page-side script = %q", safariRenderedDOMPageScript)
	}
}

func TestAcquireSafariActiveTabRenderedDOMMapsProcessFailuresExplicitly(t *testing.T) {
	tests := []struct {
		name    string
		result  safariCommandResult
		runErr  error
		wantErr error
	}{
		{name: "process start failure", runErr: errors.New("synthetic start failure"), wantErr: ErrSafariOSAStartFailure},
		{name: "automation permission denied", result: safariCommandResult{exitCode: 1, stderr: []byte("Not authorized to send Apple events to Safari. (-1743)")}, wantErr: ErrSafariAutomationPermissionDenied},
		{name: "JavaScript from Apple Events disabled", result: safariCommandResult{exitCode: 1, stderr: []byte("JavaScript from Apple Events is turned off")}, wantErr: ErrSafariJavaScriptFromAppleEventsDisabled},
		{name: "Safari not running", result: safariCommandResult{exitCode: 1, stderr: []byte("Error: SCANFB_SAFARI_NOT_RUNNING")}, wantErr: ErrSafariNotRunning},
		{name: "no active window", result: safariCommandResult{exitCode: 1, stderr: []byte("Error: SCANFB_SAFARI_NO_ACTIVE_WINDOW")}, wantErr: ErrSafariNoActiveWindow},
		{name: "no active tab", result: safariCommandResult{exitCode: 1, stderr: []byte("Error: SCANFB_SAFARI_NO_ACTIVE_TAB")}, wantErr: ErrSafariNoActiveTab},
		{name: "unknown nonzero exit", result: safariCommandResult{exitCode: 1, stderr: []byte("synthetic private diagnostic")}, wantErr: ErrSafariAcquisitionNonzeroExit},
		{name: "bounded stdout exceeded", result: safariCommandResult{stdoutExceeded: true}, wantErr: ErrOversizedSafariRenderedDOM},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runner := safariCommandRunnerFunc(func(context.Context, string, []string) (safariCommandResult, error) {
				return tc.result, tc.runErr
			})
			snapshot, err := acquireSafariActiveTabRenderedDOM(context.Background(), phase10B1CapturedAt(), time.Second, runner)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("acquireSafariActiveTabRenderedDOM() error = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(snapshot, SafariRenderedDOMSnapshot{}) {
				t.Fatalf("failed acquisition returned snapshot %#v", snapshot)
			}
			if strings.Contains(err.Error(), "synthetic private diagnostic") {
				t.Fatalf("raw stderr leaked through error %q", err)
			}
		})
	}
}

func TestAcquireSafariActiveTabRenderedDOMMapsTimeoutAndCancellation(t *testing.T) {
	blockingRunner := safariCommandRunnerFunc(func(ctx context.Context, _ string, _ []string) (safariCommandResult, error) {
		<-ctx.Done()
		return safariCommandResult{}, ctx.Err()
	})

	if _, err := acquireSafariActiveTabRenderedDOM(context.Background(), phase10B1CapturedAt(), time.Millisecond, blockingRunner); !errors.Is(err, ErrSafariAcquisitionTimeout) {
		t.Fatalf("timeout error = %v, want %v", err, ErrSafariAcquisitionTimeout)
	}

	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := acquireSafariActiveTabRenderedDOM(canceledContext, phase10B1CapturedAt(), time.Second, blockingRunner); !errors.Is(err, ErrSafariAcquisitionCanceled) {
		t.Fatalf("cancellation error = %v, want %v", err, ErrSafariAcquisitionCanceled)
	}
}

func TestSafariRenderedDOMSnapshotContainsNoSensitiveBrowserState(t *testing.T) {
	typeOfSnapshot := reflect.TypeOf(SafariRenderedDOMSnapshot{})
	for i := 0; i < typeOfSnapshot.NumField(); i++ {
		field := strings.ToLower(typeOfSnapshot.Field(i).Name)
		for _, forbidden := range []string{"cookie", "credential", "session", "token", "header", "storage", "profile", "history", "cache", "windowid", "tabid", "processid"} {
			if strings.Contains(field, forbidden) {
				t.Fatalf("SafariRenderedDOMSnapshot contains sensitive/transient field %q", typeOfSnapshot.Field(i).Name)
			}
		}
	}
}

func TestSafariRenderedDOMProductionScriptsExcludeForbiddenBehavior(t *testing.T) {
	pageForbidden := []string{
		"document.cookie", "localStorage", "sessionStorage", "indexedDB", "caches", "credentials",
		"fetch(", "XMLHttpRequest", "WebSocket", "EventSource", "sendBeacon",
		"location=", "location.assign", "location.replace", "history.", ".click(", ".scroll", ".focus(", ".reload(", ".submit(",
		"appendChild", "removeChild", "replaceChild", "insertBefore", "setAttribute", "removeAttribute", "MutationObserver",
		"setTimeout", "setInterval", "requestAnimationFrame", "while(", "for(",
	}
	for _, fragment := range pageForbidden {
		if strings.Contains(safariRenderedDOMPageScript, fragment) {
			t.Errorf("page-side script contains forbidden behavior %q", fragment)
		}
	}

	jxaForbidden := []string{
		"activate()", "tabs()", "source()", "openLocation", "System Events", "Accessibility", "WebKit", "Safari extension",
	}
	for _, fragment := range jxaForbidden {
		if strings.Contains(safariRenderedDOMJXAScript, fragment) {
			t.Errorf("JXA script contains forbidden behavior %q", fragment)
		}
	}
}

func TestPhase10B2dSourceExcludesSelectorsPipelineAndForbiddenInfrastructure(t *testing.T) {
	body, err := os.ReadFile("safari_rendered_dom.go")
	if err != nil {
		t.Fatalf("read safari_rendered_dom.go: %v", err)
	}
	forbidden := []string{
		"time.Now(", "Chrome", "Chromium", "Accessibility", "System Events", "WebKit", "Safari extension", "browser extension",
		"net/http", "net.Dial", "net.Listen", "localhost", "socket", "document.cookie", "localStorage", "sessionStorage", "indexedDB",
		"fetch(", "XMLHttpRequest", "WebSocket", "EventSource", "sendBeacon", "MutationObserver", "RawPost", "ExtractPreparedPage(",
		"RunScanBatch(", "database/sql", "sqlite", "persistence", "bridge", "Swift", "Xcode", "retry", "article", "permalink", "author selector",
	}
	for _, fragment := range forbidden {
		if strings.Contains(string(body), fragment) {
			t.Errorf("Phase 10B2d source contains forbidden behavior %q", fragment)
		}
	}
}

func phase10B2dValidResponse(t *testing.T) []byte {
	t.Helper()
	return phase10B2dMarshalResponse(t, phase10B2dWireResponse())
}

func phase10B2dWireResponse() safariRenderedDOMWireResponse {
	return safariRenderedDOMWireResponse{
		SchemaVersion: safariRenderedDOMResponseVersion,
		PageURL:       "https://example.test/rendered",
		RenderedDOM:   "<html><body>synthetic</body></html>",
	}
}

func phase10B2dMarshalResponse(t *testing.T, response safariRenderedDOMWireResponse) []byte {
	t.Helper()
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return encoded
}
