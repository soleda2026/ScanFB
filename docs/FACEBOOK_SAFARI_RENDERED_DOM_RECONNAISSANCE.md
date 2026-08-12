# Facebook Safari Rendered-DOM Reconnaissance

Date: 2026-08-11

Closeout milestone: Phase 10B2g - rendered-DOM live reconnaissance closeout
only.

Decision: **Phase 10B2b production selectors remain BLOCKED.**

## Context

Phase 10B2d implemented and live-validated the bounded, read-only
`AcquireSafariActiveTabRenderedDOM` path. Phase 10B2e then completed exactly one
reconnaissance acquisition, but test-output compression discarded its redacted
structural report, so that run stopped inconclusive rather than claiming the
DOM lacked structure.

Phase 10B2f added pure `AnalyzeRenderedDOMStructure` and a mode-0600 `/tmp`
preservation procedure for its bounded typed report. Phase 10B2g records the
successfully preserved result from one separately user-guided acquisition
against one active Facebook group page.

This closeout does not implement selectors, extraction or scan execution.

## Privacy and acquisition

- Live acquisitions for this preserved result: 1.
- Browser target: exactly one user-prepared current tab in Safari's front
  window.
- Acquisition path: production `AcquireSafariActiveTabRenderedDOM` only.
- Analysis path: `AnalyzeRenderedDOMStructure` in memory.
- Raw rendered DOM was not committed or stored as a repository fixture.
- The preserved result contains counts, confidence and canonical redacted
  marker names only.
- No post body, author, Facebook user/group/post ID, permalink, profile URL,
  cookie, token, credential or session value is recorded here.
- No navigation, activation, tab switch, scroll, click, focus, refresh, typing,
  form submission, polling or retry occurred.

## Preserved result

| Field | Value |
| --- | ---: |
| Analyzed rendered DOM | 3,180,722 bytes |
| Candidate post containers | 2 |
| Candidates with permalink/post identity | 0 |
| Candidates with body marker | 0 |
| Candidates with author marker | 0 |
| Candidates with machine-readable timestamp | 0 |
| Candidates with relative-time-only evidence | 0 |
| Candidates with complete post evidence | 0 |
| Group-consistent permalinks | 0 |
| Group-page URL shape valid | true |
| Deterministic traversal candidates | 2 |

These counts describe one page only and do not establish cross-page or
cross-session stability.

## Confidence

| Concept | Confidence | Redacted evidence |
| --- | --- | --- |
| Post/article container | STRONG | Two distinct semantic `role=article` candidates. |
| Post permalink or post ID | NOT_FOUND | Zero candidates carried an approved permalink/identity shape. |
| Post body container | NOT_FOUND | Zero candidates carried an approved body marker. |
| Author identity/display container | NOT_FOUND | Zero candidates carried an approved author marker. |
| Created-at timestamp | NOT_FOUND | Zero machine-readable and zero relative-time-only candidates. |
| Group identity | STRONG | Validated Facebook group-page URL shape. |
| Ordering of visible posts | STRONG | Two candidates have deterministic DOM source order. |

STRONG container and traversal evidence does not compensate for missing
permalink, body, author or timestamp evidence.

## Marker decisions

Recognized stable categories in this page:

- `role=article`
- `dom-source-order`

No other post-level marker category is approved by this result.

The following categories remain explicitly rejected:

- arbitrary depth;
- broad text search;
- generated or obfuscated classes;
- generic React/Comet symbols;
- localized visible text;
- nth-child positions;
- relative-time text;
- title wording;
- transient internal IDs.

These rejected categories must not be reintroduced as selectors or private
value heuristics merely to make progress.

## Decision and blocker

The Phase 10B2b proceed bar requires strong fail-closed evidence for distinct
post containers, permalink/post identity, body, author and machine-readable
timestamp, plus sufficient traversal order. This result meets only the
container, group-page identity and traversal parts of that bar.

Complete post evidence count is zero. Without an approved identity, body,
author and timestamp relationship, a runtime parser could not produce a
complete post deterministically or fail closed when critical structure
disappears.

**Decision: keep Phase 10B2b BLOCKED and close the current Safari rendered-DOM
selector investigation.** Do not weaken selector standards. No further Safari
selector implementation should proceed unless a separately justified new
evidence or acquisition technique supplies the missing critical structure
without violating ScanFB's privacy and browser boundaries.

## Non-goals preserved

No production selector, Facebook parser, extraction struct, `RawPost`, relative
time parser, Phase 10A call, Phase 11 execution, persistence, SQLite, SwiftUI,
bridge, dependency, browser mutation or alternate browser mechanism was added.
