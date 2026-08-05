# macOS UI Architecture

## Purpose

This document records the native macOS presentation architecture for Phase 8.
It is an architecture decision and app-shell plan only. It does not implement
UI code, app packaging, a bridge, persistence wiring, or Facebook integration.

## Presentation Technology

SwiftUI is the approved presentation technology for the future native macOS app.
The app is Apple Silicon first. The minimum macOS deployment target remains
intentionally undecided until the app-shell milestone.

## Repository Layout

- Future native app root: `macos/ScanFBApp/`
- The Go core remains under the current Go module.
- SwiftUI code must not be placed in `internal/ui`.
- This milestone creates no new repository folder.

`internal/ui` remains only a Go-layer documentation/package placeholder until a
future narrow integration milestone decides whether it is still needed.

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
No bridge mechanism is selected or implemented in Phase 8A.

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
- Phase 8B: empty native SwiftUI app shell only.
- Phase 8C: static fixture dashboard and batch summary.
- Phase 8D: fixture lead tabs and lead cards.
- Phase 8E: fixture Dry Run review.
- Phase 8F: fixture settings and blocklist presentation.
- Phase 8G: UI interaction state for viewed/contacted/ignored using in-memory fixture state only.
- Phase 8H: bridge evaluation and architecture decision.
- Phase 8I+: only after bridge decision, narrow Go integration milestones.

Each milestone must be independently buildable and manually testable.

## Phase 8B Initial App-Shell Requirements

Future Phase 8B documents and implements only:

- native SwiftUI macOS app;
- one main window;
- app name ScanFB;
- placeholder navigation;
- no Go bridge;
- no database;
- no Facebook;
- no networking;
- no third-party dependency;
- no production settings;
- no persistence;
- no generated fixture business data beyond minimal visual placeholders.

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
- Deployment target.
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

- No UI code.
- No Xcode project.
- No Swift package.
- No Go code.
- No dependency.
- No bridge.
- No database.
- No Facebook integration.
- No production scan workflow.
- No seller mode.
