# Code Graph

Tai lieu nay mo ta graph kien truc muc tieu, module boundaries va dependency direction. Phase 2 da co Go domain types toi thieu nhung chua gia dinh codebase-memory-mcp/codegraph da duoc cau hinh va khong tu cai tool.

## Mermaid dependency graph

```mermaid
flowchart TD
    UI["App/UI"] --> APP["Application services"]
    UI --> ORCH["Use-case orchestration"]
    ORCH --> APP
    ORCH --> PERSIST_CONTRACT
    APP --> DOMAIN["Domain"]
    PERSIST_CONTRACT["Persistence-facing contracts"] --> APP
    PERSIST_CONTRACT --> DOMAIN
    PERSIST_IMPL["Persistence implementation"] --> PERSIST_CONTRACT
    SQLITE["SQLite schema bootstrap, SaveBatch, and concrete LoadBatch"] --> PERSIST_CONTRACT
    WATCHED_STATE["Phase 9E2 dedicated WatchedGroup-state SQLite v1"] --> PERSIST_IMPL
    MACOS_APP["macos/ScanFBApp SwiftUI shell"] -.future bridge.-> ORCH
    BRIDGE_HELPER["cmd/scanfb-bridge-helper typed operations"] -.typed request/response.-> BRIDGE_CORE
    BRIDGE_CORE["internal/bridge readiness and watched groups"]
    MACOS_APP -.local subprocess.-> BRIDGE_HELPER
    GROUPS_UI["Phase 9E3b enrolled Watched Groups presentation"] --> MACOS_APP
    OVERVIEW_FIXTURE["Overview fixture model and views"] --> MACOS_APP
    LEADS_FIXTURE["Leads fixture model and views"] --> MACOS_APP
    LEADS_INTERACTION_STATE["Leads session interaction state"] --> LEADS_FIXTURE
    LEADS_BROWSER_HANDOFF["Leads fixture source URL browser handoff"] --> LEADS_FIXTURE
    DRYRUN_FIXTURE["Dry Run fixture model and views"] --> MACOS_APP
    BLOCKLIST_FIXTURE["Blocklist fixture model and views"] --> MACOS_APP
    SETTINGS_FIXTURE["Settings fixture model and views"] --> MACOS_APP
    FB["Facebook adapter"] --> APP
    FB --> RAW["RawPost mapping contract"]
    SAFARI_ACTIVE_TAB["Phase 10B1 Safari active-tab snapshot"] --> FB
    DOM_RECON["Phase 10B2a source reconnaissance: blocked"] -.documents missing post DOM.-> SAFARI_ACTIVE_TAB
    RENDERED_DOM_DECISION["Phase 10B2c rendered-DOM decision"] -.approves future bounded Apple Events probe.-> SAFARI_ACTIVE_TAB
    RENDERED_DOM["Phase 10B2d bounded rendered-DOM snapshot"] --> FB
    RENDERED_DOM_DECISION -.implemented by.-> RENDERED_DOM
    JOINED_GROUP_RECON["Phase 9E3a joined-groups reconnaissance: inconclusive"] -.blocks discovery edge.-> RENDERED_DOM
    RENDERED_DOM_RECON["Phase 10B2e rendered-DOM reconnaissance: inconclusive"] -.retained no post-level evidence.-> RENDERED_DOM
    REDACTED_RECON_REPORT["Phase 10B2f bounded redacted evidence report"] -.future manual input.-> RENDERED_DOM_RECON
    RENDERED_DOM_CLOSEOUT["Phase 10B2g selector investigation closeout"] -.blocks selector edge.-> REDACTED_RECON_REPORT
    PREPARED_PAGE["Phase 10A typed local prepared snapshot"] --> PREPARED_EXTRACTOR["Fail-closed fixture extractor"]
    PREPARED_EXTRACTOR --> RAW
    RAW --> APP
    APP --> LIFECYCLE["Phase 9A in-memory batch lifecycle"]
    APP --> WATCHED_GROUPS["Phase 9B in-memory WatchedGroup collection"]
    WATCHED_GROUPS --> DOMAIN
    APP --> GROUP_SELECTOR["Phase 9C active-only circular five-group selector"]
    GROUP_SELECTOR --> DOMAIN
    BRIDGE_CORE --> WATCHED_GROUPS
    BRIDGE_CORE --> GROUP_SELECTOR
    BRIDGE_CORE --> WATCHED_STATE
    GROUP_SELECTOR --> LIFECYCLE_MAPPER["Phase 9D selection-to-lifecycle mapper"]
    LIFECYCLE_MAPPER --> LIFECYCLE
    PROFILE["SearchProfile"] --> DOMAIN
    DOMAIN --> TYPES["Domain types and reason codes"]
    BLOCKLIST["Blocklist identity primitives"] --> DOMAIN
```

## Module ownership

- `cmd/scanfb`: binary entry point toi thieu.
- `internal/domain`: Go package cho domain ownership.
- `internal/application`: Go package cho application service ownership.
- `internal/blocklist`: Go package cho deterministic local blocklist identity primitives.
- `internal/rules`: Go package cho deterministic buyer rules ownership.
- `internal/dedup`: Go package cho deduplication ownership.
- `internal/persistence`: Go package cho completed-batch persistence va independent Phase 9E2 WatchedGroup-state SQLite schema v1/repository ownership.
- `internal/orchestration`: Go package cho thin synchronous use-case orchestration ownership.
- `internal/facebook`: Go package cho Facebook/browser adapter boundary ownership.
- `internal/ui`: Go-layer documentation/package placeholder; not the SwiftUI app root.
- `internal/bridge`: bounded typed adapter for Phase 8I.2 `core_readiness` and Phase 9E2 persistent watched-group list/add/set-active/next-five operations. It uses the Go-owned repository plus Phase 9B/9C services, exposes no raw path/SQL/client snapshot authority, and does not import Facebook.
- `cmd/scanfb-bridge-helper`: one-request local subprocess helper that reads stdin, writes the machine response to stdout, writes bounded diagnostics to stderr and exits. Phase 8I.2a builds it during Debug app builds and copies it to `Contents/Helpers/scanfb-bridge-helper`.
- `macos/ScanFBApp`: SwiftUI native macOS app shell implemented in Phase 8B, with Phase 8 fixture screens, Phase 8I.2 Settings readiness row and Phase 9E2 persistent Watched Groups presentation.
- App/UI: owner cua views, tabs, lead cards, settings va user actions.
- Application services: owner cua deterministic in-memory scan batch model, Phase 9A group-attempt lifecycle state machine, Phase 9B WatchedGroup collection, Phase 9C five-group selection policy, Phase 9D selection-to-lifecycle mapping, batch state va time window.
- Use-case orchestration: owner cua glue logic giua completed application result, `BatchRecord` conversion va repository save boundary.
- Domain: owner cua normalization contracts, SearchProfile, BuyerIntentClassifier, rule engine, geographic classifier, deduplication, lead aggregation va reason codes.
- Persistence-facing contracts: owner cua completed batch snapshot contracts.
- Persistence implementation: owner cua local storage implementation.
- WatchedGroup persistence: Phase 9E2 Go owner cua Application Support path, separate `watched-groups.sqlite3` schema v1, DELETE journal, full Phase 9B value plus Phase 9C cursor transaction boundary and fail-closed restore.
- Facebook adapter: owner cua Phase 10B1 Safari-only user-triggered current-tab URL/title/bounded-source acquisition va fail-closed adapter errors; production DOM parsing/selector validation van deferred.
- Prepared-page fixture extractor: Phase 10A owner cua typed local snapshot validation va deterministic ordered `RawPost` mapping; no live DOM/browser edge.
- Safari active-tab acquisition: Phase 10B1 owner cua direct `/usr/bin/osascript` JXA call, caller-supplied capture time, bounded stdout/stderr, HTTPS URL validation va timeout/cancellation; no edge toi `RawPost`, application pipeline, persistence, SwiftUI hoac bridge.
- Safari rendered-DOM acquisition: Phase 10B2d owner cua fixed read-only page-side JavaScript qua direct `/usr/bin/osascript`, exactly one current tab, caller-supplied capture time, 8 MiB decoded DOM, finite stdout envelope va fail-closed process/permission/content errors; no edge toi `RawPost`, selector, application pipeline, persistence, SwiftUI hoac bridge.
- SwiftUI shell: owner cua native windows, navigation, presentation state va accessibility.
- Overview fixture model/views: SwiftUI-only presentation nodes for Phase 8C sample dashboard; no edge to Go core.
- Leads fixture model/views: SwiftUI-only presentation nodes for Phase 8D buyer lead tabs/cards; no edge to Go core.
- Leads session interaction state: SwiftUI-only presentation state for Phase 8G/8H `new/viewed/ignored`; reset on app restart and not a persisted domain status.
- Leads fixture source URL browser handoff: SwiftUI-only Phase 8H action that validates a synthetic HTTPS source URL and hands it to the macOS default browser without mutating lead state.
- Dry Run fixture model/views: SwiftUI-only presentation nodes for Phase 8E review tabs/cards; no edge to Go core.
- Blocklist fixture model/views: SwiftUI-only presentation nodes for Phase 8F sample blocklist entries; no edge to Go core and no ownership of blocklist semantics.
- Settings fixture model/views: SwiftUI presentation nodes for Phase 8F read-only sample settings plus Phase 8I.2 readiness-only Go bridge check; no persistence writes and no lead/search/product data.
- Watched Groups UI/store: Phase 9E2/9E3b SwiftUI presentation owner. It exposes one-time group enrollment through the existing authoritative Add operation, displays bridge-returned state, keeps active toggles and a read-only next-five preview, and has no manual queue advance. It has no SQLite/path, Facebook discovery or scan-execution edge.

Phase 2 files trong `internal/domain` gom minimal models cho `RawPost`, `AuthorIdentity`, `SearchProfile`, `GeographicMode`, `ScanWindow` va `ScanRequest`. Domain package chi duoc import Go standard library.

Phase 3A files trong `internal/rules` gom deterministic primitives cho post-time eligibility va author exclusion. Phase 3B them deterministic buyer-intent va seller/noise text matching dua tren active `SearchProfile`. Rules package chi duoc import Go standard library va `internal/domain`.

Phase 3C them `internal/rules/geography.go` va `internal/rules/geography_test.go` cho finite MVP geographic classification, `GeographicMode` evaluation va composed buyer-search-with-geography evaluation. Khong co geocoder, location database, foreign classifier hoac foreign exclusion.

Phase 4A them `internal/dedup/identity.go`, `internal/dedup/compare.go` va `internal/dedup/identity_test.go` cho stable author key, buyer need key, candidate key va deterministic duplicate comparison primitives. Chua co lead aggregation, persistence hoac source-post merging.

Phase 4B them `internal/dedup/aggregate.go` va `internal/dedup/aggregate_test.go` cho deterministic in-memory lead aggregation. Moi source post duoc preserve bang full `RawPost`; post khong auto-aggregated va source conflicts duoc tra ve explicit. Chua co persistence, repository, Facebook adapter, UI hoac scan orchestration.

Phase 4C them `internal/blocklist` cho local deterministic blocklist identity primitives. Supported stable identity kinds la Facebook user ID, canonical profile URL va username. Matching dung strongest available stable author identity, exact same-kind normalized key va fail closed khi thieu stable identity. Display name chi la metadata, khong duoc dung de block. Chua co persistence, scan orchestration, application integration, UI hoac CLI.

Phase 4D them `internal/application/lead_filter.go` cho in-memory filtering cua aggregated buyer leads bang local blocklist. Ket qua tach explicit allowed, blocked va unresolved leads; blocked/unresolved leads van giu nguyen source posts. Chua co persistence, scan orchestration, raw-post rule evaluation, UI hoac CLI wiring.

Phase 5A them `internal/application/evaluation_pipeline.go` cho deterministic in-memory pipeline tu already-collected `RawPost` values qua rules, eligible selection, dedup aggregation va blocklist filtering. Pipeline preserve evaluated, eligible, review, excluded, unaggregated, conflicts, allowed, blocked va unresolved outputs. Chua co Facebook adapter, persistence, UI, CLI behavior, scheduling, concurrency hoac network behavior.

Phase 5B them `internal/application/scan_batch.go` cho deterministic in-memory manual batch model gom mot den nam explicit groups. Batch validate group identity va post/group consistency, flatten posts theo group order roi post order, goi Phase 5A pipeline mot lan va tao batch/per-group count summaries. Chua co Facebook collection, persistence, UI, CLI behavior, scheduling, retries, progress reporting, concurrency hoac network behavior.

Phase 9A them `internal/application/group_lifecycle.go` cho deterministic in-memory lifecycle cua mot production-shaped scan batch dung dung 5 groups. Lifecycle preserve caller order, dung caller-supplied batch/attempt IDs, state set chi gom `pending`, `running`, `succeeded`, `failed`, `skipped` va `expired_at_day_boundary`, enforce one-running-at-a-time, khong auto retry, khong auto tao batch moi, va dung supplied time theo `Asia/Ho_Chi_Minh` cho day-boundary expiration. Chua co Facebook collection, persistence, SQLite schema/table, UI, bridge operation, scheduler, retry, goroutine/concurrency hoac call toi Phase 5B `RunScanBatch`.

Phase 9B them `internal/domain/watched_group.go` va `internal/application/watched_group_collection.go` cho caller-supplied WatchedGroup identity, metadata, active/inactive lifecycle va deterministic in-memory insertion-order inspection. Collection khong gioi han tong so group, khong chon next five, khong dinh nghia queue ordering/cursor/rotation, khong goi `NewScanBatchLifecycle`, va khong co Facebook, persistence, SQLite, SwiftUI, bridge, scheduler, retry, goroutine/concurrency, networking hoac generated ID.

Phase 9C them `internal/application/five_group_selection.go` cho pure deterministic selection tu WatchedGroup snapshot. Selector bat dau tai explicit caller-managed collection-position cursor, traverses toi da mot circular cycle theo insertion order, skip inactive, tra dung 5 distinct active groups va next cursor ngay sau group thu nam. Insufficient active groups fail closed; display/time metadata khong sort; cursor khong persist; khong co lifecycle construction, generated ID, scan execution, Facebook, persistence, SQLite, SwiftUI, bridge, scheduler, retry, goroutine/concurrency hoac networking.

Phase 9D them `internal/application/selection_lifecycle.go` cho deterministic mapping tu mot approved `FiveGroupSelection` sang `ScanBatchLifecycle`. Mapper preserve exact selection order, ghep dung 5 caller-supplied attempt ID theo index va delegate batch ID, `ScanWindow` va attempt validation cho Phase 9A constructor. Mapper khong re-select, khong doc collection, khong dung/advance cursor, khong start attempt, khong chay scan, khong tao ID va khong co Facebook, persistence, SQLite, SwiftUI, bridge, scheduler, retry, goroutine/concurrency hoac networking.

Phase 9E1 them typed watched-group operations trong `internal/bridge`, `WatchedGroupsStore` va UI tai `macos/ScanFBApp`. Swift giu snapshot/cursor chi cho current session va gui chung tren moi one-shot helper call; Go tai tao Phase 9B collection, ap dung add/active mutation va goi Phase 9C selector de tra exact bridge order. UI khong tu sort/chon group, va slice khong co lifecycle, scan execution, Facebook, persistence, SQLite, scheduler, retry, goroutine/concurrency hoac networking.

Phase 9E2a them docs-only [WATCHED_GROUP_PERSISTENCE_DECISION.md](WATCHED_GROUP_PERSISTENCE_DECISION.md). Decision approve hai Go-owned SQLite databases duoi `<user-application-support>/com.soleda.ScanFB/`: `completed-batches.sqlite3` giu schema v1 hien tai va `watched-groups.sqlite3` bat dau schema v1 rieng. Phase 9E2a chua co runtime edge; Phase 9E2 ben duoi trien khai decision nay.

Phase 9E2 them `SQLiteWatchedGroupRepository`, independent schema v1 va Go Application Support resolver. Repository restore full groups theo explicit insertion position, validate authoritative identity/metadata/cursor, dung DELETE journal, va transact add/set-active/cursor advance. Watched-group bridge v2 khong nhan client collection/cursor/path; Swift store chi render authoritative response va explicit loading/storage-error states. Existing completed-batch schema v1 khong doi; khong co migration runner, Facebook, scan execution, scheduler, retry, networking hoac generated ID.

Phase 9E2b historically corrected the primary UI toward future joined-group discovery without a new runtime edge. Phase 9E3b supersedes that product assumption: existing Add is now the primary one-time enrollment path for user-approved groups, while `watched_groups_next_five` remains a future internal progression primitive. Rendering list/preview never advances cursor.

Phase 9E3a adds only [FACEBOOK_JOINED_GROUPS_RECONNAISSANCE.md](FACEBOOK_JOINED_GROUPS_RECONNAISSANCE.md) after exactly one user-guided call to the existing Phase 10B2d API. Redacted evidence found one canonical group-link/name association and one tentative containing section, but no semantic group-item container, explicit joined-membership marker or strong multi-item traversal. The dotted reconnaissance edge adds no parser/discovery/runtime behavior and blocks Phase 9E3; there is no edge to WatchedGroup persistence, cursor, bridge/UI, `RawPost`, scan or Phase 11/12.

Phase 9E3b changes only product copy, the existing Groups presentation and focused Swift source checks. Enrollment calls the unchanged `watched_groups_add` bridge operation and consumes its full authoritative state/selection response. It adds no schema, persistence owner, discovery/acquisition edge, membership verification, cursor advancement or Phase 11/12 execution.

Phase 10A them `internal/facebook/prepared_page.go` cho typed local `PreparedPageSnapshot` va `ExtractPreparedPage`. Extractor validate schema version, caller-supplied group/capture metadata, body, absolute RFC3339 timestamp, optional absolute HTTPS post URL va embedded group consistency; output la ordered `[]domain.RawPost`. Phase nay khong parse live Facebook DOM, khong acquire browser page, khong co cookie/credential/session/network, khong goi scan/lifecycle va khong co persistence, SwiftUI hoac bridge behavior.

Phase 10B1 them `internal/facebook/safari_active_tab.go` cho mot Safari-only acquisition boundary. `AcquireSafariActiveTab` goi truc tiep `/usr/bin/osascript` bang JXA de doc URL, optional title va bounded source cua dung current tab trong front Safari window; timestamp do caller cung cap. Boundary validate absolute HTTPS URL, gioi han decoded content 4 MiB, tach stdout/stderr, map explicit process/TCC/tab/content errors va own timeout/cancellation process. No khong auto-login/navigation/tab switching/scrolling/polling, khong doc browser secrets/profile stores, khong dung network/Accessibility/WebKit/extension, khong tao `RawPost` va khong goi pipeline/lifecycle/persistence/SwiftUI/bridge. Production selector validation thuoc Phase 10B2.

Phase 10B2a them docs-only [FACEBOOK_SAFARI_DOM_RECONNAISSANCE.md](FACEBOOK_SAFARI_DOM_RECONNAISSANCE.md) sau exactly one read-only acquisition cua user-prepared group page. Active URL xac nhan page identity, nhung `tab.source()` khong expose post container, permalink/ID, body, author hoac absolute timestamp markers. Reconnaissance blocked/inconclusive, khong them selector/parser/`RawPost` edge va khong cho phep Phase 10B2b tren source acquisition hien tai.

Phase 10B2c them docs-only [SAFARI_RENDERED_DOM_ACQUISITION_DECISION.md](SAFARI_RENDERED_DOM_ACQUISITION_DECISION.md). Decision approve mot future acquisition-only node dung fixed read-only page-side JavaScript qua Safari Apple Events tren exactly one current tab, bounded va fail closed. Dotted decision edge khong phai runtime edge: chua co implementation, entitlement, Xcode change, selector, `RawPost`, pipeline, persistence, SwiftUI/bridge behavior, extension, Accessibility, WebKit hoac network listener.

Phase 10B2d them `internal/facebook/safari_rendered_dom.go` cho separate acquisition-only runtime node. `AcquireSafariActiveTabRenderedDOM` executes one fixed outerHTML expression tren front-window current tab, preserves exact HTTPS URL/title/rendered document/caller timestamp, caps decoded DOM tai 8 MiB va stdout transport tai 50,397,184 bytes, va reuse owned-process timeout/cancellation plus bounded stderr. Node khong parse Facebook structure, khong tao selector/`RawPost`, khong goi Phase 10A/11 va khong co browser mutation, private browser state, network/listener, Accessibility/System Events, extension/WebKit, persistence, SwiftUI hoac bridge edge. Automated tests va user-guided live validation pass; Phase 10B2e separate redacted reconnaissance stops inconclusive.

Phase 10B2e them docs-only [FACEBOOK_SAFARI_RENDERED_DOM_RECONNAISSANCE.md](FACEBOOK_SAFARI_RENDERED_DOM_RECONNAISSANCE.md) sau exactly one successful production acquisition tren user-confirmed Facebook group tab. Temporary in-memory analyzer output was filtered to a pass summary, nen no post-container/permalink/body/author/machine-time/traversal counts survived. Reconnaissance stops inconclusive, adds no selector/`RawPost` edge and keeps Phase 10B2b blocked.

Phase 10B2f them `internal/facebook/rendered_dom_reconnaissance.go` cho pure bounded evidence-report node. `AnalyzeRenderedDOMStructure` consumes only rendered DOM plus optional page URL and returns count/shape/confidence-only `RenderedDOMReconnaissanceReport`; marker arrays toi da 16 canonical strings, moi string toi da 64 bytes. Node khong acquire Safari, khong write file, khong return private match, khong implement selector/`RawPost`, va khong co edge toi Phase 10A/11, persistence, SwiftUI, bridge hoac network. Future manual helper co the encode only typed report vao mode-0600 `/tmp` file de preserve evidence independent of RTK stdout.

Phase 10B2g records one preserved live report: 3,180,722 analyzed bytes, two semantic article candidates and traversal count two, but zero approved permalink/body/author/machine-time/complete-evidence candidates. Only `role=article` and `dom-source-order` are recognized. The closeout adds no runtime edge and keeps Phase 10B2b blocked; current Safari selector investigation ends unless a separately justified technique supplies the missing critical structure.

Phase 5C them `internal/persistence/batch_record.go` cho completed scan batch snapshot contract. Contract gom opaque `BatchRecordID`, immutable-style `BatchRecord`, structural validation, deterministic converter tu `application.ScanBatchInput`/`ScanBatchResult` va save-only `BatchRepository.SaveBatch`. Chua co SQLite, schema, migration, file I/O, load/list/update/delete/search/paging API, ID generation, Facebook adapter, UI/CLI, concurrency hoac network behavior.

Phase 5D them `internal/persistence/in_memory_batch_repository.go` cho deterministic in-memory adapter satisfy `BatchRepository`. Adapter validate `BatchRecord`, reject duplicate ID without overwrite, preserve insertion order bang slice, dung map chi cho lookup, va expose concrete helpers `Count`, `Records`, `RecordByID`. `BatchRepository` van save-only; chua co durable storage, SQLite, SQL, schema, migration, JSON/file I/O, goroutine, network hoac ID generation.

Phase 5E them `internal/orchestration/run_and_save_scan_batch.go` cho synchronous use case `RunAndSaveScanBatch`. Use case validate repository boundary, chap nhan caller-supplied `BatchRecordID`, chay `application.RunScanBatch`, convert successful result bang `persistence.NewBatchRecord`, save dung mot lan qua `persistence.BatchRepository`, va chi tra result/record sau khi save thanh cong. Chua co UI/CLI wiring, Facebook collection, durable storage, concurrency, retry, generated ID hoac rule moi.

Phase 5F them documentation-only SQLite schema design tai `docs/PERSISTENCE_SCHEMA.md`. Phase 5G1 them `internal/persistence/sqlite_repository.go` va `internal/persistence/sqlite_schema.go` cho SQLite foundation: `modernc.org/sqlite` driver import chi trong persistence, explicit-path open/create, foreign-key enable/verify, transactional empty schema v1 creation, schema metadata validation va `Close`. Phase 5G2 them `SQLiteBatchRepository.SaveBatch(record BatchRecord) error`, nen SQLite adapter satisfy save-only `BatchRepository` va ghi mot completed snapshot vao schema version 1 trong mot transaction. Phase 5G3 them concrete-only `SQLiteBatchRepository.LoadBatch(id BatchRecordID) (BatchRecord, error)` de reconstruct mot snapshot trong read transaction, fail closed voi malformed schema/data va khong mo rong `BatchRepository`. Chua co list/update/delete/search/paging/migration execution, production DB path, UI/CLI wiring hoac Facebook behavior.

Phase 8A documents native macOS UI direction only. SwiftUI is approved as the presentation shell for a future app at `macos/ScanFBApp/`. No `macos/` directory, Xcode project, Swift package, Swift source, bridge, fixture UI, database path or app bundle is implemented in Phase 8A.

Phase 8B implements the empty native SwiftUI macOS app shell at `macos/ScanFBApp/`. The shell has one app target, one unit test target, one `WindowGroup`, a `NavigationSplitView`, six placeholder sections and no Go bridge, Facebook behavior, SQLite/direct persistence access, networking, fixture business data or production dependency on Go packages.

Phase 8C replaces only the `Tổng quan` placeholder with a static fixture dashboard and batch summary. The fixture model and Overview views are SwiftUI-only presentation code with explicit stable values, no filesystem fixture loading, no current time, no randomness, no package, no bridge and no dependency on Go packages.

Phase 8D replaces only the `Leads` placeholder with fixture-only buyer lead tabs/cards. The Leads fixture model and views are SwiftUI-only presentation code with explicit stable values, local tab filtering by declared fixture category, disabled placeholder actions, no filesystem fixture loading, no current time, no randomness, no package, no bridge and no dependency on Go packages. Dry Run, Nhóm, Blocklist and Cài đặt remain placeholders.

Phase 8E replaces only the `Dry Run` placeholder with fixture-only included/review/excluded post tabs/cards. The Dry Run fixture model and views are SwiftUI-only presentation code with explicit stable values, local tab filtering by declared fixture category, disabled placeholder action, no filesystem fixture loading, no current time, no randomness, no package, no bridge and no dependency on Go packages. Nhóm, Blocklist and Cài đặt remain placeholders.

Phase 8F replaces only the `Blocklist` and `Cài đặt` placeholders with fixture-only presentation screens. The Blocklist and Settings fixture models and views are SwiftUI-only presentation code with explicit stable values, disabled blocklist actions, read-only settings rows, no display-name-only block identity, no filesystem fixture loading, no current time, no randomness, no package, no bridge and no dependency on Go packages. Nhóm remains placeholder.

Phase 8G extends only the `Leads` fixture screen with session-memory interaction state. Phase 8H narrows that state model to `new`, `viewed` and `ignored`, and adds stateless `Tương tác` browser handoff for deterministic synthetic HTTPS fixture source URLs. The state is SwiftUI-only presentation state, starts as `new` for every fixture lead, can be changed only by viewed/ignored card actions, resets on app restart, does not change eligibility tabs/categories, does not recompute reasons, and has no persistence, database, bridge, Facebook SDK/API, WebKit, browser automation, networking client, timestamp, random value or package dependency.

Phase 8I.1 documents the bridge decision only. Phase 8I.2 implements the first
readiness-only local subprocess slice using typed versioned schemas, Phase
8I.2a adds Debug helper packaging, Phase 9E1 introduced four bounded
watched-group operations, and Phase 9E2 backs them with Go-owned local state.
No socket, C binding, network listener, new dependency,
generated code, Release packaging policy, lead/search bridge or broad command
bus exists.

## Allowed dependencies

- App/UI duoc goi Application services.
- Application services duoc goi Domain, Rules, Dedup va Blocklist.
- Persistence-facing contracts duoc goi Application services va Domain de copy completed batch snapshots.
- Persistence implementation duoc implement Persistence-facing contracts.
- SQLite schema bootstrap, durable `SaveBatch` va concrete SQLite `LoadBatch` duoc implement trong `internal/persistence`; orchestration van chi nhan interface `BatchRepository`, khong import concrete SQLite adapter.
- Facebook adapter duoc goi Application services bang adapter boundary.
- Tests duoc import Domain truc tiep de chay fixture deterministic.
- `internal/rules` duoc import `internal/domain`.
- `internal/dedup` duoc import `internal/domain`.
- `internal/blocklist` duoc import `internal/domain`.
- `internal/application` duoc import `internal/domain`, `internal/rules`, `internal/dedup` va `internal/blocklist`.
- `internal/persistence` duoc import `internal/application` va `internal/domain`.
- `modernc.org/sqlite` chi duoc import trong `internal/persistence`.
- `internal/orchestration` duoc import `internal/application` va `internal/persistence`.
- `macos/ScanFBApp` depends on Go only through typed local subprocess requests; it does not import Go packages or access SQLite directly.
- Future SwiftUI-Go bridge slices may use only the selected local subprocess request/response model with typed versioned payloads.
- Phase 9E2 adds only a narrow `internal/bridge` dependency on the dedicated Go WatchedGroup-state persistence boundary; bridge code exposes no raw SQL, filesystem paths or database-local IDs.
- Phase 8C Overview fixture views may depend only on local Swift value models and SwiftUI.
- Phase 8D Leads fixture views may depend only on local Swift value models and SwiftUI.
- Phase 8E Dry Run fixture views may depend only on local Swift value models and SwiftUI.
- Phase 8F Blocklist and Settings fixture views may depend only on local Swift value models and SwiftUI.
- Phase 8G/8H Leads interaction state and source URL handoff may depend only on local Swift value models and SwiftUI state/openURL.

## Forbidden dependencies

- Domain khong duoc import App/UI.
- Domain khong duoc import Facebook adapter.
- Domain khong duoc chua CSS selector, DOM traversal, browser automation hoac Facebook-specific parsing.
- Facebook adapter khong duoc chua buyer business rules hoac product-specific extraction.
- UI khong duoc tu quyet dinh include/exclude thay rule engine.
- Persistence implementation khong duoc dieu khien browser.
- Application, Domain, Rules, Dedup va Blocklist khong duoc import `internal/persistence`.
- Application, Domain, Rules, Dedup, Blocklist va Persistence khong duoc import `internal/orchestration`.
- `internal/orchestration` khong duoc import Facebook adapter, UI hoac CLI package.
- `internal/persistence` khong duoc import Facebook adapter, UI, CLI package hoac persistence implementation package.
- Khong package nao ngoai `internal/persistence` duoc import `modernc.org/sqlite`.
- Khong module nao trong ScanFB duoc them seller mode, `SellerLead`, `SellerIntentClassifier`, `LeadIntent` buyer/seller hoac `SELLER_SCAN`.
- SwiftUI code must not import or mirror Facebook adapter internals, directly access SQLite, expose database-local IDs, or reimplement Go business logic.
- SwiftUI code must not add localhost HTTP/WebSocket bridge, direct SQLite bridge, browser extension bridge, in-process Go binding or arbitrary command bus without a new architecture decision.
- Go application/domain packages must not depend on Swift or macOS UI code.

## Entry points

- User bam Scan: App/UI goi Application service tao `ScanSession` va `ScanBatch`.
- User bam `Tương tác`: App/UI validate source URL HTTPS tu fixture hien tai hoac future `LeadSource` data roi handoff cho macOS default browser; khong mutate lead state.
- User danh dau viewed/ignored: App/UI cap nhat presentation state trong session hien tai; future production state workflow can milestone rieng.
- User bo qua account: App/UI goi Application service tao `BlockedAuthor`.
- Facebook adapter: application service kich hoat adapter doc group hien tai trong batch.

## Data flow cho Scan batch

```text
User Scan
-> App/UI
-> Application services tao ScanSession voi startedAt va geographicMode
-> Gan built-in MacBook SearchProfile trong MVP
-> Chon dung 5 WatchedGroup cho ScanBatch
-> Voi tung group: GroupScanAttempt pending -> running
-> Facebook adapter doc page va tao RawPost
-> Application services dua RawPost vao domain pipeline
-> Luu RawPost, FilterDecision, Lead/LeadSource
-> GroupScanAttempt succeeded/failed/skipped/expired_at_day_boundary
-> Batch summary tra ve UI
```

## Data flow cho filter/dedup

```text
RawPost
-> In-memory ScanBatch group validation
-> Deterministic flattening theo group order va post order
-> Rule evaluation: time, author, SearchProfile target keyword, BuyerIntentClassifier va geographic classification
-> Eligible/review/excluded separation
-> In-memory buyer lead aggregation
-> Local blocklist lead filtering
-> Explicit allowed, blocked, unresolved, unaggregated va conflict outputs
-> Count-only batch summary va per-group rule-stage summaries
-> Persistence-facing completed BatchRecord snapshot contract
-> Optional in-memory BatchRepository adapter cho deterministic inspection/testing
-> Optional thin RunAndSaveScanBatch orchestration cho explicit caller-supplied record ID
-> Optional SQLite schema-bootstrap, SaveBatch and concrete LoadBatch adapter neu caller mo explicit local DB path
-> Phase 8C/8D/8E/8F/8G/8H SwiftUI fixture Overview, Leads, Dry Run, Blocklist, Settings, session-only Leads interaction presentation and stateless browser handoff, currently standalone until selected local subprocess bridge is implemented in a separate milestone
-> Phase 8I.1 selected future local subprocess typed request/response bridge; implementation deferred
```

## SearchProfile va buyer-only boundary

MVP chi co built-in MacBook Search Profile. Graph duoc giu trung lap cho phan scan/post/author/geo/dedup de sau nay co the them buyer Search Profile khac, nhung khong bien Phase hien tai thanh framework da san pham.

App tim nguoi can ban, neu co trong tuong lai, la du an/app khac. ScanFB hien tai khong co seller tab, seller mode selector, seller classifier hay enum intent buyer/seller.

## Cap nhat code graph khi code thay doi

Khi bat dau co code, moi thay doi module boundary phai cap nhat tai lieu nay neu:

- Them module moi.
- Doi dependency direction.
- Them entry point moi.
- Doi pipeline scan/filter/dedup.
- Them persistence hoac adapter implementation moi.
- Thay doi SQLite schema bootstrap, dependency boundary hoac schema version.

Khong cap nhat graph de hop thuc hoa dependency bi cam. Neu code can dependency bi cam, phai sua design hoac xin quyet dinh san pham/kien truc truoc.

## Huong dan dung codebase-memory-mcp/codegraph ve sau

Neu project sau nay duoc cau hinh codebase-memory-mcp hoac CodeGraph:

- Dung graph tools de tim symbol, module boundary va call path truoc khi grep.
- Dung grep/file search cho string literal, config, markdown hoac khi graph chua co ket qua.
- Khong tu cai, khoi tao hoac rebuild tool trong milestone khong cho phep.
- Khong gia dinh graph da dung neu `.codegraph/` hoac cau hinh MCP khong ton tai.
