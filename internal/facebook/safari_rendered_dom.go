package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	SafariRenderedDOMMaxBytes         = 8 << 20
	safariRenderedDOMResponseMaxBytes = 6*SafariRenderedDOMMaxBytes + 64<<10
	safariRenderedDOMResponseVersion  = 1
)

const safariRenderedDOMPageScript = `document.documentElement ? document.documentElement.outerHTML : ""`

const safariRenderedDOMJXAScript = `const safari = Application("Safari");
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
const renderedDOM = safari.doJavaScript('` + safariRenderedDOMPageScript + `', {in: tab});
JSON.stringify({
    schema_version: 1,
    page_url: pageURL || "",
    page_title: pageTitle || "",
    rendered_dom: renderedDOM || ""
});`

var (
	ErrSafariJavaScriptFromAppleEventsDisabled = errors.New("facebook: Safari JavaScript from Apple Events is disabled")
	ErrMalformedSafariRenderedDOMResponse      = errors.New("facebook: Safari rendered DOM response is malformed")
	ErrUnsupportedSafariRenderedDOMVersion     = errors.New("facebook: Safari rendered DOM response version is unsupported")
	ErrEmptySafariRenderedDOM                  = errors.New("facebook: Safari rendered DOM is empty")
	ErrOversizedSafariRenderedDOM              = errors.New("facebook: Safari rendered DOM exceeds the limit")
)

// SafariRenderedDOMSnapshot is one bounded read-only serialization of the
// already-rendered document in Safari's active tab.
type SafariRenderedDOMSnapshot struct {
	PageURL     string
	PageTitle   string
	RenderedDOM string
	CapturedAt  time.Time
}

type safariRenderedDOMWireResponse struct {
	SchemaVersion int    `json:"schema_version"`
	PageURL       string `json:"page_url"`
	PageTitle     string `json:"page_title"`
	RenderedDOM   string `json:"rendered_dom"`
}

// AcquireSafariActiveTabRenderedDOM reads one bounded rendered-document
// serialization from the current tab of Safari's front window.
func AcquireSafariActiveTabRenderedDOM(ctx context.Context, capturedAt time.Time) (SafariRenderedDOMSnapshot, error) {
	runner := osaScriptCommandRunner{stdoutMaxBytes: safariRenderedDOMResponseMaxBytes}
	return acquireSafariActiveTabRenderedDOM(ctx, capturedAt, safariAcquisitionTimeout, runner)
}

func acquireSafariActiveTabRenderedDOM(
	ctx context.Context,
	capturedAt time.Time,
	timeout time.Duration,
	runner safariCommandRunner,
) (SafariRenderedDOMSnapshot, error) {
	if capturedAt.IsZero() {
		return SafariRenderedDOMSnapshot{}, ErrInvalidSafariCapturedAt
	}
	if err := ctx.Err(); err != nil {
		return SafariRenderedDOMSnapshot{}, ErrSafariAcquisitionCanceled
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := runner.Run(runCtx, safariOSAExecutablePath, []string{"-l", "JavaScript", "-e", safariRenderedDOMJXAScript})
	if errors.Is(ctx.Err(), context.Canceled) {
		return SafariRenderedDOMSnapshot{}, ErrSafariAcquisitionCanceled
	}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return SafariRenderedDOMSnapshot{}, ErrSafariAcquisitionTimeout
	}
	if errors.Is(err, context.Canceled) {
		return SafariRenderedDOMSnapshot{}, ErrSafariAcquisitionCanceled
	}
	if err != nil {
		return SafariRenderedDOMSnapshot{}, ErrSafariOSAStartFailure
	}
	if result.exitCode != 0 {
		return SafariRenderedDOMSnapshot{}, classifySafariRenderedDOMCommandFailure(result.stderr)
	}
	if result.stdoutExceeded {
		return SafariRenderedDOMSnapshot{}, ErrOversizedSafariRenderedDOM
	}
	return parseSafariRenderedDOMResponse(result.stdout, capturedAt)
}

func classifySafariRenderedDOMCommandFailure(stderr []byte) error {
	message := strings.ToLower(string(stderr))
	if strings.Contains(message, "javascript from apple events") {
		return ErrSafariJavaScriptFromAppleEventsDisabled
	}
	return classifySafariCommandFailure(stderr)
}

func parseSafariRenderedDOMResponse(response []byte, capturedAt time.Time) (SafariRenderedDOMSnapshot, error) {
	if capturedAt.IsZero() {
		return SafariRenderedDOMSnapshot{}, ErrInvalidSafariCapturedAt
	}
	if len(response) > safariRenderedDOMResponseMaxBytes {
		return SafariRenderedDOMSnapshot{}, ErrOversizedSafariRenderedDOM
	}

	var wire safariRenderedDOMWireResponse
	decoder := json.NewDecoder(bytes.NewReader(response))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return SafariRenderedDOMSnapshot{}, ErrMalformedSafariRenderedDOMResponse
	}
	if err := requireSafariResponseEOF(decoder); err != nil {
		return SafariRenderedDOMSnapshot{}, ErrMalformedSafariRenderedDOMResponse
	}
	if wire.SchemaVersion != safariRenderedDOMResponseVersion {
		return SafariRenderedDOMSnapshot{}, ErrUnsupportedSafariRenderedDOMVersion
	}
	if !validSafariActiveTabURL(wire.PageURL) {
		return SafariRenderedDOMSnapshot{}, ErrUnsupportedSafariActiveTabURL
	}
	if strings.TrimSpace(wire.RenderedDOM) == "" {
		return SafariRenderedDOMSnapshot{}, ErrEmptySafariRenderedDOM
	}
	if len([]byte(wire.RenderedDOM)) > SafariRenderedDOMMaxBytes {
		return SafariRenderedDOMSnapshot{}, ErrOversizedSafariRenderedDOM
	}

	return SafariRenderedDOMSnapshot{
		PageURL:     wire.PageURL,
		PageTitle:   wire.PageTitle,
		RenderedDOM: wire.RenderedDOM,
		CapturedAt:  capturedAt,
	}, nil
}
