# Production Group Collector Source Decision

Date: 2026-08-12

Milestone: Phase 11B0 - production `GroupPostCollector` source decision only.

Outcome: **DEFER because no candidate has sufficient evidence.** No production
source or collector implementation is approved by this milestone.

## Phase 11A context

Phase 11A defines a one-call `GroupPostCollector` boundary for exactly one
already-enrolled active `WatchedGroup` and its existing `ScanWindow`. A
successful collector must return ordered, group-bound `RawPost` values with a
non-empty body, author evidence, an absolute machine-readable `CreatedAt`, and
post identity or permalink when the source exposes one. Missing or conflicting
required data must fail closed. Collector failure remains a failed attempt with
no retry, cursor advancement, persistence write, or fabricated result.

Phase 10A proves only deterministic conversion of a caller-prepared typed
snapshot. It does not prove how trustworthy Facebook post data is obtained.
Phase 10B2g proves that the current Safari rendered DOM contains two semantic
article candidates in deterministic order, but no approved stable evidence for
permalink, body, author, or machine timestamp. Zero candidates contain complete
post evidence.

## Decision bar

Approval requires one realistic source that can supply every required field,
preserve exact group binding and order, provide a reliable absolute creation
time, avoid invented identity, remain finite and fail closed, avoid private
browser-state scraping, support deterministic fixture tests, and fit an MVP
workflow. Browser access alone is not evidence that the required post fields
can be identified reliably.

## Candidate comparison

`Yes` means the source currently proves the field. `Conditional` means a user
or an unproven extraction rule would have to supply it. `No` means current
evidence does not establish it.

| Candidate | Group binding | Order | Body | Author | Absolute `CreatedAt` | Post identity / permalink | Result |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A. Safari active-tab rendered DOM | Yes, from the validated page URL | Yes, for the two observed article containers | No | No | No | No | Reject on current evidence |
| B. Fixed page-side JavaScript returning typed post records | Yes, from the same validated tab URL | Conditional on the same article traversal | No stable field evidence | No stable field evidence | No stable field evidence | No stable field evidence | Defer; changing output shape does not create missing evidence |
| C. User-prepared bounded local snapshot | Conditional on binding import to the selected enrolled group | Conditional on user/export order | Conditional | Conditional | Conditional | Conditional | Defer; no proven practical source supplies a complete trustworthy group feed |
| D. Official Meta API | No applicable current Groups API | No | No | No | No | No | Reject |
| E. Safari extension/content script | Yes, with website permission and URL validation | Conditional on page traversal | No stable field evidence | No stable field evidence | No stable field evidence | No stable field evidence | Reject for this milestone |
| F. Accessibility/UI scripting | Conditional on window/page state | Presentation order only | Visible text only, not established as exact body | Visible labels only | No; visible time may be relative/localized | No stable identity evidence | Reject |
| G. WebDriver, WebKit, or alternate browser automation | Conditional on a separate automated browsing context | Conditional on selectors/scrolling | No stable field evidence | No stable field evidence | No stable field evidence | No stable field evidence | Reject |
| H. Other repo-consistent bounded source | None found | None found | None found | None found | None found | None found | No candidate to approve |

All candidates could impose byte limits in isolation. Bounds do not compensate
for absent provenance or field evidence.

## Candidate findings

### A. Safari rendered DOM

Acquisition works and can bind one snapshot to the current HTTPS group page.
The preserved live report found two `role=article` containers and deterministic
DOM source order. It found zero permalink-bearing, body-bearing,
author-bearing, machine-timestamp, or complete-evidence candidates. Generated
classes, arbitrary depth, broad text search, localized text, relative-time text,
`nth-child`, title wording, transient IDs, and private-value heuristics remain
rejected. This source cannot currently satisfy `RawPost`.

### B. Structured page-side JavaScript

A fixed page-side script could reduce transport volume by returning typed data
instead of full HTML. It would still inspect the same rendered document. The
existing live evidence does not identify stable body, author, timestamp, or
permalink evidence for such a script to read. Returning JSON would alter the
transport shape, not the evidentiary quality. Candidate B therefore remains
deferred unless materially new redacted evidence establishes every required
field without weakening selector standards.

### C. User-prepared local snapshot

A narrow local typed snapshot can structurally carry all `RawPost` fields and
can be bounded, replayed, validated, and processed without browser secrets or
network access. It is not yet a proven production source. Meta's export tools
export a person's Facebook information and activity; Meta also states that an
export does not include information another person shared. That does not
establish a complete feed of posts from arbitrary private groups the user has
joined. Manual re-entry would require the user to find and accurately preserve
post identity, author, exact group, body, order, and an absolute timestamp for
each post; syntax validation cannot prove those values are faithful. This is
too error-prone and cumbersome to approve as the current scan input.

Candidate C is the least invasive fallback for a future product redefinition,
but Phase 11B0 does not design a generic importer, alter Phase 10A, or approve
manual transcription as production collection.

### D. Official Meta API

Meta's Graph API v19 announcement deprecated the Groups API,
`publish_to_groups`, `groups_access_member_info`, and group app installation,
with removal applying to all versions on 2024-04-22. No current official Meta
documentation was found for an API that lets a third-party desktop app list or
read posts from arbitrary groups merely because its user is a member. The
official API therefore does not support ScanFB's intended private-group/member
model, and there is no permission or app-review path to evaluate for this use
case.

### E. Browser extension/content script

Apple documents that Safari web extensions can run content scripts on matching
sites after the user grants website access. This could place code in the
authenticated page, but it does not establish stable Facebook post fields; it
would still need the missing selectors or equivalent evidence. It also adds a
persistent browser component, host permission, extension target, signing,
distribution, and lifecycle surface. Candidate E is not approved without new
field evidence and a separate product/security decision.

### F. Accessibility/UI scripting

Apple's Accessibility API exposes application UI elements and can perform UI
actions. It does not provide an authoritative Facebook DOM or stable machine
timestamp/post identity contract. Visible labels and relative/localized time
are insufficient, while Accessibility authority is broader than this read-only
collector requires. Candidate F is rejected.

### G. WebDriver, WebKit, and alternate automation

Apple documents Safari WebDriver as browser-test automation through a local
REST server in isolated automation windows, separate from normal browsing data.
It does not adopt the user's prepared authenticated tab and would introduce a
new automation context, setup, browser dependency, and likely navigation or
scrolling. An app-owned `WKWebView` similarly owns a separate browsing context.
Neither mechanism resolves the absent Facebook field evidence. Candidate G is
rejected.

### H. Other source

No repo-consistent source was found that supplies all required fields. Browser
profile/cache/database reads, cookie or token extraction, unsupported/private
Facebook interfaces, hidden listeners, and broad automated scraping violate
existing boundaries and are not fallback candidates.

## User friction

| Candidate | User action per scan | One-time setup | MVP assessment |
| --- | --- | --- | --- |
| A/B | Open, authenticate, navigate, and prepare one Safari group tab | Safari developer setting and Automation consent | Low-to-moderate friction, but insufficient data evidence |
| C | Obtain or manually prepare a complete typed snapshot for each group and scan window | Learn an exact file/input procedure | High recurring friction and unproven field provenance |
| D | Authorize a Meta app | App registration/review would be required if an API existed | Unavailable for the product model |
| E | Prepare a group page and invoke/allow the extension | Install, enable, and grant Facebook website access | Moderate setup plus a new persistent subsystem |
| F | Prepare the page and keep UI state compatible | Grant broad Accessibility permission | Brittle and over-privileged |
| G | Use an automation browsing session | Enable remote automation and add an automation stack | High setup and wrong-session fit |

The more manual candidate C is acceptable in principle, but its current form
does not merely add friction: it leaves the critical timestamp and identity
provenance unproven.

## Privacy and security

- A and B retain Safari session ownership and can remain local, bounded, and
  one-shot, but their authenticated page access is sensitive and currently
  yields insufficient field evidence.
- C can remain fully local and requires no credential, cookie, token, browser
  profile, or persistent permission. Its exported snapshot may contain private
  group content and would require explicit local retention/redaction policy in
  any future product decision.
- D would require tokens and Meta platform permissions, but no applicable API
  exists.
- E introduces persistent website access to private Facebook pages. Permission
  must be least-privilege, yet the extension still lacks approved extraction
  evidence.
- F grants broad UI-observation/control authority and is disproportionate.
- G adds an automation context or app-owned web session. Session transfer,
  credential reuse, private profile reads, and hidden listeners remain
  forbidden.

No candidate may read cookies, credentials, browser profile/session files, or
unsupported private endpoints. Diagnostic output must never contain private
post content.

## Reliability

The current Safari candidates are limited to posts Facebook has rendered in
the prepared page; no pagination, scrolling, or completeness guarantee is
approved. DOM and extension candidates remain sensitive to Facebook structure.
Accessibility is additionally sensitive to localization and presentation.
WebDriver/WebKit changes the browsing context. A typed local snapshot is stable
after capture but cannot be more trustworthy than its unproven source. The
removed official Groups API supplies no reliability path.

Every future source must reject a post when required body, author, absolute
creation time, or group binding is absent or conflicting. It must not derive an
absolute time from relative visible text or fabricate a post identity.

## Packaging implications

- A/B reuse the existing Safari Apple Events acquisition boundary; any future
  change still requires validated TCC attribution and final app packaging.
- C would require only a narrow local-input UI/adapter decision, with
  sandbox-safe file access and private-data handling defined before code.
- D would add network, token storage, app registration/review, and API policy,
  but is unavailable.
- E requires a Safari extension target, manifest/website permissions, signing,
  distribution, and extension lifecycle.
- F requires Accessibility consent and potentially Automation integration.
- G requires a browser automation or WebKit subsystem; Safari WebDriver also
  uses a local REST service and developer configuration.

Phase 11B0 makes none of these packaging changes.

## Testability

A future approved adapter must use synthetic, non-private fixtures to prove
strict decoding, group binding, exact order, field preservation, finite bounds,
missing-field rejection, deterministic replay, cancellation, and zero result on
failure. User-guided live validation may report only redacted counts and field
coverage; private Facebook payloads must not enter the repository or logs.

A/B/E/F/G can test transport or mechanics synthetically, but those tests cannot
prove stable Facebook field semantics. C can test typed import deterministically
but cannot prove source truth without a separately validated acquisition/export
workflow. D has no applicable endpoint to test.

## Selected outcome

**DEFER because no candidate has sufficient evidence.** There is no approved
production input source, and Phase 11B implementation must not begin.

The exact blocker is absence of one supported, bounded source that proves all
of: enrolled-group binding, ordered post coverage, exact body, author evidence,
absolute machine-readable `CreatedAt`, and post identity/permalink when exposed.
The current Safari page lacks four critical post-level evidence categories; the
official Groups API is removed; official user export is not a joined-group feed;
and the remaining browser mechanisms only change access or packaging without
proving the missing fields.

## Next milestone

No Phase 11B implementation milestone is approved. The only permitted next
step is a separately authorized **Phase 11B0a source-evidence renewal** after a
materially new source or acquisition technique is identified. That docs/manual
evidence slice must prove complete redacted field coverage, timestamp quality,
group binding, bounds, permissions, and an understandable user workflow before
requesting an implementation decision. Re-running the closed Safari selector
approach or merely wrapping the same DOM in structured JavaScript is not new
evidence.

## Stop conditions

Stop any future source investigation or implementation if it requires:

- weakening the Phase 10B2g selector evidence standard;
- generated CSS, arbitrary depth, `nth-child`, broad or localized text search,
  relative-time conversion, or private-value heuristics;
- cookie, token, credential, browser profile/session/cache/database extraction;
- unsupported/private Facebook APIs or hidden network listeners;
- automated login, navigation, scrolling, clicking, polling, retry, or
  background collection;
- fabricated identity, inferred missing fields, unbounded output, or private
  live fixtures in the repository;
- Phase 11A, persistence, cursor, bridge, Swift, or Xcode changes before a
  source is approved.

## Non-goals

This milestone does not implement a collector, selector, parser, importer,
extension, API client, browser automation path, persistence change, scan UI, or
Phase 11A behavior. It does not redefine `RawPost`, loosen required-field
validation, reopen joined-group discovery, or approve a generic file-import
system.

## Official sources

- Meta: [Introducing Facebook Graph and Marketing API v19](https://developers.facebook.com/blog/post/2024/01/23/introducing-facebook-graph-and-marketing-api-v19/)
- Meta: [Learn what categories of information are available to export from your Facebook profile](https://www.facebook.com/help/326826564067688)
- Meta: [Export a copy of your Facebook information](https://www.facebook.com/help/212802592074644)
- Apple: [Managing Safari web extension permissions](https://developer.apple.com/documentation/safariservices/managing-safari-web-extension-permissions)
- Apple: [Using content script and style sheet keys](https://developer.apple.com/documentation/safariservices/using-content-script-and-style-sheet-keys)
- Apple: [Testing with WebDriver in Safari](https://developer.apple.com/documentation/webkit/testing-with-webdriver-in-safari)
- Apple: [`AXUIElement`](https://developer.apple.com/documentation/applicationservices/axuielement)
- Apple: [`WKWebView`](https://developer.apple.com/documentation/webkit/wkwebview)
