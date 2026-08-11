package facebook

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestAnalyzeRenderedDOMStructureCountsCompleteSemanticCandidates(t *testing.T) {
	report, err := AnalyzeRenderedDOMStructure(phase10B2fCompleteSyntheticDOM(), "https://www.facebook.com/groups/private-group-123")
	if err != nil {
		t.Fatalf("AnalyzeRenderedDOMStructure() error = %v", err)
	}

	if report.CandidatePostContainerCount != 2 || report.DeterministicTraversalCount != 2 {
		t.Fatalf("container/traversal counts = %d/%d", report.CandidatePostContainerCount, report.DeterministicTraversalCount)
	}
	if report.CandidatePermalinkBearingCount != 2 || report.GroupConsistentPermalinkCount != 2 {
		t.Fatalf("permalink counts = %d group-consistent=%d", report.CandidatePermalinkBearingCount, report.GroupConsistentPermalinkCount)
	}
	if report.CandidateBodyBearingCount != 2 || report.CandidateAuthorBearingCount != 2 || report.CandidateMachineTimestampCount != 2 {
		t.Fatalf("body/author/time counts = %d/%d/%d", report.CandidateBodyBearingCount, report.CandidateAuthorBearingCount, report.CandidateMachineTimestampCount)
	}
	if report.CandidateCompleteEvidenceCount != 2 || report.CandidateRelativeTimeOnlyCount != 0 {
		t.Fatalf("complete/relative counts = %d/%d", report.CandidateCompleteEvidenceCount, report.CandidateRelativeTimeOnlyCount)
	}
	if !report.GroupPageURLShapeValid {
		t.Fatal("GroupPageURLShapeValid = false")
	}

	for name, confidence := range map[string]RenderedDOMReconnaissanceConfidence{
		"container": report.PostContainerConfidence,
		"permalink": report.PermalinkConfidence,
		"body":      report.BodyConfidence,
		"author":    report.AuthorConfidence,
		"timestamp": report.MachineTimestampConfidence,
		"group":     report.GroupIdentityConfidence,
		"traversal": report.TraversalConfidence,
	} {
		if confidence != RenderedDOMReconnaissanceStrong {
			t.Errorf("%s confidence = %q, want STRONG", name, confidence)
		}
	}
}

func TestAnalyzeRenderedDOMStructureReturnsCanonicalRedactedMarkers(t *testing.T) {
	report, err := AnalyzeRenderedDOMStructure(phase10B2fCompleteSyntheticDOM(), "https://facebook.com/groups/private-group-123")
	if err != nil {
		t.Fatalf("AnalyzeRenderedDOMStructure() error = %v", err)
	}
	want := []string{
		"data-ad-comet-preview=message",
		"data-ad-preview=message",
		"data-utime",
		"dom-source-order",
		"href=/groups/<group>/posts/<post>/",
		"href=/groups/<group>/user/<author>/",
		"role=article",
		"tag=article",
		"time[datetime]",
	}
	if !reflect.DeepEqual(report.MarkerCategories, want) {
		t.Fatalf("MarkerCategories = %#v, want %#v", report.MarkerCategories, want)
	}
}

func TestAnalyzeRenderedDOMStructurePartialCoverageIsTentative(t *testing.T) {
	dom := `<html><body>
<div role="article"><a href="/groups/group-a/posts/post-a/"></a><div data-ad-preview="message"></div></div>
<div role="article"></div>
</body></html>`
	report, err := AnalyzeRenderedDOMStructure(dom, "https://facebook.com/groups/group-a")
	if err != nil {
		t.Fatalf("AnalyzeRenderedDOMStructure() error = %v", err)
	}
	if report.PostContainerConfidence != RenderedDOMReconnaissanceStrong {
		t.Fatalf("PostContainerConfidence = %q", report.PostContainerConfidence)
	}
	if report.PermalinkConfidence != RenderedDOMReconnaissanceTentative || report.BodyConfidence != RenderedDOMReconnaissanceTentative {
		t.Fatalf("partial confidences = permalink %q body %q", report.PermalinkConfidence, report.BodyConfidence)
	}
	if report.AuthorConfidence != RenderedDOMReconnaissanceNotFound || report.MachineTimestampConfidence != RenderedDOMReconnaissanceNotFound {
		t.Fatalf("missing confidences = author %q time %q", report.AuthorConfidence, report.MachineTimestampConfidence)
	}
}

func TestAnalyzeRenderedDOMStructureRejectsClassAndDepthOnlyEvidence(t *testing.T) {
	dom := `<html><body><div class="x1"><div class="x2"><div class="x3">private body</div></div></div></body></html>`
	report, err := AnalyzeRenderedDOMStructure(dom, "")
	if err != nil {
		t.Fatalf("AnalyzeRenderedDOMStructure() error = %v", err)
	}
	if report.CandidatePostContainerCount != 0 || report.PostContainerConfidence != RenderedDOMReconnaissanceNotFound {
		t.Fatalf("CSS/depth-only structure became evidence: %#v", report)
	}
	for _, rejected := range []string{"generated-or-obfuscated-class", "nth-child-position", "arbitrary-depth"} {
		if !slices.Contains(report.RejectedUnstableMarkerCategories, rejected) {
			t.Errorf("missing rejected marker %q", rejected)
		}
	}
}

func TestAnalyzeRenderedDOMStructureRelativeTimeOnlyIsNotStrong(t *testing.T) {
	dom := `<article><time>2h</time></article>`
	report, err := AnalyzeRenderedDOMStructure(dom, "")
	if err != nil {
		t.Fatalf("AnalyzeRenderedDOMStructure() error = %v", err)
	}
	if report.CandidateRelativeTimeOnlyCount != 1 {
		t.Fatalf("CandidateRelativeTimeOnlyCount = %d", report.CandidateRelativeTimeOnlyCount)
	}
	if report.MachineTimestampConfidence != RenderedDOMReconnaissanceTentative {
		t.Fatalf("MachineTimestampConfidence = %q, want TENTATIVE", report.MachineTimestampConfidence)
	}
	if slices.Contains(report.MarkerCategories, "relative-time-text") {
		t.Fatal("relative display text was promoted to a stable marker")
	}
}

func TestAnalyzeRenderedDOMStructureClassifiesOptionalGroupURLShape(t *testing.T) {
	dom := `<article></article>`
	tests := []struct {
		name       string
		pageURL    string
		wantValid  bool
		confidence RenderedDOMReconnaissanceConfidence
	}{
		{name: "valid group", pageURL: "https://www.facebook.com/groups/private-group", wantValid: true, confidence: RenderedDOMReconnaissanceStrong},
		{name: "empty optional URL", pageURL: "", wantValid: false, confidence: RenderedDOMReconnaissanceNotFound},
		{name: "non group", pageURL: "https://www.facebook.com/home", wantValid: false, confidence: RenderedDOMReconnaissanceNotFound},
		{name: "userinfo", pageURL: "https://user:secret@facebook.com/groups/private", wantValid: false, confidence: RenderedDOMReconnaissanceNotFound},
		{name: "foreign host", pageURL: "https://example.test/groups/private", wantValid: false, confidence: RenderedDOMReconnaissanceNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report, err := AnalyzeRenderedDOMStructure(dom, tc.pageURL)
			if err != nil {
				t.Fatalf("AnalyzeRenderedDOMStructure() error = %v", err)
			}
			if report.GroupPageURLShapeValid != tc.wantValid || report.GroupIdentityConfidence != tc.confidence {
				t.Fatalf("group result = valid %v confidence %q", report.GroupPageURLShapeValid, report.GroupIdentityConfidence)
			}
		})
	}
}

func TestAnalyzeRenderedDOMStructureNeverReturnsPrivateValues(t *testing.T) {
	privateValues := []string{
		"PRIVATE-BODY-ALPHA",
		"PRIVATE-BODY-BETA",
		"PRIVATE-AUTHOR-ALPHA",
		"PRIVATE-AUTHOR-BETA",
		"private-post-111",
		"private-post-222",
		"private-group-123",
		"https://www.facebook.com/groups/private-group-123/posts/private-post-111/",
	}
	report, err := AnalyzeRenderedDOMStructure(phase10B2fCompleteSyntheticDOM(), "https://www.facebook.com/groups/private-group-123")
	if err != nil {
		t.Fatalf("AnalyzeRenderedDOMStructure() error = %v", err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	for _, privateValue := range privateValues {
		if strings.Contains(string(encoded), privateValue) {
			t.Errorf("redacted report leaked private value %q", privateValue)
		}
	}
}

func TestRenderedDOMReconnaissanceResultIsBounded(t *testing.T) {
	report, err := AnalyzeRenderedDOMStructure(phase10B2fCompleteSyntheticDOM(), "https://facebook.com/groups/private-group-123")
	if err != nil {
		t.Fatalf("AnalyzeRenderedDOMStructure() error = %v", err)
	}
	for name, markers := range map[string][]string{
		"recognized": report.MarkerCategories,
		"rejected":   report.RejectedUnstableMarkerCategories,
	} {
		if len(markers) > RenderedDOMReconnaissanceMaxMarkerCategories {
			t.Fatalf("%s marker count = %d", name, len(markers))
		}
		for _, marker := range markers {
			if len(marker) > RenderedDOMReconnaissanceMaxMarkerLength {
				t.Fatalf("%s marker %q length = %d", name, marker, len(marker))
			}
		}
	}
	if !sortStringsAreStable(report.MarkerCategories) || !sortStringsAreStable(report.RejectedUnstableMarkerCategories) {
		t.Fatal("marker arrays are not deterministically sorted")
	}
}

func TestAnalyzeRenderedDOMStructureFailsClosedForInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		dom     string
		wantErr error
	}{
		{name: "empty", dom: "", wantErr: ErrEmptyRenderedDOMReconnaissanceInput},
		{name: "whitespace", dom: " \n\t ", wantErr: ErrEmptyRenderedDOMReconnaissanceInput},
		{name: "oversized", dom: strings.Repeat("x", SafariRenderedDOMMaxBytes+1), wantErr: ErrOversizedRenderedDOMReconnaissanceInput},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report, err := AnalyzeRenderedDOMStructure(tc.dom, "")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("AnalyzeRenderedDOMStructure() error = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(report, RenderedDOMReconnaissanceReport{}) {
				t.Fatalf("invalid input returned report %#v", report)
			}
		})
	}
}

func TestAnalyzeRenderedDOMStructureIsDeterministic(t *testing.T) {
	dom := phase10B2fCompleteSyntheticDOM()
	pageURL := "https://facebook.com/groups/private-group-123"
	first, err := AnalyzeRenderedDOMStructure(dom, pageURL)
	if err != nil {
		t.Fatalf("first analysis error = %v", err)
	}
	second, err := AnalyzeRenderedDOMStructure(dom, pageURL)
	if err != nil {
		t.Fatalf("second analysis error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated result changed: first=%#v second=%#v", first, second)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if !reflect.DeepEqual(firstJSON, secondJSON) {
		t.Fatal("deterministic typed result encoded differently")
	}
}

func TestPhase10B2fSourceExcludesRuntimeAndPrivateOutputBehavior(t *testing.T) {
	body, err := os.ReadFile("rendered_dom_reconnaissance.go")
	if err != nil {
		t.Fatalf("read rendered_dom_reconnaissance.go: %v", err)
	}
	forbidden := []string{
		"time.Now(", "os.", "io/fs", "os/exec", "WriteFile", "Create(", "OpenFile(",
		"net/http", "net.Dial", "net.Listen", "localhost", "Application(\"Safari\")", "osascript", "doJavaScript",
		"document.cookie", "localStorage", "sessionStorage", "indexedDB", "fetch(", "XMLHttpRequest", "WebSocket",
		"Accessibility", "System Events", "WebKit", "Safari extension", "RawPost", "ExtractPreparedPage(", "RunScanBatch(",
		"database/sql", "sqlite", "persistence", "bridge", "Swift", "Xcode", "Phase 11", "querySelector", "querySelectorAll",
	}
	for _, fragment := range forbidden {
		if strings.Contains(string(body), fragment) {
			t.Errorf("Phase 10B2f source contains forbidden behavior %q", fragment)
		}
	}
}

func TestAnalyzeRenderedDOMStructureDoesNotMutateInput(t *testing.T) {
	dom := phase10B2fCompleteSyntheticDOM()
	before := dom
	if _, err := AnalyzeRenderedDOMStructure(dom, "https://facebook.com/groups/private-group-123"); err != nil {
		t.Fatalf("AnalyzeRenderedDOMStructure() error = %v", err)
	}
	if dom != before {
		t.Fatal("rendered DOM input changed")
	}
}

func phase10B2fCompleteSyntheticDOM() string {
	return `<html><body>
<div role="article" class="obfuscated-one">
  <a href="https://www.facebook.com/groups/private-group-123/posts/private-post-111/"></a>
  <div data-ad-preview="message">PRIVATE-BODY-ALPHA</div>
  <a href="/groups/private-group-123/user/private-author-111/">PRIVATE-AUTHOR-ALPHA</a>
  <time datetime="2026-08-11T08:00:00Z">relative display</time>
</div>
<article class="obfuscated-two">
  <a href="/groups/private-group-123/posts/private-post-222/"></a>
  <div data-ad-comet-preview="message">PRIVATE-BODY-BETA</div>
  <a href="/groups/private-group-123/user/private-author-222/">PRIVATE-AUTHOR-BETA</a>
  <abbr data-utime="1786438800">relative display</abbr>
</article>
</body></html>`
}

func sortStringsAreStable(values []string) bool {
	return slices.IsSorted(values)
}
