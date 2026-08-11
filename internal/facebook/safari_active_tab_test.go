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

func TestParseSafariActiveTabResponsePreservesBoundedSnapshotExactly(t *testing.T) {
	capturedAt := phase10B1CapturedAt()
	response := []byte(`{
  "schema_version": 1,
  "page_url": "https://www.facebook.com/groups/macbook-buyers",
  "page_title": "MacBook Buyers Việt Nam",
  "page_content": "<html><body>Cần mua MacBook Pro</body></html>"
}`)

	snapshot, err := parseSafariActiveTabResponse(response, capturedAt)
	if err != nil {
		t.Fatalf("parseSafariActiveTabResponse() error = %v", err)
	}
	if snapshot.PageURL != "https://www.facebook.com/groups/macbook-buyers" {
		t.Fatalf("PageURL = %q", snapshot.PageURL)
	}
	if snapshot.PageTitle != "MacBook Buyers Việt Nam" {
		t.Fatalf("PageTitle = %q", snapshot.PageTitle)
	}
	if snapshot.PageContent != "<html><body>Cần mua MacBook Pro</body></html>" {
		t.Fatalf("PageContent = %q", snapshot.PageContent)
	}
	if !snapshot.CapturedAt.Equal(capturedAt) {
		t.Fatalf("CapturedAt = %v, want caller value %v", snapshot.CapturedAt, capturedAt)
	}
}

func TestParseSafariActiveTabResponseFailsClosed(t *testing.T) {
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
			wantErr:    ErrMalformedSafariAcquisitionResponse,
		},
		{
			name:       "invalid JSON",
			response:   func(t *testing.T) []byte { return []byte("not-json") },
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrMalformedSafariAcquisitionResponse,
		},
		{
			name: "unknown field",
			response: func(t *testing.T) []byte {
				return []byte(`{"schema_version":1,"page_url":"https://example.test","page_title":"","page_content":"content","extra":true}`)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrMalformedSafariAcquisitionResponse,
		},
		{
			name: "trailing response",
			response: func(t *testing.T) []byte {
				return append(phase10B1ValidResponse(t), []byte(` {}`)...)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrMalformedSafariAcquisitionResponse,
		},
		{
			name: "unsupported response schema",
			response: func(t *testing.T) []byte {
				wire := phase10B1WireResponse()
				wire.SchemaVersion++
				return phase10B1MarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrMalformedSafariAcquisitionResponse,
		},
		{
			name: "empty page content",
			response: func(t *testing.T) []byte {
				wire := phase10B1WireResponse()
				wire.PageContent = " \n\t "
				return phase10B1MarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrEmptySafariPageContent,
		},
		{
			name: "non HTTPS URL",
			response: func(t *testing.T) []byte {
				wire := phase10B1WireResponse()
				wire.PageURL = "http://example.test/page"
				return phase10B1MarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrUnsupportedSafariActiveTabURL,
		},
		{
			name: "relative URL",
			response: func(t *testing.T) []byte {
				wire := phase10B1WireResponse()
				wire.PageURL = "/page"
				return phase10B1MarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrUnsupportedSafariActiveTabURL,
		},
		{
			name: "userinfo URL",
			response: func(t *testing.T) []byte {
				wire := phase10B1WireResponse()
				wire.PageURL = "https://user:secret@example.test/page"
				return phase10B1MarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrUnsupportedSafariActiveTabURL,
		},
		{
			name: "oversized page content",
			response: func(t *testing.T) []byte {
				wire := phase10B1WireResponse()
				wire.PageContent = strings.Repeat("x", SafariPageContentMaxBytes+1)
				return phase10B1MarshalResponse(t, wire)
			},
			capturedAt: phase10B1CapturedAt(),
			wantErr:    ErrOversizedSafariPageContent,
		},
		{
			name:       "missing captured at",
			response:   phase10B1ValidResponse,
			capturedAt: time.Time{},
			wantErr:    ErrInvalidSafariCapturedAt,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snapshot, err := parseSafariActiveTabResponse(tc.response(t), tc.capturedAt)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("parseSafariActiveTabResponse() error = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(snapshot, SafariActiveTabSnapshot{}) {
				t.Fatalf("invalid response returned snapshot %#v", snapshot)
			}
		})
	}
}

func TestParseSafariActiveTabResponseIsDeterministicAndAllowsEmptyTitle(t *testing.T) {
	response := phase10B1ValidResponse(t)
	capturedAt := phase10B1CapturedAt()

	first, err := parseSafariActiveTabResponse(response, capturedAt)
	if err != nil {
		t.Fatalf("first parse error = %v", err)
	}
	second, err := parseSafariActiveTabResponse(response, capturedAt)
	if err != nil {
		t.Fatalf("second parse error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same response changed: first=%#v second=%#v", first, second)
	}
	if first.PageTitle != "" {
		t.Fatalf("PageTitle = %q, want optional empty title", first.PageTitle)
	}
}

func TestParseSafariActiveTabResponseAcceptsContentAboveFormerLimit(t *testing.T) {
	const formerPageContentMaxBytes = 1 << 20
	wire := phase10B1WireResponse()
	wire.PageContent = strings.Repeat("x", formerPageContentMaxBytes+1)

	snapshot, err := parseSafariActiveTabResponse(phase10B1MarshalResponse(t, wire), phase10B1CapturedAt())
	if err != nil {
		t.Fatalf("parseSafariActiveTabResponse() above former limit error = %v", err)
	}
	if len([]byte(snapshot.PageContent)) != formerPageContentMaxBytes+1 {
		t.Fatalf("decoded content bytes = %d", len([]byte(snapshot.PageContent)))
	}
}

func TestParseSafariActiveTabResponseAcceptsEscapedContentAtExactDecodedLimit(t *testing.T) {
	wire := phase10B1WireResponse()
	wire.PageContent = strings.Repeat("<", SafariPageContentMaxBytes)
	response := phase10B1MarshalResponse(t, wire)
	if len(response) <= SafariPageContentMaxBytes {
		t.Fatalf("fixture did not exercise JSON expansion: encoded bytes = %d", len(response))
	}

	snapshot, err := parseSafariActiveTabResponse(response, phase10B1CapturedAt())
	if err != nil {
		t.Fatalf("parseSafariActiveTabResponse() exact decoded limit error = %v", err)
	}
	if len([]byte(snapshot.PageContent)) != SafariPageContentMaxBytes {
		t.Fatalf("decoded content bytes = %d", len([]byte(snapshot.PageContent)))
	}
}

func TestSafariAcquisitionResponseEnvelopeIsFiniteAndConservative(t *testing.T) {
	const wantResponseMaxBytes = 25_231_360
	if SafariPageContentMaxBytes != 4_194_304 {
		t.Fatalf("decoded content limit = %d, want 4 MiB", SafariPageContentMaxBytes)
	}
	if safariAcquisitionResponseMaxBytes != wantResponseMaxBytes {
		t.Fatalf("transport envelope = %d, want %d", safariAcquisitionResponseMaxBytes, wantResponseMaxBytes)
	}

	wire := phase10B1WireResponse()
	wire.PageContent = strings.Repeat("<", SafariPageContentMaxBytes)
	response := phase10B1MarshalResponse(t, wire)
	if len(response) > safariAcquisitionResponseMaxBytes {
		t.Fatalf("worst-case escaped fixture bytes = %d, envelope = %d", len(response), safariAcquisitionResponseMaxBytes)
	}
}

func TestSafariActiveTabSnapshotContainsNoSensitiveBrowserState(t *testing.T) {
	typeOfSnapshot := reflect.TypeOf(SafariActiveTabSnapshot{})
	for i := 0; i < typeOfSnapshot.NumField(); i++ {
		field := strings.ToLower(typeOfSnapshot.Field(i).Name)
		for _, forbidden := range []string{"cookie", "credential", "session", "token", "header", "storage", "profile", "history", "cache", "windowid", "tabid"} {
			if strings.Contains(field, forbidden) {
				t.Fatalf("SafariActiveTabSnapshot contains sensitive/transient field %q", typeOfSnapshot.Field(i).Name)
			}
		}
	}
}

func TestAcquireSafariActiveTabUsesDirectOsaScriptAndSeparatesDiagnostics(t *testing.T) {
	var executable string
	var arguments []string
	runner := safariCommandRunnerFunc(func(_ context.Context, path string, args []string) (safariCommandResult, error) {
		executable = path
		arguments = append([]string(nil), args...)
		return safariCommandResult{
			stdout: phase10B1ValidResponse(t),
			stderr: []byte("bounded diagnostic that must not corrupt stdout"),
		}, nil
	})

	snapshot, err := acquireSafariActiveTab(context.Background(), phase10B1CapturedAt(), time.Second, runner)
	if err != nil {
		t.Fatalf("acquireSafariActiveTab() error = %v", err)
	}
	if executable != safariOSAExecutablePath {
		t.Fatalf("executable = %q, want %q", executable, safariOSAExecutablePath)
	}
	wantArguments := []string{"-l", "JavaScript", "-e", safariActiveTabJXAScript}
	if !reflect.DeepEqual(arguments, wantArguments) {
		t.Fatalf("arguments = %#v, want %#v", arguments, wantArguments)
	}
	if snapshot.PageURL != "https://example.test/page" || snapshot.PageContent != "<html>content</html>" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestAcquireSafariActiveTabMapsProcessFailuresExplicitly(t *testing.T) {
	tests := []struct {
		name    string
		result  safariCommandResult
		runErr  error
		wantErr error
	}{
		{
			name:    "process start failure",
			runErr:  errors.New("synthetic start failure"),
			wantErr: ErrSafariOSAStartFailure,
		},
		{
			name: "automation permission denied",
			result: safariCommandResult{
				exitCode: 1,
				stderr:   []byte("Not authorized to send Apple events to Safari. (-1743)"),
			},
			wantErr: ErrSafariAutomationPermissionDenied,
		},
		{
			name: "Safari not running",
			result: safariCommandResult{
				exitCode: 1,
				stderr:   []byte("Error: SCANFB_SAFARI_NOT_RUNNING"),
			},
			wantErr: ErrSafariNotRunning,
		},
		{
			name: "no active window",
			result: safariCommandResult{
				exitCode: 1,
				stderr:   []byte("Error: SCANFB_SAFARI_NO_ACTIVE_WINDOW"),
			},
			wantErr: ErrSafariNoActiveWindow,
		},
		{
			name: "no active tab",
			result: safariCommandResult{
				exitCode: 1,
				stderr:   []byte("Error: SCANFB_SAFARI_NO_ACTIVE_TAB"),
			},
			wantErr: ErrSafariNoActiveTab,
		},
		{
			name: "unknown nonzero exit",
			result: safariCommandResult{
				exitCode: 1,
				stderr:   []byte("synthetic private diagnostic"),
			},
			wantErr: ErrSafariAcquisitionNonzeroExit,
		},
		{
			name: "bounded stdout exceeded",
			result: safariCommandResult{
				stdoutExceeded: true,
			},
			wantErr: ErrOversizedSafariPageContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runner := safariCommandRunnerFunc(func(context.Context, string, []string) (safariCommandResult, error) {
				return tc.result, tc.runErr
			})
			snapshot, err := acquireSafariActiveTab(context.Background(), phase10B1CapturedAt(), time.Second, runner)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("acquireSafariActiveTab() error = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(snapshot, SafariActiveTabSnapshot{}) {
				t.Fatalf("failed acquisition returned snapshot %#v", snapshot)
			}
			if strings.Contains(err.Error(), "synthetic private diagnostic") {
				t.Fatalf("raw stderr leaked through error %q", err)
			}
		})
	}
}

func TestAcquireSafariActiveTabMapsTimeoutAndCancellation(t *testing.T) {
	blockingRunner := safariCommandRunnerFunc(func(ctx context.Context, _ string, _ []string) (safariCommandResult, error) {
		<-ctx.Done()
		return safariCommandResult{}, ctx.Err()
	})

	if _, err := acquireSafariActiveTab(context.Background(), phase10B1CapturedAt(), time.Millisecond, blockingRunner); !errors.Is(err, ErrSafariAcquisitionTimeout) {
		t.Fatalf("timeout error = %v, want %v", err, ErrSafariAcquisitionTimeout)
	}

	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := acquireSafariActiveTab(canceledContext, phase10B1CapturedAt(), time.Second, blockingRunner); !errors.Is(err, ErrSafariAcquisitionCanceled) {
		t.Fatalf("cancellation error = %v, want %v", err, ErrSafariAcquisitionCanceled)
	}
}

func TestSafariActiveTabJXAScriptReadsOnlyCurrentTabProperties(t *testing.T) {
	required := []string{
		`Application("Safari")`,
		"safari.running()",
		"safari.windows()",
		"windows[0].currentTab()",
		`throw new Error("SCANFB_SAFARI_NO_ACTIVE_TAB")`,
		"tab.url()",
		"tab.name()",
		"tab.source()",
		"JSON.stringify",
	}
	for _, fragment := range required {
		if !strings.Contains(safariActiveTabJXAScript, fragment) {
			t.Errorf("JXA script missing %q", fragment)
		}
	}
	for _, forbidden := range []string{"activate()", "doJavaScript", "System Events", "click", "scroll", "reload", "openLocation", "tabs()"} {
		if strings.Contains(safariActiveTabJXAScript, forbidden) {
			t.Errorf("JXA script contains forbidden behavior %q", forbidden)
		}
	}
}

func TestBoundedSafariOutputBufferCapsStoredBytes(t *testing.T) {
	buffer := newBoundedSafariOutputBuffer(4)
	written, err := buffer.Write([]byte("123456"))
	if err != nil || written != 6 {
		t.Fatalf("Write() = (%d, %v), want (6, nil)", written, err)
	}
	if string(buffer.Bytes()) != "1234" || !buffer.Exceeded() {
		t.Fatalf("buffer = %q exceeded=%v", buffer.Bytes(), buffer.Exceeded())
	}
}

func TestOSAScriptCommandRunnerOwnsProcessAndBoundsSeparateOutput(t *testing.T) {
	runner := osaScriptCommandRunner{}

	result, err := runner.Run(context.Background(), os.Args[0], phase10B1HelperArguments("separate"))
	if err != nil {
		t.Fatalf("separate output Run() error = %v", err)
	}
	if string(result.stdout) != "machine-response" || string(result.stderr) != "private-diagnostic" {
		t.Fatalf("separate output = stdout %q stderr %q", result.stdout, result.stderr)
	}
	if result.exitCode != 0 || result.stdoutExceeded {
		t.Fatalf("separate output result = %#v", result)
	}

	result, err = runner.Run(context.Background(), os.Args[0], phase10B1HelperArguments("oversized-stderr"))
	if err != nil {
		t.Fatalf("oversized stderr Run() error = %v", err)
	}
	if len(result.stderr) != safariDiagnosticMaxBytes {
		t.Fatalf("stderr bytes = %d, want cap %d", len(result.stderr), safariDiagnosticMaxBytes)
	}

	result, err = runner.Run(context.Background(), os.Args[0], phase10B1HelperArguments("oversized-stdout"))
	if err != nil {
		t.Fatalf("oversized stdout Run() error = %v", err)
	}
	if len(result.stdout) != safariAcquisitionResponseMaxBytes || !result.stdoutExceeded {
		t.Fatalf("stdout bytes = %d exceeded=%v", len(result.stdout), result.stdoutExceeded)
	}

	result, err = runner.Run(context.Background(), os.Args[0], phase10B1HelperArguments("nonzero"))
	if err != nil {
		t.Fatalf("nonzero Run() error = %v", err)
	}
	if result.exitCode != 7 {
		t.Fatalf("nonzero exit code = %d, want 7", result.exitCode)
	}

	timeoutContext, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	startedAt := time.Now()
	_, err = runner.Run(timeoutContext, os.Args[0], phase10B1HelperArguments("block"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("owned process timeout error = %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Fatalf("owned process was not terminated promptly: %v", elapsed)
	}
}

func TestSafariCommandRunnerHelperProcess(t *testing.T) {
	if len(os.Args) < 2 || os.Args[len(os.Args)-2] != "scanfb-phase10b1-helper" {
		return
	}

	switch os.Args[len(os.Args)-1] {
	case "separate":
		_, _ = os.Stdout.WriteString("machine-response")
		_, _ = os.Stderr.WriteString("private-diagnostic")
	case "oversized-stderr":
		_, _ = os.Stderr.WriteString(strings.Repeat("x", safariDiagnosticMaxBytes+1))
	case "oversized-stdout":
		_, _ = os.Stdout.WriteString(strings.Repeat("x", safariAcquisitionResponseMaxBytes+1))
	case "nonzero":
		os.Exit(7)
	case "block":
		time.Sleep(5 * time.Second)
	default:
		os.Exit(8)
	}
	os.Exit(0)
}

func phase10B1HelperArguments(mode string) []string {
	return []string{"-test.run=^TestSafariCommandRunnerHelperProcess$", "--", "scanfb-phase10b1-helper", mode}
}

func TestPhase10B1SourceExcludesDeferredBrowserAndPipelineBehavior(t *testing.T) {
	body, err := os.ReadFile("safari_active_tab.go")
	if err != nil {
		t.Fatalf("read safari_active_tab.go: %v", err)
	}
	forbidden := []string{
		"time.Now(",
		"Chrome",
		"Chromium",
		"Accessibility",
		"System Events",
		"WebKit",
		"Safari extension",
		"browser extension",
		"chromedp",
		"playwright",
		"selenium",
		"net/http",
		"net.Dial",
		"localhost",
		"WebSocket",
		"websocket",
		"cookie",
		"credential",
		"Keychain",
		"local storage",
		"IndexedDB",
		"profile path",
		"history database",
		"cache database",
		"RawPost",
		"ExtractPreparedPage(",
		"RunScanBatch(",
		"StartNextPending(",
		"StartAttempt(",
		"SucceedAttempt(",
		"FailAttempt(",
		"SkipAttempt(",
		"ExpireAtDayBoundary(",
		"database/sql",
		"sqlite",
		"persistence",
		"bridge",
		"Swift",
		"Xcode",
		"go func",
		"sync.",
		"scheduler",
		"retry",
	}
	for _, fragment := range forbidden {
		if strings.Contains(string(body), fragment) {
			t.Errorf("Phase 10B1 source contains deferred behavior %q", fragment)
		}
	}
}

func phase10B1ValidResponse(t *testing.T) []byte {
	t.Helper()
	return phase10B1MarshalResponse(t, phase10B1WireResponse())
}

func phase10B1WireResponse() safariActiveTabWireResponse {
	return safariActiveTabWireResponse{
		SchemaVersion: safariAcquisitionResponseVersion,
		PageURL:       "https://example.test/page",
		PageContent:   "<html>content</html>",
	}
}

func phase10B1MarshalResponse(t *testing.T, response safariActiveTabWireResponse) []byte {
	t.Helper()
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return encoded
}

func phase10B1CapturedAt() time.Time {
	return time.Date(2026, time.August, 11, 14, 30, 0, 0, time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60))
}
