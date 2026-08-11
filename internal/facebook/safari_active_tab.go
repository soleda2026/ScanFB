package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

const (
	SafariPageContentMaxBytes         = 4 << 20
	safariAcquisitionResponseMaxBytes = 6*SafariPageContentMaxBytes + 64<<10
	safariAcquisitionResponseVersion  = 1
	safariDiagnosticMaxBytes          = 16 << 10
	safariAcquisitionTimeout          = 5 * time.Second
	safariOSAExecutablePath           = "/usr/bin/osascript"
)

const safariActiveTabJXAScript = `const safari = Application("Safari");
if (!safari.running()) {
    throw new Error("SCANFB_SAFARI_NOT_RUNNING");
}
const windows = safari.windows();
if (windows.length === 0) {
    throw new Error("SCANFB_SAFARI_NO_ACTIVE_WINDOW");
}
let tab;
try {
    tab = windows[0].currentTab();
} catch (_) {
    throw new Error("SCANFB_SAFARI_NO_ACTIVE_TAB");
}
if (!tab) {
    throw new Error("SCANFB_SAFARI_NO_ACTIVE_TAB");
}
const pageURL = tab.url();
const pageTitle = tab.name();
const pageContent = tab.source();
JSON.stringify({
    schema_version: 1,
    page_url: pageURL || "",
    page_title: pageTitle || "",
    page_content: pageContent || ""
});`

var (
	ErrSafariNotRunning                   = errors.New("facebook: Safari is not running")
	ErrSafariNoActiveWindow               = errors.New("facebook: Safari has no active window")
	ErrSafariNoActiveTab                  = errors.New("facebook: Safari has no active tab")
	ErrSafariAutomationPermissionDenied   = errors.New("facebook: Safari automation permission is denied")
	ErrSafariOSAStartFailure              = errors.New("facebook: osascript could not start")
	ErrSafariAcquisitionTimeout           = errors.New("facebook: Safari acquisition timed out")
	ErrSafariAcquisitionCanceled          = errors.New("facebook: Safari acquisition was canceled")
	ErrSafariAcquisitionNonzeroExit       = errors.New("facebook: Safari acquisition exited unsuccessfully")
	ErrMalformedSafariAcquisitionResponse = errors.New("facebook: Safari acquisition response is malformed")
	ErrUnsupportedSafariActiveTabURL      = errors.New("facebook: Safari active tab URL is unsupported")
	ErrEmptySafariPageContent             = errors.New("facebook: Safari active tab content is empty")
	ErrOversizedSafariPageContent         = errors.New("facebook: Safari active tab content exceeds the limit")
	ErrInvalidSafariCapturedAt            = errors.New("facebook: Safari acquisition captured at is invalid")
)

// SafariActiveTabSnapshot is one bounded read-only capture of Safari's active tab.
type SafariActiveTabSnapshot struct {
	PageURL     string
	PageTitle   string
	PageContent string
	CapturedAt  time.Time
}

type safariActiveTabWireResponse struct {
	SchemaVersion int    `json:"schema_version"`
	PageURL       string `json:"page_url"`
	PageTitle     string `json:"page_title"`
	PageContent   string `json:"page_content"`
}

type safariCommandResult struct {
	stdout         []byte
	stderr         []byte
	exitCode       int
	stdoutExceeded bool
}

type safariCommandRunner interface {
	Run(context.Context, string, []string) (safariCommandResult, error)
}

type safariCommandRunnerFunc func(context.Context, string, []string) (safariCommandResult, error)

func (run safariCommandRunnerFunc) Run(ctx context.Context, executable string, args []string) (safariCommandResult, error) {
	return run(ctx, executable, args)
}

type osaScriptCommandRunner struct {
	stdoutMaxBytes int
	stderrMaxBytes int
}

// AcquireSafariActiveTab reads one bounded snapshot from the current tab of
// Safari's front window without navigating or interacting with page content.
func AcquireSafariActiveTab(ctx context.Context, capturedAt time.Time) (SafariActiveTabSnapshot, error) {
	return acquireSafariActiveTab(ctx, capturedAt, safariAcquisitionTimeout, osaScriptCommandRunner{})
}

func acquireSafariActiveTab(
	ctx context.Context,
	capturedAt time.Time,
	timeout time.Duration,
	runner safariCommandRunner,
) (SafariActiveTabSnapshot, error) {
	if capturedAt.IsZero() {
		return SafariActiveTabSnapshot{}, ErrInvalidSafariCapturedAt
	}
	if err := ctx.Err(); err != nil {
		return SafariActiveTabSnapshot{}, ErrSafariAcquisitionCanceled
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := runner.Run(runCtx, safariOSAExecutablePath, []string{"-l", "JavaScript", "-e", safariActiveTabJXAScript})
	if errors.Is(ctx.Err(), context.Canceled) {
		return SafariActiveTabSnapshot{}, ErrSafariAcquisitionCanceled
	}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return SafariActiveTabSnapshot{}, ErrSafariAcquisitionTimeout
	}
	if errors.Is(err, context.Canceled) {
		return SafariActiveTabSnapshot{}, ErrSafariAcquisitionCanceled
	}
	if err != nil {
		return SafariActiveTabSnapshot{}, ErrSafariOSAStartFailure
	}
	if result.exitCode != 0 {
		return SafariActiveTabSnapshot{}, classifySafariCommandFailure(result.stderr)
	}
	if result.stdoutExceeded {
		return SafariActiveTabSnapshot{}, ErrOversizedSafariPageContent
	}
	return parseSafariActiveTabResponse(result.stdout, capturedAt)
}

func classifySafariCommandFailure(stderr []byte) error {
	message := strings.ToLower(string(stderr))
	switch {
	case strings.Contains(message, "scanfb_safari_not_running"):
		return ErrSafariNotRunning
	case strings.Contains(message, "scanfb_safari_no_active_window"):
		return ErrSafariNoActiveWindow
	case strings.Contains(message, "scanfb_safari_no_active_tab"):
		return ErrSafariNoActiveTab
	case strings.Contains(message, "-1743"),
		strings.Contains(message, "not authorized to send apple events"),
		strings.Contains(message, "not permitted to send apple events"):
		return ErrSafariAutomationPermissionDenied
	default:
		return ErrSafariAcquisitionNonzeroExit
	}
}

func (runner osaScriptCommandRunner) Run(ctx context.Context, executable string, args []string) (safariCommandResult, error) {
	stdoutMaxBytes, stderrMaxBytes := runner.outputLimits()
	stdout := newBoundedSafariOutputBuffer(stdoutMaxBytes)
	stderr := newBoundedSafariOutputBuffer(stderrMaxBytes)
	command := exec.CommandContext(ctx, executable, args...)
	command.Stdout = stdout
	command.Stderr = stderr

	if err := command.Start(); err != nil {
		return safariCommandResult{}, err
	}
	err := command.Wait()
	result := safariCommandResult{
		stdout:         stdout.Bytes(),
		stderr:         stderr.Bytes(),
		stdoutExceeded: stdout.Exceeded(),
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if err == nil {
		return result, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.exitCode = exitError.ExitCode()
		return result, nil
	}
	result.exitCode = -1
	return result, nil
}

func (runner osaScriptCommandRunner) outputLimits() (int, int) {
	stdoutMaxBytes := runner.stdoutMaxBytes
	if stdoutMaxBytes <= 0 {
		stdoutMaxBytes = safariAcquisitionResponseMaxBytes
	}
	stderrMaxBytes := runner.stderrMaxBytes
	if stderrMaxBytes <= 0 {
		stderrMaxBytes = safariDiagnosticMaxBytes
	}
	return stdoutMaxBytes, stderrMaxBytes
}

type boundedSafariOutputBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func newBoundedSafariOutputBuffer(limit int) *boundedSafariOutputBuffer {
	return &boundedSafariOutputBuffer{limit: limit}
}

func (buffer *boundedSafariOutputBuffer) Write(data []byte) (int, error) {
	written := len(data)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining < len(data) {
		buffer.exceeded = true
	}
	if remaining > 0 {
		if remaining < len(data) {
			data = data[:remaining]
		}
		_, _ = buffer.buffer.Write(data)
	}
	return written, nil
}

func (buffer *boundedSafariOutputBuffer) Bytes() []byte {
	return append([]byte(nil), buffer.buffer.Bytes()...)
}

func (buffer *boundedSafariOutputBuffer) Exceeded() bool {
	return buffer.exceeded
}

func parseSafariActiveTabResponse(response []byte, capturedAt time.Time) (SafariActiveTabSnapshot, error) {
	if capturedAt.IsZero() {
		return SafariActiveTabSnapshot{}, ErrInvalidSafariCapturedAt
	}
	if len(response) > safariAcquisitionResponseMaxBytes {
		return SafariActiveTabSnapshot{}, ErrOversizedSafariPageContent
	}

	var wire safariActiveTabWireResponse
	decoder := json.NewDecoder(bytes.NewReader(response))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return SafariActiveTabSnapshot{}, ErrMalformedSafariAcquisitionResponse
	}
	if err := requireSafariResponseEOF(decoder); err != nil {
		return SafariActiveTabSnapshot{}, err
	}
	if wire.SchemaVersion != safariAcquisitionResponseVersion {
		return SafariActiveTabSnapshot{}, ErrMalformedSafariAcquisitionResponse
	}
	if !validSafariActiveTabURL(wire.PageURL) {
		return SafariActiveTabSnapshot{}, ErrUnsupportedSafariActiveTabURL
	}
	if strings.TrimSpace(wire.PageContent) == "" {
		return SafariActiveTabSnapshot{}, ErrEmptySafariPageContent
	}
	if len([]byte(wire.PageContent)) > SafariPageContentMaxBytes {
		return SafariActiveTabSnapshot{}, ErrOversizedSafariPageContent
	}

	return SafariActiveTabSnapshot{
		PageURL:     wire.PageURL,
		PageTitle:   wire.PageTitle,
		PageContent: wire.PageContent,
		CapturedAt:  capturedAt,
	}, nil
}

func requireSafariResponseEOF(decoder *json.Decoder) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ErrMalformedSafariAcquisitionResponse
	}
	return nil
}

func validSafariActiveTabURL(value string) bool {
	if value == "" || value != strings.TrimSpace(value) {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs() && strings.EqualFold(parsed.Scheme, "https") && parsed.Hostname() != "" && parsed.User == nil
}
