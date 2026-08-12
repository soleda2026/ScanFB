# MVP Scan Input Workflow Decision

Date: 2026-08-12

Milestone: Phase 11C0 - MVP input workflow decision only.

Outcome: **APPROVE one transitional MVP workflow: a ScanFB-owned manual
prepared-post form for exactly one enrolled group, producing one bounded,
versioned prepared-snapshot JSON payload.** No workflow is implemented by this
milestone.

## Phase 11B0 context

Phase 11B0 deferred a production Facebook collector because no examined source
proved complete, trustworthy post identity, body, author, absolute creation
time, and group binding. Safari source and rendered DOM, structured page-side
scripts, extensions, Accessibility, WebDriver/WebKit, the removed official
Groups API, and other candidates remain unapproved. Phase 11B stays blocked.

Phase 11C0 does not claim to solve automatic Facebook collection. It defines a
different trust boundary for a transitional MVP: the user explicitly supplies
post facts through a deterministic ScanFB-owned form, while ScanFB supplies the
authoritative enrolled-group identity and capture time. "Trustworthy" here
means every accepted field has explicit provenance, is preserved rather than
inferred, and can be rejected when absent or malformed. It does not mean ScanFB
independently verifies the user's transcription against Facebook.

## Decision bar

The workflow must provide enough data for Phase 10A and `RawPost`, preserve
exact group binding and absolute `CreatedAt`, use one finite versioned payload,
fail the whole input closed, remain local, require no cookie/session/profile or
credential access, support deterministic synthetic tests, avoid a generic
import subsystem, and remain replaceable behind the Phase 11A
`GroupPostCollector` contract.

## Candidate comparison

| Candidate | Field coverage and provenance | User friction | Determinism, privacy, and packaging | Decision |
| --- | --- | --- | --- | --- |
| A. Structured local snapshot file | Can carry every field, but a user-authored file has no ScanFB-owned capture flow and repeats the Phase 11B0 provenance problem. | High: learn a schema, create/edit a file, and select it for every group. | Bounded JSON is replayable and local, but adds file picker, sandbox access, cleanup, and private artifact handling. | Reject as the MVP user action. |
| B. ScanFB-owned manual prepared-post form | Complete: app binds group and capture time; user explicitly supplies body, author, absolute creation time, and available identity/permalink. | Moderate-to-high recurring effort, but fields are labeled, no schema editing is required, and no second tool is needed beyond the user-prepared Facebook page. | One bounded typed payload, session-only, local, deterministic, and directly adaptable to Phase 10A. | **APPROVE.** |
| C. Structured clipboard/paste payload | Can carry every field, but whole-payload clipboard provenance is ambiguous and stale or unrelated content is easy to submit. | Lower than file editing only when another tool already generates the payload; no such approved tool exists. | Clipboard broadens private-data exposure and adds parsing/error ambiguity without improving source trust. | Reject. Individual field paste inside B is allowed; automatic clipboard reading is not. |
| D. CSV or JSON file import | JSON supports nested author/version data; CSV does not represent the contract cleanly. Neither provides a trustworthy generation workflow. | CSV requires escaping/column rules; JSON requires schema authoring or an unapproved generator. | Adds an importer/file lifecycle. JSON is used only as the app-generated local DTO encoding for B, not as a user-authored file format. | Reject file import; reject CSV entirely. |
| E. Browser-generated local export/helper | Complete only if a helper can identify the same missing Facebook fields reliably. No such evidence exists. | Requires another executable/action and browser permission or setup. | Recreates the blocked collector problem and expands packaging/browser attack surface. | Reject. |
| F. Other repo-consistent local mechanism | No smaller mechanism was found. A free-form text area would lose field boundaries; drag/drop is another file-import surface. | Either ambiguous or duplicates A-C. | Does not improve provenance or testability. | Reject. |

## Selected workflow

The canonical input unit is exactly one manual prepared snapshot for exactly one
already-enrolled active `WatchedGroup`.

The eventual user interaction is:

1. From the scan flow, the user opens **Nhập dữ liệu quét** for the one group
   ScanFB currently presents. The group name and identity are read-only.
2. The user adds one or more post rows in visible order.
3. For each row, the user pastes or types the exact body, supplies at least one
   author field, enters an explicit absolute creation date and time, and adds a
   post ID or HTTPS permalink when Facebook exposes one.
4. The user presses **Quét dữ liệu đã nhập** once. ScanFB freezes row order,
   creates the scan window and capture time, validates the entire payload, and
   invokes the one-group Phase 11A path through the prepared-snapshot collector.
5. Any malformed row rejects the whole payload. The user corrects the form and
   submits again; ScanFB does not silently drop a row or substitute a value.

The user may paste text into individual fields using normal macOS behavior.
ScanFB does not monitor, bulk-read, retain, or clear the system clipboard. There
is no file picker, drag/drop import, raw JSON editor, or alternate input mode in
the MVP workflow.

This is a per-group input unit. It does not change the product invariant that a
full MVP scan batch contains exactly five active enrolled groups and runs them
sequentially. Phase 12 must decide how five instances of this one-group input
unit are coordinated and when the persisted cursor advances. Phase 11C0 does
not expose a standalone one-group Scan action or mutate the cursor.

## Canonical format and ownership

The canonical format is **prepared snapshot JSON schema version 1**, generated
by ScanFB from the form and consumed only by the local Go boundary. It is not a
user-authored file format and is not a generic import contract.

| Field | Type and rule | Owner |
| --- | --- | --- |
| `schema_version` | Integer, exactly `1` | ScanFB |
| `watched_group_id` | Non-empty authoritative local WatchedGroup ID | ScanFB/Go |
| `watched_group_name` | Current authoritative enrolled-group name | ScanFB/Go |
| `captured_at` | Absolute RFC3339Nano instant | ScanFB caller |
| `posts` | Ordered array containing 1-100 rows | Form order |
| `posts[].post_id` | Optional exact source value; never inferred from URL | User |
| `posts[].post_url` | Optional exact absolute HTTPS URL without user info | User |
| `posts[].author.facebook_user_id` | Optional exact source value | User |
| `posts[].author.canonical_profile_url` | Optional exact source value; no new normalization in this workflow | User |
| `posts[].author.username` | Optional exact source value | User |
| `posts[].author.display_name` | Optional exact display value | User |
| `posts[].body` | Required non-whitespace source text, preserved exactly | User |
| `posts[].created_at` | Required absolute RFC3339Nano instant | User plus ScanFB date/time control |

At least one author field must be non-whitespace. Display-name-only identity
remains display-name-only and must not be promoted into a stable identity. Post
ID and URL remain optional because the existing contract forbids fabricating
unavailable values. The form should make available identity/permalink fields
easy to supply, but absence must remain explicit.

The Go-side input DTO owns schema/version and bounds validation. It maps
one-to-one into the existing Phase 10A `PreparedPageSnapshot`; the domain and
Phase 11A contracts do not own JSON or UI concerns. Swift may render fields and
send the typed payload in a future slice, but it must not independently map,
infer, normalize, classify, or repair post data.

## Group-binding rules

- The workflow starts from exactly one active enrolled WatchedGroup selected by
  authoritative Go state.
- Group ID and name are not editable user fields.
- The local request identifies the selected WatchedGroup; Go reloads or uses
  that authoritative value and constructs `PreparedPageSnapshot` group fields.
- The public form DTO contains no per-post group selector. The adapter assigns
  the same authoritative group ID to every Phase 10A `PreparedPost`.
- Any duplicate/conflicting wrapper or post group value introduced below the UI
  boundary rejects the whole input before application processing.
- The collector result returns exactly the same WatchedGroup ID requested by
  Phase 11A. Existing Phase 11A mismatch handling remains authoritative.

The group binding proves which enrolled group the user says the rows came from;
it does not verify current Facebook membership or access permission.

## Timestamp rules

- `created_at` is required for every row. The form uses an explicit date/time
  control labeled for `Asia/Ho_Chi_Minh`; it starts unset and cannot default to
  the current time or `captured_at`.
- ScanFB serializes the user-supplied local date/time as an absolute RFC3339Nano
  value with the explicit `+07:00` offset and preserves the represented instant.
- Relative text such as elapsed minutes or hours is not accepted. ScanFB does
  not infer from browser locale, visible wording, DOM position, post ID, URL, or
  capture time.
- If the user cannot obtain an exact absolute creation time, that row cannot be
  submitted. There is no review fallback for missing source time.
- On final submission, the caller creates the `ScanWindow`; `captured_at` is the
  exact caller-supplied `ScanWindow.ScanStarted()` instant and is not editable.
- Phase 10A validates absolute syntax and preserves the instant. Existing scan
  rules remain authoritative for deciding whether `created_at` lies inside the
  current-day scan window; the input workflow does not duplicate that policy.

## Bounds

- Exactly one snapshot and one enrolled group per submission.
- Schema version exactly `1`.
- Between 1 and 100 post rows.
- Maximum encoded UTF-8 JSON payload size: **1 MiB (1,048,576 bytes)**.
- The aggregate byte limit includes every string and structural byte; no field
  can bypass it.
- Order is exactly the visible form-row order and is frozen at submission.
- Oversize, empty, truncated, duplicate-key, trailing-data, or unsupported
  version payloads fail closed. The implementing slice must use strict
  structured decoding and must not truncate.

These are MVP bounds, not a reusable importer framework. Changing them requires
an explicit later decision and tests.

## Privacy and local-storage behavior

- Input remains local and in memory for one form/submission lifecycle.
- The workflow does not create or import a snapshot file and adds no watched
  group, completed-batch, or other persistence schema.
- Canceling the form discards its in-memory values. After one terminal result or
  input error is dismissed, the form payload is discarded rather than retained
  as import history.
- No raw payload, body, author, URL, or private Facebook value is written to
  stdout/stderr or diagnostic logs.
- No cookie, token, credential, Keychain item, browser profile/session/cache,
  Safari storage, or private Facebook interface is accessed.
- A later separately approved completed-batch save may persist accepted
  `RawPost` through the existing persistence contract; this decision adds no
  input-artifact persistence or retention rule.

## Fail-closed behavior

Before Phase 11A is invoked, strict transport decoding rejects malformed JSON,
duplicate keys, trailing data, unsupported version, oversize payload, missing
author/body/time, invalid HTTPS URL, empty rows, and missing or unauthorized
group selection without mutation.

After transport acceptance, the prepared-snapshot collector calls Phase 10A
exactly once. A Phase 10A extraction error returns zero posts and becomes an
explicit collection failure under existing Phase 11A semantics. No partial row
set reaches `RunScanBatch`; no invalid field is trimmed into validity, inferred,
defaulted, repaired, or silently omitted. Existing application time, author,
blocklist, geography, intent, keyword, deduplication, and aggregation rules then
evaluate the exact accepted `RawPost` values.

## Phase 10A integration

The future adapter converts the strict v1 DTO into one
`PreparedPageSnapshot`:

- authoritative group ID/name become snapshot group fields;
- `captured_at` becomes `CapturedAt`;
- form order becomes `Posts` order;
- optional post identity/URL and exact author/body/time values map directly;
- each internal `PreparedPost.GroupID` is assigned the authoritative group ID.

`ExtractPreparedPage` remains the single deterministic validation/mapping path
to ordered `[]RawPost`. Phase 11C0 does not change Phase 10A behavior or claim
that Phase 10A authenticates Facebook source truth.

## Phase 11A integration

A future one-shot prepared-snapshot collector implements the existing
`GroupPostCollector` interface. For one request it validates that the supplied
snapshot is bound to `request.WatchedGroup`, invokes Phase 10A, and returns one
`GroupCollectionResult` with the same group ID and ordered posts. It performs no
selection, cursor movement, retry, persistence, clock read, background work, or
browser action.

Phase 11A remains unchanged: it owns lifecycle transitions, calls the injected
collector once, rejects group mismatch, and sends successful posts to the
existing application pipeline. A future production collector can replace the
manual collector behind the same interface without changing orchestration,
rules, domain data, or batch semantics.

## Rejected alternatives

- **User-authored JSON file:** deterministic after creation but burdens ordinary
  users with schema editing and leaves source provenance unclear.
- **CSV:** poor fit for nested author identity, optional fields, Unicode body,
  versioning, and strict structured validation.
- **Bulk clipboard payload:** depends on an absent generator, can be stale or
  unrelated, and unnecessarily exposes the whole private payload through the
  system clipboard.
- **Free-form pasted text:** cannot preserve field boundaries or absolute time
  without forbidden inference.
- **External/browser helper export:** no approved helper has trustworthy access
  to the fields Phase 11B0 found missing; moving the code to another executable
  does not solve that blocker.
- **Drag/drop or multiple formats:** duplicates file import and creates a small
  importer framework without improving data quality.

## Next implementation milestone

**Phase 11C1 - Go-only bounded prepared-snapshot collector adapter.**

That milestone may implement only the strict v1 JSON DTO/decoder, 1 MiB and
100-post bounds, authoritative one-group binding, mapping through unchanged
Phase 10A, and a concrete one-shot `GroupPostCollector` adapter with focused
synthetic tests. It must not add Swift/UI, bridge wiring, files, clipboard
access, persistence, batch-five orchestration, cursor movement, Safari/browser
runtime, selector, API client, retry, scheduler, concurrency, or background
work. The form and typed local bridge wiring require a later separately scoped
milestone after the Go adapter contract passes.

## Non-goals and stop conditions

This decision does not implement an importer, collector, UI, bridge operation,
file picker, drag/drop target, clipboard reader, persistence change, browser
integration, or batch execution. It does not reopen Phase 11B or the Safari
selector investigation.

Stop implementation if it requires multiple formats, a generic importer,
user-authored JSON, relative-time conversion, inferred author/post/group data,
automatic clipboard access, raw private-data logging, unbounded input, partial
acceptance, browser/session access, a Phase 11A change, persistence/schema work,
or cursor progression.
