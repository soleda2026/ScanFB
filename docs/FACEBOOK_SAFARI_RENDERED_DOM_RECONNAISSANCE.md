# Facebook Safari Rendered-DOM Reconnaissance

Date: 2026-08-11

Milestone: Phase 10B2e - one-page rendered-DOM reconnaissance only.

Recommendation: **STOP / INCONCLUSIVE. Do not proceed to Phase 10B2b.**

## Context

Phase 10B2d implemented `AcquireSafariActiveTabRenderedDOM` as the only
approved acquisition path. Before the live call, the user confirmed that
Safari's front-window current tab was the intended logged-in Facebook group
page and that `Allow JavaScript from Apple Events` was enabled.

Exactly one production acquisition was executed. The acquisition and in-memory
analysis test passed, including its Facebook-group URL guard. No retry or
second acquisition occurred.

Known Phase 10B2d manual-validation evidence placed the rendered document at
approximately 2.9 MiB. The Phase 10B2e test runner retained only its passing
summary and filtered the bounded redacted structural report, so this milestone
cannot independently report a narrower size or post-level counts.

## Privacy and redaction

The rendered DOM remained in process memory only. It was not written under the
repository or to a temporary content file. The analysis emitted no raw DOM,
post body, author name, Facebook user ID, post ID, permalink, cookie, token or
session value. The temporary overlay and analyzer under `/tmp` contained code
only and were deleted after the single run.

No screenshot was taken. No Safari navigation, activation, tab switch, scroll,
click, focus, refresh, typing, form submission, polling or retry occurred.

## Page inspected

- Page type: one user-prepared Facebook group page.
- Active-page evidence: user confirmation plus the passing analyzer's strict
  Facebook host and `/groups/<redacted>/` path guard.
- Acquisition path: production `AcquireSafariActiveTabRenderedDOM` only.
- Approximate rendered-DOM size context: approximately 2.9 MiB from the
  preceding Phase 10B2d live validation; the Phase 10B2e exact byte count was
  not retained.

## Structural findings

The temporary analyzer was designed to report only bounded counts, semantic
attribute names and redacted href shapes. It checked top-level
`role="article"` candidates and their descendant permalink, body, author and
machine-time coverage without printing private values.

The command-output filter reduced the successful test output to a pass summary
and did not retain that redacted report. No credible post-level marker or count
can therefore be asserted from the surviving Phase 10B2e evidence. A missing
retained count is **not** recorded as zero and is not evidence that the marker
was absent from the rendered DOM.

## Confidence

| Concept | Confidence | Retained evidence |
| --- | --- | --- |
| Post/article container | NOT FOUND | Candidate counts and marker categories were not retained. |
| Post permalink or post ID | NOT FOUND | No redacted href-shape result survived the run. |
| Post body container | NOT FOUND | Per-candidate body-marker coverage was not retained. |
| Author identity/display container | NOT FOUND | Per-candidate profile-link coverage was not retained. |
| Created-at timestamp | NOT FOUND | Machine-time and structured timestamp counts were not retained. |
| Group identity | STRONG | User-confirmed target plus passing strict Facebook group URL guard. |
| Ordering of visible posts | NOT FOUND | Candidate-container count and traversal evidence were not retained. |

`NOT FOUND` above means not established by retained evidence in this milestone;
it does not claim that the corresponding structure is absent from the private
page.

## Structural counts

- Live rendered-DOM acquisitions: 1.
- Confirmed Facebook group pages: 1.
- Candidate post containers: unavailable, not zero.
- Candidates with permalink/post identity: unavailable, not zero.
- Candidates with body marker: unavailable, not zero.
- Candidates with author shape: unavailable, not zero.
- Candidates with machine-readable timestamp: unavailable, not zero.
- Candidates missing required fields: unavailable.

These are one-page reconnaissance results only.

## Marker decisions

The only marker category established by retained evidence is the validated
active-page URL shape `/groups/<group>/` for page-level group identity.

No post-level category is approved. In particular, this milestone does not
approve `role="article"`, `/groups/<group>/posts/<post>/`, profile-link shapes,
body-preview attributes, `time[datetime]`, timestamp fields or DOM order because
their observed counts and correlations were not retained.

Generated or obfuscated CSS classes, nth-child positions, arbitrary depth,
localized text, relative-time strings, title wording, generic React/Comet
symbols, transient IDs and broad text searches remain explicitly rejected as
production selectors regardless of this tooling failure.

## Blocker and recommendation

The proceed bar requires STRONG evidence for distinct post containers,
permalink/identity, body, author and machine-readable timestamp, plus sufficient
deterministic traversal order. The retained evidence proves only page-level
group identity.

A second acquisition would be required to recover the missing structural
metadata. Phase 10B2e explicitly forbids a second acquisition just to
understand the first, so this milestone stops fail closed.

**Recommendation: STOP / INCONCLUSIVE. Phase 10B2b production selectors remain
blocked.** Any repeat reconnaissance must be a separately approved milestone
whose temporary analyzer writes only the redacted report to a mode-0600 file
before process exit, verifies the report, and deletes it after documentation.
It must not retain raw DOM or implement selectors.

## Non-goals preserved

No production selector, Facebook parser, extraction struct, `RawPost`, relative
time parser, Phase 10A call, Phase 11 execution, persistence, SQLite, SwiftUI,
bridge, dependency, browser mutation or alternate browser mechanism was added.
