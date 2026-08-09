# macOS UI Architecture

## Purpose

This document records the native macOS presentation architecture for Phase 8.
It records the Phase 8A architecture decision, the Phase 8B app-shell state,
the Phase 8C static Overview fixture state, the Phase 8D fixture-only Leads
presentation state, the Phase 8E fixture-only Dry Run review state, the
Phase 8F fixture-only Blocklist/Settings state, the Phase 8G session-only
Leads interaction state, the Phase 8H fixture source URL browser handoff, the
Phase 8I.2 readiness-only bridge status row and Phase 8I.2a Debug helper
packaging. Phase 8C/8D/8E/8F/8G/8H implement only sample presentation screens;
Phase 8I.2 implements only a transport readiness check, and Phase 8I.2a only
packages that helper for Debug builds. Neither adds persistence wiring, Facebook
integration, production UI features or live lead data behavior.

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

## Bridge Decision Boundary

Phase 8I.1 selects exactly one narrow integration model: local subprocess
request/response between SwiftUI and a bundled Go helper. Phase 8I.2 implements
only the first read-only `core_readiness` slice, and Phase 8I.2a packages that
helper into Debug app bundles. The decision is documented in
[BRIDGE_DECISION.md](BRIDGE_DECISION.md). Phase 8I.2a does not add a socket,
generated binding, broad schema, lead/search API, Release packaging or
production signing/notarization policy.

Candidate categories evaluated:

- local subprocess with structured typed request/response;
- local IPC boundary;
- C-compatible exported Go library.

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
- broad command bus APIs.

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
- Phase 8F: fixture settings and blocklist presentation. Implemented for
  `Blocklist` and `Cài đặt` only with typed immutable fixture data, disabled
  blocklist actions, read-only settings rows, no display-name-only block
  identity, no bridge, database, Facebook, networking, persistence,
  credentials, cookies, import/export, blocklist writes or settings writes.
- Phase 8G: UI interaction state for viewed/ignored using in-memory fixture state only.
  Implemented for `Leads` only with `new/viewed/ignored` presentation
  state scoped to the current SwiftUI session. It resets on app restart and
  does not persist, bridge, recompute eligibility or recompute reason codes.
- Phase 8H: lead interaction browser handoff. Implemented for `Leads` only with
  deterministic synthetic HTTPS source URLs, validation and SwiftUI `openURL`;
  `Tương tác` is an action, not a state, and does not mutate interaction state.
- Phase 8I.1: bridge evaluation and decision only. Selects local subprocess
  request/response for typed SwiftUI-Go slices.
- Phase 8I.2: first read-only bridge slice. Implements only explicit
  user-triggered Go core readiness in `Cài đặt` -> Integration status, with no
  auto-run, polling, product data, Facebook, persistence or mutation.
- Phase 8I.2a: Debug helper packaging only. Builds the existing Go helper during
  Debug app builds and copies exactly `scanfb-bridge-helper` into
  `Contents/Helpers`.
- Phase 8I.3+: later narrow Go integration milestones, each separately scoped.

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

## Phase 8F Blocklist And Settings Fixture Requirements

Phase 8F implements only:

- `Blocklist` sample entries with exactly four synthetic identities;
- `Cài đặt` read-only sample sections for scan defaults, privacy, data/storage
  and integration status;
- visible `Dữ liệu minh họa` labels and sample-only disclaimers;
- disabled blocklist actions only.

Blocklist fixture identity kinds must align with authoritative Go support:
Facebook user ID, canonical profile URL and username. Display name is only a
visible label and is not an authoritative block identity. The Settings fixture
is read-only presentation and must not add toggles, forms, save/reset behavior,
AppStorage, UserDefaults, persistence writes, direct SQLite access, bridge,
Facebook integration, credentials or cookies. Nhóm remains placeholder.
Fixture data is not loaded from files, does not come from Facebook, does not
use current time or randomness, and is not a future bridge schema.

## Phase 8I.2 Readiness Bridge Requirements

Phase 8I.2 implements only:

- `Cài đặt` -> Integration status -> Go bridge readiness check;
- one button labeled `Kiểm tra kết nối`;
- display states `Chưa kiểm tra`, `Đang kiểm tra`, `Sẵn sàng` and `Lỗi`;
- one local subprocess helper invocation per user-triggered check;
- one request schema with `schema_version` and `operation`;
- one response schema with `schema_version`, `readiness_status` and
  `core_identity`.

The helper operation is exactly `core_readiness`. The response values are
`readiness_status` `ready` or `error`, and `core_identity` `scanfb-core`.
Swift resolves the bundled helper explicitly at
`Contents/Helpers/scanfb-bridge-helper` and fails closed if it is absent; no
developer-machine absolute path, `/tmp` fallback or PATH search is used. Phase
8I.2a packages this helper for Debug app builds only. The readiness call has a
2.0 second timeout and terminates the owned helper on timeout/cancellation, with
a 0.5 second force-kill grace. stdout is reserved for the bounded
machine-readable response; stderr is diagnostics-only, bounded and not shown raw
in UI.

Phase 8I.2 must not add auto-run, polling, networking, sockets, shell
invocation, direct Swift SQLite, Facebook SDK/API, WebKit, browser/session
access, credentials, cookies, persistence writes, settings writes, lead/search
data, SearchProfile data, capability inventory, business-rule summaries or a
generic bridge command bus.

Phase 8I.2a uses one target-local Xcode Debug shell build phase because Xcode has
no native Go command build rule. The phase builds into DerivedData under
`DERIVED_FILE_DIR`, copies only the helper executable into
`TARGET_BUILD_DIR/CONTENTS_FOLDER_PATH/Helpers`, preserves executable
permission, and does not create repo-local binaries. It may use an explicit
`GO_EXECUTABLE` build setting or checked conventional Go install paths; it must
not search arbitrary paths or use PATH fallback.

## Phase 8G/8H Leads Interaction State And Browser Handoff Requirements

Phase 8G/8H implements only:

- session-memory interaction state for existing fixture Leads;
- supported states `new`, `viewed` and `ignored`;
- visible Vietnamese labels `Mới`, `Đã xem` and `Bỏ qua`;
- compact card actions for `Đánh dấu đã xem`, `Tương tác` and `Bỏ qua`;
- deterministic synthetic HTTPS source URL values for existing fixture leads;
- browser handoff through SwiftUI `openURL` only after URL validation.

All fixture leads start as `new`. `viewed/ignored` are SwiftUI
presentation states only, not persisted domain statuses and not a future bridge
schema. The state resets when the app process restarts. Valid transitions are
`new -> viewed`, `new -> ignored` and `viewed -> ignored`; marking a lead as
viewed again leaves it unchanged. `Tương tác` is a stateless action: it hands a
validated source URL to the macOS default browser and does not change the lead's
state. Opening that URL does not imply the user liked, commented, messaged or
completed any action on Facebook. Interaction state must not change the existing
`Tất cả`, `Đủ điều kiện` or `Cần xem xét` tab filtering, eligibility category,
reason codes or fixture order. Phase 8H must not add persistence, AppStorage,
UserDefaults, direct SQLite access, bridge, Facebook SDK/API, WebKit, embedded
browser, browser automation, networking client, timestamps, history, credential
access, cookie access or production workflow.

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
- Phase 8H fixture source URLs must be synthetic and must not claim to be real
  buyer posts. Future authoritative source URLs come from Go-backed lead data.
- Phase 8C Overview, Phase 8D Leads, Phase 8E Dry Run and Phase 8F
  Blocklist/Settings fixtures must also
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
- The default browser owns Facebook login, session and cookies.
- No automated Facebook like, comment, message or profile inspection.
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

- Bridge implementation.
- Deployment target beyond the Phase 8B minimum.
- Signing.
- Notarization.
- Release helper packaging.
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
- No Go code beyond the Phase 8I.2 readiness-only helper/core adapter.
- No dependency.
- No bridge beyond Phase 8I.2 `core_readiness`.
- No database.
- No Facebook integration.
- No production scan workflow.
- No seller mode.
