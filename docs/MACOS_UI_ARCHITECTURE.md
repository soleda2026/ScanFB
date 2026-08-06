# macOS UI Architecture

## Purpose

This document records the native macOS presentation architecture for Phase 8.
It records the Phase 8A architecture decision, the Phase 8B app-shell state,
the Phase 8C static Overview fixture state, the Phase 8D fixture-only Leads
presentation state, and the Phase 8E fixture-only Dry Run review state. Phase
8C/8D/8E implement only sample presentation screens; they do not implement a
bridge, persistence wiring, Facebook integration, production UI features, or
live data behavior.

## Presentation Technology

SwiftUI is the approved presentation technology for the native macOS app.
The Phase 8B app shell is Apple Silicon first and uses macOS 14.0 as the
minimum deployment target.

## Repository Layout

- Native app root: `macos/ScanFBApp/`
- The Go core remains under the current Go module.
- SwiftUI code must not be placed in `internal/ui`.
- Phase 8B creates `macos/ScanFBApp/ScanFBApp.xcodeproj` with one app target
  and one unit test target.

`internal/ui` remains only a Go-layer documentation/package placeholder. The
SwiftUI app shell lives outside the Go package tree.

## Layer Ownership

SwiftUI owns:

- windows;
- navigation;
- tabs;
- presentation state;
- formatting;
- accessibility;
- user-triggered actions;
- fixture-driven screens.

Go owns:

- domain invariants;
- buyer-intent classification;
- geographic classification;
- deduplication;
- lead aggregation;
- blocklist semantics;
- batch evaluation;
- persistence contracts and SQLite storage;
- reason codes and deterministic decisions.

## Prohibited Duplication

- Swift must not reimplement classification rules.
- Swift must not infer or regenerate reason codes.
- Swift must not recompute summaries, deduplication, or blocklist outcomes.
- Go must not own macOS visual layout or view-specific state.

## Future Bridge Decision Boundary

A later milestone must evaluate and select exactly one narrow integration model.
No bridge mechanism is selected or implemented in Phase 8A or Phase 8B.

Candidate categories that may be evaluated later:

- C-compatible exported Go library;
- subprocess with structured stdin/stdout;
- local IPC boundary.

Required evaluation criteria:

- deterministic request/response behavior;
- no hidden network;
- no credential transfer;
- explicit schemas;
- error propagation;
- cancellation behavior;
- build complexity;
- packaging;
- testing;
- crash isolation;
- Apple Silicon support.

Explicitly rejected:

- embedding business logic in Swift;
- HTTP server by default;
- cloud API;
- arbitrary JSON maps;
- SQLite access directly from Swift;
- exposing database-local IDs;
- selecting a bridge without a separate milestone.

## Phase 8 UI Slices

- Phase 8A: architecture decision and plan.
- Phase 8B: empty native SwiftUI app shell only. Implemented with one
  `WindowGroup`, one `NavigationSplitView`, six placeholder sections and no
  bridge, database, Facebook, fixtures, settings persistence or networking.
- Phase 8C: static fixture dashboard and batch summary. Implemented for
  `Tổng quan` only with typed immutable fixture data, no bridge, database,
  Facebook, networking, persistence or production controls.
- Phase 8D: fixture lead tabs and lead cards. Implemented for `Leads` only
  with typed immutable buyer lead fixture data, local tab filtering by declared
  fixture category, disabled placeholder actions and no bridge, database,
  Facebook, networking, persistence, status workflow or production controls.
- Phase 8E: fixture Dry Run review. Implemented for `Dry Run` only with typed
  immutable included/review/excluded post fixture data, local tab filtering by
  declared fixture category, one disabled placeholder action and no bridge,
  database, Facebook, networking, persistence, restore workflow or production
  controls.
- Phase 8F: fixture settings and blocklist presentation.
- Phase 8G: UI interaction state for viewed/contacted/ignored using in-memory fixture state only.
- Phase 8H: bridge evaluation and architecture decision.
- Phase 8I+: only after bridge decision, narrow Go integration milestones.

Each milestone must be independently buildable and manually testable.

## Phase 8B Initial App-Shell Requirements

Phase 8B implements only:

- native SwiftUI macOS app;
- one main window;
- app name ScanFB;
- placeholder navigation;
- deployment target macOS 14.0;
- no Go bridge;
- no database;
- no Facebook;
- no networking;
- no third-party dependency;
- no production settings;
- no persistence;
- no generated fixture business data beyond minimal visual placeholders.

The implemented shell uses local view state only. It has no AppKit bridge,
menu-bar-only mode, secondary windows, settings scene, persistence, networking,
browser APIs, subprocesses, IPC, Go FFI or third-party dependency.

## Phase 8C Overview Fixture Requirements

Phase 8C implements only:

- `Tổng quan` dashboard and completed batch summary;
- one deterministic sample batch for the MacBook profile;
- exactly five synthetic groups represented by summary counts;
- sample/demo labeling visible in the dashboard;
- display-only fixture labels that are not Go reason codes;
- no lead cards, Dry Run records, group management, blocklist behavior,
  settings behavior or mutable workflow state.

The other sidebar sections remain Phase 8B placeholders. Fixture data is
declared in Swift source as immutable value types. It is not loaded from files,
does not come from Facebook, does not use current time or randomness, and is
not a future bridge schema.

## Phase 8D Leads Fixture Requirements

Phase 8D implements only:

- `Leads` tabs `Tất cả`, `Đủ điều kiện` and `Cần xem xét`;
- exactly four synthetic buyer lead cards declared in Swift fixture source;
- fixed fixture identities, groups, dates, locations, source counts and reason
  code strings;
- tab counts and filtering derived only from each lead's declared fixture
  category;
- disabled placeholder actions only.

The fixture uses existing repository reason-code strings as display values, but
Swift does not infer, generate or validate business reasons. The Leads screen is
buyer-only and must not add seller presentation, seller tabs or seller workflow.
Dry Run, Nhóm, Blocklist and Cài đặt remain Phase 8B placeholders. Fixture data
is not loaded from files, does not come from Facebook, does not use current time
or randomness, and is not a future bridge schema.

## Phase 8E Dry Run Fixture Requirements

Phase 8E implements only:

- `Dry Run` tabs `Được chọn`, `Cần xem xét` and `Đã loại`;
- exactly ten synthetic MacBook-related post cards declared in Swift fixture
  source;
- fixed fixture authors, groups, dates, locations and reason code strings;
- tab counts and filtering derived only from each post's declared fixture
  category;
- one disabled placeholder action only.

The fixture uses existing repository reason-code strings as display values, but
Swift does not infer, generate, validate or recompute business reasons. The Dry
Run screen is presentation-only and must not add restore, approve, reject,
edit, persistence or scan behavior. Nhóm, Blocklist and Cài đặt remain Phase 8B
placeholders. Fixture data is not loaded from files, does not come from
Facebook, does not use current time or randomness, and is not a future bridge
schema.

## Product UI Structure

The intended high-level screens are:

- Overview / batch summary;
- Leads;
- Dry Run review;
- Groups;
- Blocklist;
- Settings.

Groups may remain placeholder until Phase 9. Facebook scan controls remain
disabled or absent until later phases. Dry Run is default-on in product
behavior. Seller tabs and seller mode are forbidden.

## Fixture Policy

- Fixtures must be deterministic.
- Fixtures must contain no real Facebook credentials, cookies, tokens, or private browser data.
- Fixtures may contain synthetic Vietnamese names and post text.
- Vietnamese diacritics must be preserved.
- Fixture UI must not claim to be live Facebook data.
- Fixture data should be clearly labeled as sample/demo data in manual validation builds.
- Phase 8C Overview, Phase 8D Leads and Phase 8E Dry Run fixtures must also
  state that they are not connected to the Go core and do not come from
  Facebook.

## UI Dependency Rules

- SwiftUI layer may depend conceptually on future narrow application/orchestration adapters.
- SwiftUI must not import or mirror Facebook adapter internals.
- SwiftUI must not directly access SQLite.
- Go application/domain packages must not depend on Swift or macOS UI code.
- Existing Go architecture tests remain authoritative.

## Security And Privacy

- No Facebook credentials.
- No cookies or browser profile copies.
- No automatic login.
- No hidden networking.
- No cloud sync.
- No telemetry in Phase 8.
- No user content leaves the Mac.
- Fixture screenshots or logs must avoid real user data.

## Testing Strategy

Future SwiftUI milestones must include:

- Xcode build;
- Swift tests where logic exists;
- preview or fixture rendering where appropriate;
- manual app launch;
- stale app process termination before relaunch;
- verification that the rebuilt app bundle is the one being tested;
- no regression to Go tests:
  - `go test ./...`
  - `go vet ./...`
  - Go CLI still builds and prints `ScanFB foundation ready`.

## Deferred Decisions

- Bridge mechanism.
- Deployment target beyond the Phase 8B minimum.
- Signing.
- Notarization.
- Packaging.
- App sandbox entitlements.
- Database production path.
- Persistence wiring.
- Real Facebook adapter.
- Status persistence.
- Search/list APIs.
- App updates.
- Telemetry.
- Localization beyond current Vietnamese-first product text.

## Explicit Non-Goals

- No production UI features.
- No Swift package.
- No Go code.
- No dependency.
- No bridge.
- No database.
- No Facebook integration.
- No production scan workflow.
- No seller mode.
