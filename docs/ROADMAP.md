# Roadmap

Moi milestone phai co exact scope, protected areas, acceptance criteria, tests va stop conditions. Khong gop nhieu he thong lon trong mot moc.

## Phase 0 - Foundation docs

Exact scope: Tao README, agent instructions va tai lieu foundation.

Protected areas: Khong production code, khong dependency, khong Git write.

Acceptance criteria: Tai lieu day du, noi dung khong mau thuan, links noi bo hop le.

Tests: Read-only checks cho tree, links, placeholder va mau thuan ro.

Stop conditions: Yeu cau tao code, cai dependency hoac chon ngon ngu cuoi cung.

Future note sau MVP: co the them buyer Search Profile khac nhu iPhone, may anh hoac laptop chi sau khi MacBook profile hoat dong on dinh. Moi profile moi phai co fixture va deterministic rules rieng. App tim nguoi can ban khong nam trong roadmap ScanFB.

## Phase 1 - Project skeleton va test harness

Exact scope: Tao Go module `github.com/soleda2026/ScanFB`, CLI toi thieu, package boundaries trong `internal/`, `testdata/README.md` va architecture test bang standard library.

Protected areas: Khong Facebook integration, khong UI day du, khong persistence that, khong SearchProfile implementation, khong rule engine, khong dependency ben thu ba.

Acceptance criteria: `go test ./...`, `go vet ./...`, `go build ./cmd/scanfb` pass; CLI in `ScanFB foundation ready`; architecture test kiem tra package skeleton va domain khong import adapter.

Tests: Architecture test va Go tooling checks trong milestone.

Stop conditions: Can cai dependency ngoai scope, can browser/DB/UI implementation that hoac can mo rong sang rule/domain implementation.

## Phase 2 - Domain models

Exact scope: Implement minimal Go domain types cho normalized post input va cau hinh mot scan: `RawPost`, `AuthorIdentity`, `SearchProfile`, `GeographicMode`, `ScanWindow`, `ScanRequest`, gom built-in MacBook `SearchProfile`.

Protected areas: Khong browser, khong UI, khong database implementation, khong seller mode.

Acceptance criteria: Model co invariant co ban cho timezone, same-day scan window, geographic mode va batch 1-5 group; domain chi import standard library; slices khong leak mutation.

Tests: Unit tests cho valid/invalid value cases va invariant.

Stop conditions: Field identity khong ro.

## Phase 3A - Deterministic author and time eligibility rules

Exact scope: Implement deterministic rule primitives cho `RawPost.CreatedAt` voi `ScanWindow` va author anonymous/no-space display-name exclusion.

Protected areas: Khong buyer intent, khong seller/noise matching, khong geographic classification, khong dedup, khong persistence, khong Facebook adapter, khong UI.

Acceptance criteria: Stable `Decision`, `ReasonCode`, deterministic reason ordering, rules dung `CreatedAt` va supplied `ScanWindow`, khong dung current clock.

Tests: Unit tests cho time boundaries, anonymous labels, no-space author policy va composition ordering.

Stop conditions: Can mo rong sang buyer intent, geographic classification, blocklist hoac dedup.

## Phase 3B - Deterministic buyer-intent and seller/noise rules

Exact scope: Implement deterministic text matching cho product terms, buyer-intent terms va seller/noise exclusion terms tu active `SearchProfile`.

Protected areas: Khong geographic classification, khong dedup, khong persistence, khong Facebook adapter, khong UI, khong seller mode.

Acceptance criteria: Product va buyer-intent term deu phai match, seller/noise term co uu tien exclude, empty/unmatched body fail-closed, matching boundary-aware va deterministic.

Tests: Unit tests cho product matching, buyer intent, seller/noise precedence, empty input, MacBook profile examples va composition voi Phase 3A rules.

Stop conditions: Can fuzzy matching, scoring, LLM, marketplace classifier, geography, blocklist hoac dedup.

## Phase 3C - Deterministic geographic classification

Exact scope: Implement finite MVP geographic classification va `GeographicMode` evaluation bang approved vocabulary trong [SCAN_RULES.md](SCAN_RULES.md): HCM, Vietnam outside HCM, unknown va conflict/review.

Protected areas: Khong foreign classification, khong foreign exclusion, khong geocoder, khong location database, khong district/ward/county recognition, khong dedup, khong persistence, khong Facebook adapter, khong UI.

Acceptance criteria: Domestic vocabulary match deterministic, unknown va conflict vao review, mode HCM/non-HCM/all-Vietnam duoc enforce, composition voi Phase 3A/3B giu stable reason ordering.

Tests: Unit tests synthetic cho approved vocabulary, boundary matching, unknown, conflict, mode evaluation va composition.

Stop conditions: Can vocabulary ngoai approved MVP terms hoac can infer dia ly tu external/location metadata.

## Phase 3 - Deterministic normalization

Exact scope: Normalize text, URL, timestamp, author va keywords.

Protected areas: Khong rule include/exclude phuc tap, khong Facebook DOM parser.

Acceptance criteria: Fixture raw input tao normalized output deterministic.

Tests: Normalization fixtures.

Stop conditions: Timestamp tao bai goc khong xac dinh duoc.

## Phase 4 - Rule engine

Exact scope: BuyerIntentClassifier, target keyword matching cho MacBook Search Profile, author rules, blocklist rules va reason log.

Protected areas: Khong geo classifier day du, khong dedup, khong UI, khong seller mode, khong `LeadIntent` buyer/seller.

Acceptance criteria: Rule decision co reason codes on dinh.

Tests: T01-T07, T23-T24 va SearchProfile checks trong [TESTING.md](TESTING.md).

Stop conditions: Intent ambiguity khong co rule deterministic.

## Phase 4A - Deterministic dedup identity primitives

Exact scope: Implement stable author identity keys, buyer need keys va deterministic duplicate comparison cho same-author same-need posts under active `SearchProfile`.

Protected areas: Khong lead aggregation, khong persistence, khong source-post merging, khong blocklist, khong Facebook adapter, khong UI.

Acceptance criteria: Display name alone khong authorize auto-merge, insufficient stable identity fail closed, different stable authors khong merge chi vi text giong nhau, outcome va reason codes deterministic.

Tests: Synthetic unit tests cho author key priority, need key normalization/evidence, duplicate comparison outcomes, edge cases va dependency boundary.

Stop conditions: Can fuzzy matching, semantic similarity, profile lookup qua network, lead aggregation hoac persistence.

## Phase 4B - In-memory lead aggregation

Exact scope: Implement deterministic in-memory aggregation tu duplicate buyer posts thanh logical lead, preserve full source posts va explicit unaggregated/conflict outputs.

Protected areas: Khong persistence, khong database schema, khong repository interface, khong Facebook adapter, khong blocklist, khong UI, khong scan orchestration.

Acceptance criteria: Same stable author + same deterministic need gom vao mot lead, moi source post duoc giu, duplicate source occurrence khong lap lai, source conflict fail closed, output order deterministic.

Tests: Synthetic unit tests cho basic aggregation, source preservation, separation, duplicate source occurrence, source conflicts, profile behavior va edge cases.

Stop conditions: Can storage, lead workflow status, browser data, concurrency hoac conflict resolution.

## Phase 4C - Local blocklist identity primitives

Exact scope: Implement deterministic in-memory blocklist entry, identity key, list construction va author matching primitives.

Protected areas: Khong persistence, khong database schema, khong repository interface, khong Facebook integration, khong scan orchestration, khong rules/dedup wiring, khong UI, khong CLI.

Acceptance criteria: Supported identity kinds la Facebook user ID, canonical profile URL va username; display name alone khong block; strongest available stable identity wins; exact same-kind matching; duplicate entries xu ly deterministic; outcomes va reason codes stable.

Tests: Synthetic unit tests cho identity normalization, author key priority, list construction, duplicate handling, exact matching, strongest-identity policy, fail-closed cases va architecture boundary.

Stop conditions: Can storage, import/export, profile lookup, fuzzy matching, network identity resolution, scan filtering integration hoac UI/CLI behavior.

## Phase 4D - Apply local blocklist filtering to eligible leads in memory

Exact scope: Implement application-layer in-memory filtering cua already-aggregated leads qua local blocklist primitives.

Protected areas: Khong persistence, SQLite, Facebook integration, browser automation, raw-post scan orchestration, buyer/geographic rule evaluation, UI, CLI, import/export hoac concurrency.

Acceptance criteria: Allowed, blocked va unresolved lead collections explicit; blocked leads khong bi xoa; source posts duoc preserve; strongest-identity blocklist policy duoc reuse; ordering deterministic; input va blocklist khong bi mutate.

Tests: Synthetic unit tests cho basic filtering, identity-kind matching, strongest-identity no-fallback, unresolved leads, empty input/list, ordering, source preservation, defensive copies va architecture boundary.

Stop conditions: Can persistence, repository, scan orchestration, UI/CLI, raw post classification, fuzzy matching, network lookup hoac thay doi dedup/blocklist semantics.

## Phase 5A - Deterministic in-memory evaluation pipeline

Exact scope: Implement application-layer deterministic pipeline cho already-collected `RawPost` values: rule evaluation, eligible selection, in-memory lead aggregation va local blocklist lead filtering.

Protected areas: Khong Facebook/browser integration, khong persistence, khong UI, khong CLI behavior, khong scheduling, khong network, khong concurrency va khong thay doi rules/dedup/blocklist semantics.

Acceptance criteria: Pipeline order ro rang; evaluated, eligible, review, excluded, unaggregated, conflicts, allowed, blocked va unresolved outputs explicit; invalid scan window, inactive/invalid SearchProfile va invalid geographic mode fail closed.

Tests: Synthetic in-memory unit tests cho end-to-end flow, mixed rule results, aggregation, blocklist filtering, ordering, source preservation, defensive copies va invalid configuration.

Stop conditions: Can raw Facebook collection, database/repository, UI/CLI wiring, scheduler, parallel scan hoac heuristic moi.

## Phase 5B - Deterministic in-memory scan batch model

Exact scope: Implement application-layer in-memory manual batch model cho mot den nam explicit Facebook groups, flatten posts deterministic, goi Phase 5A pipeline mot lan va tra batch/per-group count summaries.

Protected areas: Khong Facebook/browser integration, khong persistence, khong UI, khong CLI behavior, khong scheduling, khong retries, khong progress reporting, khong network, khong concurrency va khong thay doi rules/dedup/blocklist semantics.

Acceptance criteria: Zero/qua nam group fail explicit; group identity va post/group consistency duoc validate; post order preserve; full Phase 5A result duoc preserve; batch summary va per-group rule-stage summaries deterministic.

Tests: Synthetic in-memory unit tests cho batch validation, group/post consistency, flattening order, end-to-end summary, per-group summary, source preservation, defensive copies va fail-closed behavior.

Stop conditions: Can Facebook collection, database/repository, UI/CLI wiring, scheduler, retry/progress state, parallel scan hoac heuristic moi.

## Phase 5C - Persistence-facing completed batch contracts

Exact scope: Dinh nghia value contracts nho cho completed scan batch snapshot trong `internal/persistence`: opaque `BatchRecordID`, `BatchRecord`, structural validation, deterministic converter tu `ScanBatchInput`/`ScanBatchResult` va save-only `BatchRepository.SaveBatch`.

Protected areas: Khong persistence implementation, SQLite, schema, migration, JSON/file I/O, network I/O, load/list/update/delete/search/paging API, ID generation, Facebook adapter, UI/CLI, concurrency hoac dependency moi.

Acceptance criteria: Completed batch snapshot preserve scan window, SearchProfile snapshot, GeographicMode, group order, flattened post order, exact decisions/reasons, lead sources, blocklist/application outcomes, unaggregated/conflict outputs va summaries; malformed records fail closed; application/domain/rules/dedup/blocklist khong import persistence.

Tests: Unit tests cho record ID, conversion, source/outcome/reason preservation, summary consistency, invalid records, defensive copies, save-only repository contract va dependency boundary.

Stop conditions: Can storage backend, query/loading behavior, migrations, repository expansion, generated IDs, UI workflow, Facebook collection hoac schema design.

## Phase 5D - Deterministic in-memory batch persistence adapter

Exact scope: Implement `InMemoryBatchRepository` trong `internal/persistence` de satisfy save-only `BatchRepository`, validate completed `BatchRecord`, save defensive snapshots trong memory, preserve insertion order, reject duplicate `BatchRecordID` va expose concrete read-only inspection helpers.

Protected areas: Khong SQLite, SQL, durable storage, schema, migration, JSON/file I/O, network I/O, third-party dependency, repository interface expansion, update/delete/search API, ID generation, UI/CLI, Facebook adapter, concurrency hoac business-rule recomputation.

Acceptance criteria: Zero-value repository usable; valid snapshots save; malformed snapshots fail before storage; duplicate IDs fail without overwrite; order deterministic; reads return defensive snapshots; `BatchRepository` van save-only; durable persistence van deferred.

Tests: Unit tests cho construction/interface, valid saves, five snapshots, ordering, duplicate IDs, validation failures, defensive storage/read copies, source/outcome preservation, empty/not-found reads va deterministic save sequences.

Stop conditions: Can durable backend, schema/query design, generated IDs, repository load/list requirements tren interface, concurrency support, UI workflow hoac Facebook collection.

## Phase 5E - Thin run-and-save orchestration

Exact scope: Add `internal/orchestration.RunAndSaveScanBatch` de chap nhan caller-supplied `BatchRecordID`, chay `application.RunScanBatch`, convert successful result bang `persistence.NewBatchRecord`, save dung mot lan qua `persistence.BatchRepository` va tra completed result/record chi sau khi save thanh cong.

Protected areas: Khong thay application, persistence, domain, rules, dedup, blocklist, Facebook adapter, UI, CLI behavior, durable storage, concurrency, retry, generated ID, repository load/list API hoac business-rule semantics.

Acceptance criteria: Nil repository fail closed; empty ID fail qua persistence identity boundary; scan failure khong save; record conversion/save failure tra zero result; duplicate repository error detect duoc; orchestration chi import application va persistence; core packages khong import orchestration.

Tests: Unit tests voi spy repository va in-memory adapter cho success, save-once, zero-result failure paths, duplicate ID propagation, saved/returned record preservation va architecture boundary.

Stop conditions: Can UI/CLI wiring, Facebook collection, durable storage, retry/progress state, generated ID, repository expansion, concurrency hoac thay doi lower-layer semantics.

## Phase 5F - Durable persistence strategy and SQLite schema design

Exact scope: Create documentation-only SQLite schema design for future local durable persistence of completed `BatchRecord` snapshots, including root aggregate, normalized tables, explicit ordering, reason-code storage, schema version, migration policy, transaction policy, fail-closed reconstruction, indexes and deferred work.

Protected areas: Khong SQLite package, `database/sql`, executable SQL, migration file, database file, runtime I/O, durable adapter, repository expansion, production Go API change, application/orchestration behavior, business rules, UI/CLI, Facebook integration hoac dependency moi.

Acceptance criteria: [PERSISTENCE_SCHEMA.md](PERSISTENCE_SCHEMA.md) maps every `BatchRecord` field to storage, preserves ordered source posts/reasons/summaries, keeps `BatchRecordID` caller-supplied authoritative, defines duplicate-ID fail-closed transaction behavior, future load fail-closed behavior, schema-version and migration policy, and confirms durable save implementation remains deferred.

Tests: Documentation/design verification plus existing Go checks. Future implementation tests will use temporary SQLite databases for constraints, transactions, migrations, duplicate IDs, fail-closed load and deterministic reconstruction.

Stop conditions: Can SQL implementation, database driver, repository load/list API, migration execution, database file, generated ID, runtime I/O, encryption/key management decision hoac product-level deletion/search behavior.

## Phase 5G1 - SQLite schema bootstrap foundation

Exact scope: Add `modernc.org/sqlite` dependency and `SQLiteBatchRepository` foundation inside `internal/persistence` to open/create an explicit local SQLite path, enable/verify foreign keys, transactionally create empty schema version 1, initialize/validate schema metadata, and close.

Protected areas: Khong implement `SaveBatch`, khong make `SQLiteBatchRepository` satisfy `BatchRepository`, khong load/list/update/delete/search/paging API, khong migration execution, khong application/orchestration/domain/rules/dedup/blocklist/Facebook/UI/CLI behavior.

Acceptance criteria: Empty temp DB gets complete schema version 1, existing valid schema reopens without mutation, missing/malformed/unsupported metadata and partial schema fail closed, representative foreign-key/unique/check/not-null constraints work, SQLite driver import stays only in `internal/persistence`.

Tests: Temporary SQLite DB tests for bootstrap, metadata, foreign keys, representative constraints, rollback, deterministic object inventory, invalid paths and architecture dependency boundary.

Stop conditions: Can durable batch saving, repository interface expansion, migrations, production DB path policy, UI/CLI wiring, Facebook collection, generated IDs, concurrency hoac encryption/key management.

## Phase 5G2 - Transactional SQLite SaveBatch

Exact scope: Implement `SQLiteBatchRepository.SaveBatch(record BatchRecord) error` trong `internal/persistence` de validate completed `BatchRecord`, ghi root va toan bo child collections vao SQLite schema version 1 trong mot transaction, preserve explicit ordering, translate duplicate `BatchRecordID` thanh `ErrBatchRecordAlreadyExists`, rollback moi write failure va fail safely sau `Close`.

Protected areas: Khong `LoadBatch`, `ListBatches`, update/delete/search/paging API, migration execution, schema version moi, dependency moi, production DB path, UI/CLI wiring, Facebook behavior, retry, concurrency, generated ID hoac business-rule recomputation.

Acceptance criteria: `SQLiteBatchRepository` satisfy save-only `BatchRepository`; valid one-group va five-group snapshots save; rich snapshot populates all applicable tables; raw posts, decisions, outcomes, evidence, reasons, timestamps va booleans duoc preserve; validation happens before mutation; duplicate ID va child write failure leave database unchanged; no load/list API exists.

Tests: Temporary SQLite DB tests cho interface/basic save, complete table mapping, source preservation, ordering, outcome preservation, validation failures, duplicate ID no-overwrite, transaction rollback, closed repository, immutability, determinism va no deferred load/list/update/delete/search APIs.

## Phase 5G3 - Fail-closed SQLite LoadBatch reconstruction

Exact scope: Implement concrete-only `SQLiteBatchRepository.LoadBatch(id BatchRecordID) (BatchRecord, error)` trong `internal/persistence` de reconstruct mot complete `BatchRecord` tu existing SQLite schema version 1 bang explicit positions, read transaction, schema validation, strict timestamp/boolean decoding, enum validation, final `BatchRecord.Validate` va zero-record failure behavior.

Protected areas: Khong them `LoadBatch` vao `BatchRepository`, khong `ListBatches`, update/delete/search/paging API, migration execution, schema version moi, dependency moi, production DB path, UI/CLI wiring, Facebook behavior, retry, concurrency, generated ID hoac business-rule recomputation.

Acceptance criteria: Saved rich one-group va five-group snapshots load deeply equal; raw posts, decisions, outcomes, evidence, reasons, timestamps, booleans, summaries, unaggregated va conflicts preserve exact order/value; missing ID returns `ErrBatchRecordNotFound`; closed/nil repository fails safely; malformed schema/data fails closed with zero `BatchRecord`; repeated loads deterministic; `BatchRepository` van save-only.

Tests: Temporary SQLite DB tests cho complete round trip, accessor equality, no-write row-count stability, not-found/lifecycle errors, defensive reload determinism, concrete-only API boundary, schema corruption, malformed timestamp/boolean/enum-like values, missing references, duplicate/gapped positions, summary inconsistency, metadata loss va missing required table/index.

Stop conditions: Can list/search/update/delete, interface expansion, migrations, production storage path policy, UI/CLI/Facebook wiring, generated IDs, concurrency, retries hoac schema version 2.

## Phase 5 - Geographic classifier hardening

Exact scope: Harden geographic classifier behavior sau Phase 3C neu can, van theo [SCAN_RULES.md](SCAN_RULES.md) va khong mo rong vocabulary khi chua co milestone rieng.

Protected areas: Khong browser, khong dedup.

Acceptance criteria: Mode geo mac dinh va override dung theo [SCAN_RULES.md](SCAN_RULES.md).

Tests: T14-T18.

Stop conditions: Dia danh khong du ngu canh de phan loai.

## Phase 6 - Deduplication va lead aggregation

Exact scope: Fingerprint buyer nhu cau theo SearchProfile, merge source, tach nhu cau khac nhau.

Protected areas: Khong persistence implementation, khong Facebook adapter, khong seller lead.

Acceptance criteria: Mot lead co nhieu source khi dung, khong gop sai chi vi ten.

Tests: T08, T09, T21, T22.

Stop conditions: Identity nguoi dang qua yeu de quyet dinh deterministic.

## Phase 7 - SQLite persistence

Exact scope: Local persistence cho group, raw post, decision, lead, source, blocklist va settings.

Protected areas: Khong cloud backend, khong sync, khong account system.

Acceptance criteria: CRUD va migration co test.

Tests: Repository tests voi database local tam.

Stop conditions: Yeu cau dong bo cloud hoac multi-user.

## Phase 8 - UI dung fixtures, chua ket noi Facebook

Exact scope: UI lead tabs, batch summary, settings va Dry Run review bang fixture.

Protected areas: Khong browser integration.

Acceptance criteria: UI hien thi data fixture day du va thao tac status/blocklist local.

Tests: UI fixture tests va manual validation.

Stop conditions: Yeu cau scan Facebook that.

## Phase 8A - Native macOS UI architecture decision and app-shell plan

Status: complete after Phase 8A acceptance checks pass.

Exact scope: Documentation-only decision that SwiftUI is the native macOS presentation shell, future app root is `macos/ScanFBApp/`, Go core remains authoritative, bridge selection is deferred, and Phase 8 is split into narrow future milestones.

Protected areas: Khong tao `macos/`, Xcode project, Swift package, Swift source, Go bridge, IPC, HTTP, local database path, UI fixtures, production code, dependency, Facebook integration hoac seller mode.

Acceptance criteria: [MACOS_UI_ARCHITECTURE.md](MACOS_UI_ARCHITECTURE.md) documents SwiftUI, layer ownership, prohibited duplication, bridge candidates/deferred decision, fixture policy, privacy policy, testing/manual validation policy va Phase 8B-8I plan; README/architecture docs reflect the approved SwiftUI direction.

Tests: Documentation consistency checks plus existing Go verification: `go test ./...`, `go vet ./...`, CLI build/run output.

Stop conditions: Can tao app project, Swift code, bridge, database path, UI fixture implementation, Facebook integration, dependency hoac production scan workflow.

## Phase 8B - Empty native SwiftUI app shell only

Status: complete after Phase 8B acceptance checks pass. Phase 8C is next.

Exact scope: Create minimal native SwiftUI macOS app shell at `macos/ScanFBApp/` with app name ScanFB, one main window and placeholder navigation.

Protected areas: Khong Go bridge, database, Facebook, networking, third-party dependency, production settings, persistence, generated fixture business data, seller mode hoac business-rule logic in Swift.

Acceptance criteria: App builds and launches locally; window title/app identity are ScanFB; placeholder navigation is visible; six expected sections exist in stable order; no Go code or bridge is touched.

Tests: Xcode build, manual app launch after killing stale process, verify rebuilt app bundle is running, plus Go `go test ./...`, `go vet ./...` and CLI output unchanged.

Stop conditions: Need bridge, fixture dashboard, real data, persistence, signing/notarization, Facebook controls hoac deployment target decision beyond app-shell minimum.

## Phase 8C - Static fixture dashboard and batch summary

Status: complete after Phase 8C acceptance checks pass. Phase 8D is next.

Exact scope: Add deterministic sample/demo overview screen and batch summary using static fixture values only.

Protected areas: Khong Go bridge, SQLite access, Facebook data, generated business data, networking, persistence, status mutation hoac business-rule recomputation in Swift.

Acceptance criteria: Overview clearly labeled sample/demo, Vietnamese text and diacritics preserved, summary values are fixture-only and do not claim live Facebook data; other five sections remain placeholders.

Tests: Xcode build, fixture rendering/manual launch, stale-process kill before relaunch, Go regression checks.

Stop conditions: Need real batch data, bridge, database, scan button behavior hoac non-fixture summaries.

## Phase 8D - Fixture lead tabs and lead cards

Status: complete after Phase 8D acceptance checks pass. Phase 8E is next.

Exact scope: Add deterministic fixture lead tabs and lead cards for buyer-only lead presentation.

Protected areas: Khong seller tab, seller mode, Go bridge, SQLite direct access, Facebook data, status persistence, scoring recomputation hoac reason-code generation in Swift.

Acceptance criteria: Lead tabs and cards present exactly four sample buyer leads, three fixed tabs, source counts, repository-existing reason-code display strings and disabled/placeholder actions without claiming live data. Dry Run, Nhóm, Blocklist and Cài đặt remain placeholders.

Tests: Swift fixture tests, Xcode build, manual fixture validation, accessibility spot checks where practical, stale-process kill before relaunch, Go regression checks.

Stop conditions: Need real lead loading, bridge, status workflow, persistence hoac Facebook adapter.

## Phase 8E - Fixture Dry Run review

Status: complete after Phase 8E acceptance checks pass. Phase 8F is next.

Exact scope: Add deterministic fixture Dry Run review screen for included/review/excluded sample posts.

Protected areas: Khong rule recomputation, reason-code inference, restore behavior backed by persistence, Facebook data, bridge, network hoac seller review mode.

Acceptance criteria: Dry Run is presented with exactly ten sample posts, three fixed tabs, repository-existing reason-code display strings, clear sample/demo labels and disabled/placeholder action without claiming live data. Nhóm, Blocklist and Cài đặt remain placeholders.

Tests: Swift fixture tests, Xcode build, manual fixture validation, stale-process kill before relaunch, Go regression checks.

Stop conditions: Need real filter decisions, persistence, bridge, user edits hoac production review workflow.

## Phase 8F - Fixture settings and blocklist presentation

Status: complete after Phase 8F acceptance checks pass. Phase 8G is next.

Exact scope: Add deterministic fixture settings and blocklist screens.

Protected areas: Khong persistence, SQLite direct access, real blocklist import/export, Facebook identity lookup, network, bridge, credentials, cookies hoac production settings writes.

Acceptance criteria: Settings and blocklist are sample/demo only, local-first/privacy copy is visible where appropriate, display name is not represented as authoritative block identity, blocklist actions are disabled, settings are read-only, and SwiftUI does not own blocklist semantics.

Tests: Swift fixture tests, Xcode build/test, manual fixture validation after optional launch in a separate milestone, stale-process kill before relaunch when launching is in scope, Go regression checks when requested.

Stop conditions: Need real settings storage, blocklist mutation, database path, bridge hoac identity resolution.

## Phase 8G - Fixture UI interaction state

Status: complete after Phase 8G acceptance checks pass. Phase 8H is next.

Exact scope: Add in-memory fixture-only UI state for viewed/ignored interactions.

Protected areas: Khong persistence, database, bridge, production lead status, Facebook action, network, business logic recomputation hoac seller workflow.

Acceptance criteria: Interactions affect only the current in-memory fixture session and reset on relaunch; UI does not imply production persistence; existing Leads eligibility tabs, categories, reason codes and fixture order remain unchanged.

Tests: Swift interaction-state tests, Xcode build/test, manual interaction validation after stale-process kill/relaunch when launch is in scope, Go regression checks when requested.

Stop conditions: Need status persistence, Go integration, SQLite, import/export hoac real lead workflow.

## Phase 8H - Lead interaction browser handoff

Status: complete after Phase 8H acceptance checks pass. Phase 8I.1 is next.

Exact scope: Replace the old stateful Leads action with stateless `Tương tác` browser handoff using deterministic fixture source URLs and native SwiftUI `openURL`.

Protected areas: Khong production bridge, persistence, database, WebKit, embedded browser, browser automation, Facebook SDK/API, networking client, credential/cookie/browser-session access, shell `open`, subprocess, third-party package, business logic recomputation hoac seller workflow.

Acceptance criteria: Session states are exactly `new/viewed/ignored`; `Tương tác` is an action, not a state; pressing `Tương tác` validates an absolute HTTPS fixture source URL and hands it to the macOS default browser without mutating lead state; browser owns Facebook login/session/cookies; eligibility tabs, categories, reasons and order remain unchanged.

Tests: Swift interaction-state and URL handoff tests, Xcode build/test, bundle existence check and audits for forbidden APIs/dependencies. No app launch or visual inspection in this milestone.

Stop conditions: Need real lead loading, Go integration, production source URLs, browser automation, Facebook action automation, persistence, bridge, database path policy hoac live Facebook data.

## Phase 8I.1 - Bridge evaluation and decision only

Status: complete after Phase 8I.1 acceptance checks pass. Phase 8I.2 is next and remains incomplete.

Exact scope: Documentation-only evaluation of realistic SwiftUI-Go bridge options and selection of exactly one future bridge model.

Protected areas: Khong implement bridge, khong Go/Swift source changes, khong Xcode project change, khong dependency, khong subprocess runtime, khong socket runtime, khong HTTP server, khong C binding, khong generated code, khong build script, khong shell build phase, khong database access, khong Facebook integration, khong arbitrary JSON command API.

Acceptance criteria: [BRIDGE_DECISION.md](BRIDGE_DECISION.md) compares local subprocess, local IPC and in-process Go library binding against deterministic request/response, explicit schemas, error propagation, cancellation, crash isolation, no hidden network, no credential/browser transfer, ownership boundaries, build/Xcode complexity, packaging, Apple Silicon, testability, debugging, overhead, versioning, observability, fail-closed behavior, notarization/signing and narrow-slice fit; selects exactly one bridge.

Tests: Documentation verification only: `git diff --check`, `git diff --stat`, `git status --short` and audits that no source/runtime/dependency changes were made.

Stop conditions: Need implementation, broad API, live Facebook data, database path policy, packaging/signing/notarization implementation hoac bridge runtime beyond decision scope.

## Phase 8I.2 - First read-only bridge slice

Status: complete after Phase 8I.2 acceptance checks pass. Phase 8I.3+ remains incomplete.

Exact scope: Implement one local subprocess request/response slice where Swift requests a deterministic Go core readiness value and Go returns a minimal typed response.

Protected areas: Khong broad command bus, lead list/search API, persistence mutation, direct Swift SQLite access, hidden networking, Facebook automation, seller mode, migration execution, credential/cookie/browser-session transfer hoac business-rule duplication.

Acceptance criteria: One typed versioned request and minimal transport-only response concept no broader than schema version, readiness status and core identity; bounded subprocess execution; explicit error propagation; explicit cancellation/timeout policy; no database mutation; no Facebook; no SearchProfile, lead data, persistence information, business-rule summary, capability inventory or product/domain data; tests cover Go adapter/schema, Swift decoding/error mapping and one narrow subprocess integration path.

Tests: Bridge-specific Go unit/schema tests, Swift decoding/error-mapping tests, one subprocess integration test, Xcode test/build, Go `go test ./...`, Go `go vet ./...`, CLI output unchanged unless separately approved. Phase 8I.2 explicitly does not launch the app for visual inspection.

Stop conditions: Need broad production workflow, lead list/search, Facebook adapter, production database path policy, mutating operation, package/signing/notarization expansion hoac unsupported bridge behavior.

## Phase 8I.2a - Debug helper packaging only

Status: complete after Phase 8I.2a acceptance checks pass. Phase 8I.3+ remains incomplete.

Exact scope: Package the existing `scanfb-bridge-helper` into Debug `ScanFB.app` bundles at `Contents/Helpers/scanfb-bridge-helper` so the Phase 8I.2 readiness bridge can run from the built app.

Protected areas: Khong thay bridge protocol, request/response schema, operation set, Facebook, persistence, database, networking, business data, UI behavior ngoai readiness settings, Release distribution policy, signing/notarization policy, dependency hoac repo-local build artifact.

Acceptance criteria: Debug Xcode build compiles the helper for Apple Silicon/macOS into DerivedData, copies exactly one executable helper into the app bundle, preserves executable permission, Swift resolves only the deterministic bundle-relative helper path, missing helper remains fail-closed, and direct helper execution returns the exact readiness response without stderr contaminating stdout.

Tests: Go `go test ./...`, Go `go vet ./...`, Xcode test/build with fresh DerivedData, bundled helper path/executable checks, direct bundled-helper readiness invocation, `git diff --check`, diff/status and audits for no protocol/schema/runtime scope expansion.

Stop conditions: Need machine-specific Go path, broad Release packaging/signing/notarization, hidden network/listener, shell fallback, PATH search, production database path, Facebook integration hoac unsupported helper location.

## Phase 8I.3+ - Later narrow Go integration slices

Exact scope: Add later bridge slices only after Phase 8I.2 proves build integration, request serialization, response parsing, error propagation, cancellation policy, transport test parity and Go authority.

Protected areas: Khong broad bridge API, arbitrary JSON maps, direct Swift SQLite access, hidden networking, Facebook automation, seller mode, migration execution, search/list expansion without separate milestone hoac business-rule duplication.

Acceptance criteria: Each slice has explicit schema, deterministic fixture-backed behavior, bounded payloads, error propagation, cancellation policy, tests on both sides as needed, and no regression to Go core checks.

Tests: Slice-specific bridge tests, Xcode build/manual launch when UI changes, stale-process kill before relaunch when launching is in scope, Go `go test ./...`, Go `go vet ./...`, CLI output unchanged unless separately approved.

Stop conditions: Need broad production workflow, Facebook adapter, database path policy, packaging/signing/notarization hoac unsupported bridge behavior.

## Phase 9A - Group attempt and batch lifecycle state machine

Status: complete after Phase 9A acceptance checks pass.

Exact scope: Implement Go-only in-memory application-layer lifecycle state machine cho mot production-shaped batch dung 5 group, caller-supplied batch/attempt IDs, deterministic order, one-running-at-a-time va day-boundary expiration theo `Asia/Ho_Chi_Minh`.

Protected areas: Khong Facebook, SwiftUI, bridge operation, persistence, SQLite schema/table, production scan run, scheduler, retry, concurrency, networking, browser automation hoac Phase 5B merge.

Acceptance criteria: Batch requires exactly five distinct non-empty group IDs; attempts begin `pending`; valid transitions are `pending -> running`, `running -> succeeded`, `running -> failed`, `pending -> skipped`, `pending/running -> expired_at_day_boundary`; invalid transitions fail closed without partial mutation; T19 preserves failed group state; T20 expires unfinished attempts at local day boundary.

Tests: Focused `internal/application` lifecycle tests cover exact five-group invariant, ordering, supplied attempt IDs, sequential execution, all valid/invalid transitions, summary reconciliation, defensive copies, deterministic supplied-time day-boundary behavior, T19 and T20.

Stop conditions: Can Facebook adapter behavior, production group storage, persistence/schema changes, UI/bridge wiring, scheduler/retry/concurrency hoac Phase 5B integration.

## Phase 9B - WatchedGroup management foundation

Status: complete after Phase 9B acceptance checks pass.

Exact scope: Implement Go-only deterministic in-memory `WatchedGroup` value model va collection management cho caller-supplied identity/time, metadata, active/inactive lifecycle, lookup va stable insertion-order inspection voi tong so group khong gioi han.

Protected areas: Khong five-group selection, queue ordering/cursor/rotation, Phase 9A lifecycle invocation, Facebook, persistence, SQLite, SwiftUI, bridge, scheduler, retry, concurrency, networking hoac generated ID.

Acceptance criteria: Invalid identity/time fail closed; duplicate local/authoritative identity khong overwrite; metadata va active state update preserve identity/createdAt; returned collection la defensive snapshot theo insertion order; `displayOrder` chi la presentation metadata.

Tests: Focused domain/application tests cho identity, validation, duplicates, unlimited count, lookup, metadata, active/inactive, chronology, deterministic order, defensive copies va absence cua deferred behavior.

Stop conditions: Can persistence/schema, queue policy, UI/bridge, Facebook adapter hoac production scan orchestration.

## Phase 9C - Deterministic five-group round-robin selection policy

Status: complete after Phase 9C acceptance checks pass.

Exact scope: Implement Go-only pure in-memory policy de chon dung 5 active WatchedGroups bang circular insertion-order traversal tu explicit caller-managed collection-position cursor va tra cursor ngay sau group thu nam.

Protected areas: Khong scan execution, Phase 9A lifecycle construction, generated ID, cursor/group persistence, Facebook, SQLite, SwiftUI, bridge, scheduler, retry, concurrency hoac networking.

Acceptance criteria: Chi active groups eligible; inactive positions van nam trong cursor geometry; traversal wrap-around theo stable insertion order; insufficient active groups fail closed khong partial result; `displayOrder`, `createdAt` va `lastSuccessfulScanAt` khong anh huong ordering; repeated same input/cursor cho cung result.

Tests: Focused application tests cho initial/bounded cursor, exact-five, continuation, wrap-around, inactive skip/geometry, insufficient/all-inactive/empty cases, duplicate prevention, metadata neutrality, reactivation, no mutation, defensive copies va absence cua execution/infrastructure behavior.

Stop conditions: Can persistence, lifecycle mapping, UI/bridge, Facebook adapter hoac production scan execution.

## Phase 9D - Map FiveGroupSelection to ScanBatchLifecycle

Status: complete after Phase 9D acceptance checks pass.

Exact scope: Implement one Go-only deterministic application-layer mapper tu mot already-created exact-five `FiveGroupSelection` sang caller-supplied Phase 9A lifecycle inputs. Caller cung cap batch ID, `ScanWindow` va dung 5 attempt ID theo selection order.

Protected areas: Khong re-select group, khong advance/persist cursor, khong start lifecycle transition, khong goi `RunScanBatch`, khong generated ID, Facebook, persistence, SQLite, SwiftUI, bridge, scheduler, retry, concurrency hoac networking.

Acceptance criteria: Mapper preserve exact selected-group va attempt-ID order; malformed selection fail closed; Phase 9A batch/window/attempt validation errors propagate; result gom dung 5 `pending` attempts; input, selection va cursor khong mutate.

Tests: Focused application tests cho valid mapping, order, pending state, exact attempt-ID count, malformed selection/identity, Phase 9A error propagation, deterministic repetition, defensive copies va absence cua execution/infrastructure behavior.

Stop conditions: Can re-selection, generated ID, cursor ownership, lifecycle execution, Facebook, persistence hoac UI/bridge wiring.

## Phase 9E1 - Minimal macOS Watched Groups UI

Status: complete. Automated verification and user-guided manual UI verification passed.

Exact scope: Replace the `Nhóm` placeholder with a session-only SwiftUI screen that lists watched groups, adds a group from display name plus canonical HTTPS URL, toggles active state and displays the exact next five returned by Go Phase 9C. A bounded typed local-subprocess bridge reconstructs Phase 9B state from the caller snapshot on each request.

Protected areas: Khong persistence/cursor storage, SQLite, Facebook/Safari, lifecycle construction, scan execution, Phase 11, scheduler, retry, concurrency, networking, independent Swift identity/ordering/selection logic hoac broad bridge command bus.

Acceptance criteria: Empty and fewer-than-five states are clear; group rows show name and active control; add and toggle outcomes come from authoritative Go validation; exact-five and larger active sets display the bridge-returned 1-5 order; cursor advances only through an explicit UI action; state resets when the app restarts.

Tests: Focused Go bridge tests, focused Swift store/schema tests, full Go regression/vet, full macOS tests/build and CLI smoke check. Closeout automation does not relaunch the already manually verified app.

Manual verification: The freshly built Debug app launched and opened the Groups screen; empty/add/active-toggle flows passed; fewer than five active groups produced no partial selection; exactly five produced the exact application order; and six active groups advanced from `1,2,3,4,5` to `6,1,2,3,4` after one explicit `Chuyển lượt chọn` action. No scan or browser/Safari action occurred. Group identities used for verification are intentionally not recorded. Session-only state remains intentional.

Non-blocking future polish: The screen currently mixes Vietnamese labels with `Watched Groups` and `Next 5 Groups`; a later UI polish slice may make the language consistent.

Stop conditions: Can persistence, production scan, Facebook acquisition, lifecycle execution, scheduler, retry, concurrency hoac broader product data bridge.

## Phase 9E2a - Production persistence location and schema-evolution decision

Status: complete after Phase 9E2a documentation acceptance checks pass. Outcome: APPROVE.

Exact scope: Docs-only evaluation and selection of the production local-storage location, schema-version strategy, group/cursor transaction boundary, failure behavior and bridge path ownership needed to unblock Phase 9E2.

Selected architecture: Two Go-owned SQLite databases under `<user-application-support>/com.soleda.ScanFB/`. `completed-batches.sqlite3` retains its current schema v1; future `watched-groups.sqlite3` starts an independent schema v1 for ordered full Phase 9B WatchedGroups and one exact Phase 9C cursor. The helper resolves production paths internally; Swift never sees them; tests inject explicit temporary paths.

Protected areas: Documentation only. Khong Go/Swift/Xcode/runtime/schema SQL/table/database/migration/bridge behavior, Phase 11, Facebook/Safari, scheduler, retry, worker, networking, `UserDefaults` hoac `@AppStorage`.

Acceptance criteria: Candidate comparison and one combined decision are recorded in [WATCHED_GROUP_PERSISTENCE_DECISION.md](WATCHED_GROUP_PERSISTENCE_DECISION.md); path is standard macOS Application Support without machine-specific values; completed-batch v1 is unchanged; cursor corruption fails closed without modulo repair; no source or schema changes occur.

Tests: Markdown consistency/audit, `git diff --check`, full Go regression/vet and unchanged CLI smoke check. No Xcode build is required.

Stop conditions: Implementation request, unresolved bundle identity, need to migrate completed-batch schema, raw path exposure to Swift, cross-database transaction invariant or broader redesign.

## Phase 9E2 - Implement dedicated WatchedGroup-state SQLite persistence

Status: complete. Implementation, automated verification and Phase 9E3b user-guided quit/relaunch verification passed. Persisted group membership and active/inactive state are both manually verified across process relaunch.

Exact scope: Implement the approved Phase 9E2a decision: Go production path resolver, dedicated state schema v1/repository, transactional full WatchedGroup plus cursor restore/mutation, persistent watched-group bridge behavior and Swift authoritative-state refresh/error presentation.

Protected areas: Khong completed-batch schema change/migration, scan result/lifecycle persistence, Phase 11 execution, Facebook/Safari, scheduler, retry, background worker, networking, cloud/sync, Swift persistence hoac generic CRUD/SQL bridge.

Acceptance criteria: `watched-groups.sqlite3` uses independent schema v1 and DELETE journal; Go restores insertion order/full metadata/exact cursor and owns all mutations; corrupt state fails closed; bridge requests carry no collection/cursor/path authority; Swift distinguishes loading, successful empty state and storage failure. Existing completed-batch schema v1 remains unchanged.

Tests: Temporary-database Go persistence/bridge tests, focused Swift store tests, full regressions and build passed. Automated reopen tests verify persisted groups, active state, insertion order and exact cursor. Phase 9E3b manual acceptance verified group persistence after one full quit/relaunch, then inactive-state persistence after a second full quit/relaunch through the supported enrollment UI.

Stop conditions: Need completed-batch migration, arbitrary filesystem path bridge input, Swift-owned persistence, Phase 11/Facebook behavior or broader storage architecture.

## Phase 9E2b - Group discovery product-contract correction

Status: complete. Implementation, automated verification and user-guided manual UI verification passed.

Historical note: Phase 9E3b supersedes this milestone's assumption that automatic joined-group discovery is the primary product source. The verification below remains an accurate record of the Phase 9E2b UI at closeout.

Exact scope: Preserve Phase 9E2 source-neutral persistence while correcting the primary UI/product contract: future joined-group discovery is the normal group source, active toggles remain user-controlled, next-five is an informational preview, and cursor progression is internal rather than a separate user action.

Protected areas: Khong real discovery, Facebook/Safari acquisition, selector, schema expansion, account/session/browser persistence, scan execution, Phase 11/12, automatic cursor advance, dependency hoac Xcode configuration change.

Acceptance criteria: Primary Groups UI hides manual add and queue advance; disabled discovery copy is explicit; preview rendering sends list only; persistence/bridge remain authoritative and unchanged; docs define manual add as fallback/scaffolding and cursor as future batch-progression state.

Tests: Focused Swift presentation/store checks, unchanged Phase 9E2 persistence/bridge tests, full regressions/build and source audits passed. This closeout does not relaunch the already manually verified app.

Manual verification: The freshly built Groups screen opened successfully; `+ Thêm nhóm` and `Chuyển lượt chọn` were absent; disabled `Đồng bộ nhóm đã tham gia` was visible and unavailable; the empty state stated that discovery is not implemented; and `Next 5 Groups` remained a visible read-only preview. The flow did not imply manual population or cursor advancement and triggered no discovery, Facebook, Safari or browser behavior.

Persistence closeout: Automated repository, bridge and Swift tests verify reopen behavior. Phase 9E3b user-guided verification now satisfies the previously deferred manual quit/relaunch check through the supported enrollment flow; the earlier deferment was not evidence of a persistence failure.

Stop conditions: Requires real Facebook discovery, persistence provenance/schema redesign, scan execution or removal of useful fallback bridge/domain behavior.

## Phase 9E3a - Joined-groups page rendered-DOM reconnaissance

Status: complete with STOP/INCONCLUSIVE result after exactly one user-guided read-only acquisition.

Exact scope: Docs-only inspection of one user-prepared Facebook page visibly listing joined groups through the committed `AcquireSafariActiveTabRenderedDOM()` path. Raw DOM stayed in memory; only bounded counts, confidence and generic marker categories were retained.

Result: 3,208,287 rendered-DOM bytes; one tentative containing section; zero semantic group-item containers; one canonical group-link/name association; zero directly numeric IDs; zero explicit machine-readable joined-membership candidates; one ambiguous group link; and one traversal candidate. Identity and name association are STRONG, section and ordering are TENTATIVE, item and joined-versus-recommended evidence are NOT_FOUND.

Decision: Phase 9E3 remains blocked. A broad `/groups/` link scan cannot distinguish joined groups from recommendations, navigation or unrelated links. See [FACEBOOK_JOINED_GROUPS_RECONNAISSANCE.md](FACEBOOK_JOINED_GROUPS_RECONNAISSANCE.md).

Protected areas: No production discovery/parser/selector, persistence/synchronization, active-state change, cursor advancement, `RawPost`, scan, Phase 11/12, bridge/UI, dependency, browser mutation/navigation/scroll/click, retry, network/listener, Accessibility, WebKit, extension or browser private-state access.

Tests: Documentation consistency plus unchanged Go regression/vet/CLI checks. The live run used exactly one acquisition; no private value or raw DOM entered the repository.

Stop condition reached: stable item boundaries and joined-versus-recommended evidence are absent. Any new attempt requires a separately approved evidence/acquisition milestone; do not weaken markers.

## Phase 9E3b - Approved-group enrollment product correction

Status: complete. Implementation, automated verification and user-guided manual acceptance passed.

Exact scope: Restore the existing Add sheet as a one-time setup action for user-approved Facebook groups. Persist each group through the unchanged Go-owned WatchedGroup Add path, keep active/inactive eligibility and read-only next-five presentation, remove the disabled automatic-sync affordance and keep cursor progression internal.

Product contract: Automatic enumeration of every joined Facebook group is not required for MVP. Enrollment records a group the user wants scanned but does not prove current membership, access permission, Safari login validity or future post availability. A future scan must report access failure as an explicit group attempt/result rather than silently removing the enrolled group.

Protected areas: No schema/protocol redesign, discovery/acquisition, membership verification, Safari/Facebook runtime, selector, Swift persistence, cursor advancement, scan execution, Phase 11/12, networking, background work, scheduler or retry.

Acceptance criteria: Enrollment action and one-time local-storage copy are visible; blocked sync affordance and manual queue advancement are absent; successful Add consumes the authoritative backend response; persisted state loads on startup; toggles remain authoritative; next-five stays read-only and does not advance cursor; persistence errors remain explicit.

Tests: Focused Swift store/UI checks, existing bridge/persistence tests, full macOS and Go regressions, vet, CLI smoke check and Debug build passed. User-guided manual acceptance launched the fresh bundle, opened Groups, verified enrollment/empty/read-only-preview UI and absence of sync/manual-advance controls, enrolled one real group, verified it remained after full quit/relaunch, changed it inactive and verified both the group and OFF state remained after another full quit/relaunch. No Facebook Scan, Safari acquisition, discovery, browser mutation or cursor advancement occurred; private group identity is intentionally not recorded.

## Phase 9E3 - Joined-group discovery acquisition contract

Status: optional future research, currently blocked by the Phase 9E3a STOP/INCONCLUSIVE result; not required for MVP.

Exact scope: Define and validate one bounded, user-triggered acquisition contract for discovering groups joined by the Facebook account already authenticated in Safari. No persistence synchronization or scan execution.

Protected areas: Khong selectors for posts, RawPost extraction, credential/cookie/session/profile persistence, group synchronization, scan lifecycle, retry/polling, background worker hoac network service.

Acceptance criteria: Typed discovered-group identity/evidence boundary, explicit user action, bounded/redacted failures and no fabricated groups. Exact acquisition technique requires separate evidence and approval.

Tests: Synthetic contract fixtures plus separately approved user-guided validation; no private group identity enters repository fixtures or docs.

Stop conditions: Stable joined-group evidence is unavailable, permission attribution is unclear, or implementation requires private browser stores/account identity persistence.

## Phase 9E4 - Discovered-group synchronization

Status: optional future research only if a separately approved Phase 9E3 succeeds; not an MVP blocker.

Exact scope: Reconcile one validated discovered-group snapshot into the existing source-neutral WatchedGroup repository while preserving user-controlled active state and deterministic insertion/cursor semantics.

Protected areas: Khong scan execution, post selectors, provenance/schema expansion without a separate decision, silent identity merge, automatic activation policy invention, scheduler, retry or background synchronization.

Acceptance criteria: Existing identities update deterministically, new groups are added without duplicates, active choices remain stable, missing discovery evidence fails closed, and synchronization does not advance cursor.

Tests: Temporary-repository fixtures for add/update/conflict/repeated synchronization, active-state preservation, cursor immutability and atomic failure.

Stop conditions: Requires account/session storage, ambiguous identity merging, schema migration or scan execution.

## Phase 9E+ - Later adapter and orchestration slices

Exact scope: The shortest MVP path proceeds from the Phase 9E3b enrolled-group set to one known enrolled-group scan, then to an explicit user-triggered next-five batch scan. Future narrow milestones may expose lifecycle presentation through an approved bridge; automatic internal cursor progression begins only when real batch execution semantics are ready.

Protected areas: Khong gop Facebook adapter, persistence hoac production execution vao Phase 9E1; moi slice phai co scope, tests va boundaries rieng.

Acceptance criteria: Defined by each future milestone.

Tests: Future UI, bridge, adapter and orchestration tests by slice.

Stop conditions: Can broad production workflow trong mot milestone.

## Phase 10A - Prepared-page extraction contract

Status: complete after Phase 10A acceptance checks pass.

Exact scope: Implement Go-only typed local prepared-page fixture contract va deterministic fail-closed extraction sang ordered `RawPost` values cho mot caller-supplied watched group.

Protected areas: Khong live browser/DOM, selector production, network, cookie, credential, session, WebKit, automation, scan/lifecycle execution, persistence, SwiftUI hoac bridge.

Acceptance criteria: Snapshot version/group/capture va post body/time/URL/group consistency duoc validate; exact fixture order va supplied values duoc preserve; unavailable optional fields khong bi fabricate; malformed input tra zero posts va explicit error.

Tests: Focused fixture tests cho mapping, order, Vietnamese text, author identity, timestamp, URL, group consistency, determinism, immutability va absence cua deferred infrastructure.

Stop conditions: Can live DOM detail, relative-time/browser-locale inference, third-party parser, browser/session access hoac Phase 11 execution.

## Phase 10B1 - Safari user-prepared active-tab acquisition probe

Status: complete, including successful manual live Safari validation on one user-prepared Facebook group page. Acquired source was approximately 1.5-1.6 MB, below the 4 MiB decoded-content bound.

Exact scope: Implement mot Safari-only, user-triggered boundary doc URL, optional title va bounded page source cua dung current tab trong front Safari window. Nguoi dung tu mo Safari, dang nhap, dieu huong va de tab mong muon active; timestamp do caller cung cap.

Protected areas: Khong production selector, `RawPost` mapping, Phase 10A automatic extraction, Phase 11 execution, auto-login/navigation/tab switching/scrolling/clicking/polling, batch automation, Accessibility/UI scripting, WebKit/extension, network/listener, cookie/credential/session/Keychain/profile/history/cache access, SwiftUI, bridge hoac persistence.

Acceptance criteria: Direct `/usr/bin/osascript` JXA invocation chi doc active tab; output chi gom URL/title/bounded source/caller timestamp; absolute HTTPS URL va 4 MiB decoded-content bound fail closed; stdout/stderr tach rieng; timeout, cancellation, TCC permission, process, tab va malformed/content errors explicit.

Tests: Pure parser va injected runner tests cover exact preservation, bounds, URL validation, deterministic parsing, direct executable/arguments, stdout/stderr separation, timeout/cancellation, permission/process/tab errors va deferred-boundary audit. User-guided live validation da xac nhan exact active-tab acquisition ma khong print raw HTML.

Stop conditions: Can Accessibility/UI scripting, extension, WebKit, browser profile/session storage, hidden networking, production DOM selector hoac Phase 11 execution.

## Phase 10B2a - One-page live Facebook DOM reconnaissance

Status: blocked/inconclusive after exactly one read-only acquisition. The expected active group page was present, but Safari `tab.source()` did not represent the visible feed sufficiently for fail-closed post selectors.

Exact scope: Docs-only redacted inspection of one user-prepared Facebook group page source for semantic post container, permalink/ID, body, author, absolute timestamp, group identity and visible ordering evidence.

Protected areas: Khong production selector/parser, `RawPost`, Phase 10A extraction, Phase 11, repository fixture/content, browser mutation, auto login/navigation/scroll/click/refresh/poll/retry, cookie/session/profile access, networking, SwiftUI, bridge hoac persistence.

Acceptance result: Active-tab HTTPS URL provided strong page-level group identity, but post container, permalink/ID, body, author, timestamp and post ordering evidence were not found. Generic bootstrap symbols were rejected as unstable/unattributed. See [FACEBOOK_SAFARI_DOM_RECONNAISSANCE.md](FACEBOOK_SAFARI_DOM_RECONNAISSANCE.md).

Tests: Documentation consistency plus existing Go regression, vet and CLI checks. Live source remained temporary outside the repository and no private values were documented.

Stop condition reached: visible content is primarily client-rendered and absent from `tab.source()`; reliable analysis would require a separately approved acquisition-decision milestone.

## Phase 10B2b - Production selector implementation

Status: incomplete and still blocked. Phase 10B2c approves only a future rendered-DOM acquisition probe; selectors require successful acquisition plus separate redacted evidence.

Exact scope: Do not begin until a separate milestone approves and validates a bounded read-only acquisition representation that actually contains stable post-level DOM evidence.

Protected areas: Khong auto login, credential/cookie/session storage, broad browser automation, batch automation, Phase 11 execution hoac selector implementation based on obfuscated/presentation-only markers.

Acceptance criteria: Not defined by Phase 10B2a; current `tab.source()` evidence is insufficient.

Tests: Deferred with implementation.

Stop conditions: Acquisition lacks stable post container, body, author, absolute timestamp or permalink/ID; requires forbidden browser access; DOM remains unknown.

## Phase 10B2c - Rendered-DOM acquisition decision

Status: complete after documentation verification.

Exact scope: Compare realistic Safari rendered-DOM mechanisms against one-shot active-tab, permission, privacy, bounds, packaging, sandbox and testability requirements. Select one future acquisition mechanism without implementing it.

Decision: APPROVE only fixed, read-only page-side JavaScript executed through Safari Apple Events against current tab of the front window. User-trigger, exact HTTPS URL validation, finite output, direct owned `/usr/bin/osascript`, explicit timeout/cancellation and fail-closed Safari developer-setting/TCC errors are mandatory. See [SAFARI_RENDERED_DOM_ACQUISITION_DECISION.md](SAFARI_RENDERED_DOM_ACQUISITION_DECISION.md).

Rejected: WebDriver/Web Inspector automation path, Safari extension, Accessibility/System Events, embedded WebKit, Safari profile/session/cache/database scraping and hidden listener. Reasons include isolated automation windows/local server, no documented attach API, persistent browser permission surface, broad control, separate browsing context, private browser-file access or explicit no-listener conflict.

Protected areas: Khong Go/Swift/Xcode/bridge/runtime code, entitlement, Safari setting change, live Safari execution, production selector, `RawPost`, Phase 10A extraction, Phase 11, persistence, cookie/session/profile access, browser mutation, extension, Accessibility, WebKit hoac network listener.

Tests: Documentation/source audit, Go regression, vet, CLI output, diff/status and official-source link review.

Stop conditions: Official Apple evidence khong support exact mechanism; future slice needs arbitrary JavaScript, mutation, cookie/storage, browser files, listener, extension, Accessibility, WebKit, unbounded output, multiple tabs or background polling.

## Phase 10B2d - Bounded Safari rendered-DOM acquisition probe

Status: complete after automated acceptance and successful user-guided live Safari validation. The validated rendered document was approximately 2.9 MiB.

Exact scope: Acquisition-only implementation of the Phase 10B2c-approved mechanism. `AcquireSafariActiveTabRenderedDOM` executes fixed read-only `document.documentElement ? document.documentElement.outerHTML : ""` through Safari Apple Events on exactly the current tab of the front window. It returns exact HTTPS URL, optional title, rendered document and caller-supplied capture time with an 8 MiB decoded limit, 50,397,184-byte stdout envelope, 16 KiB stderr cap and 5-second timeout.

Protected areas: Khong production selector/parser, `RawPost`, Phase 10A automatic extraction, Phase 11, browser mutation/navigation/scroll/click/refresh, cookie/storage/session/profile access, extension, Accessibility, WebKit, listener, polling, retry, broad bridge/UI wiring hoac persistence.

Acceptance criteria: Automated synthetic parser/runner/boundary tests pass for exact current-tab targeting, fixed script, strict response, URL, decoded/transport limits, explicit process/TCC/Safari-setting errors, timeout/cancellation and forbidden-behavior audits. Successful acquisition alone does not approve selectors.

Tests: Fixed-script forbidden-token audit, exact command/target tests, strict response/bounds, URL, TCC/developer-setting, start/nonzero/timeout/cancellation errors, production-source boundary audit and full Go regression/vet/CLI. User-guided validation confirmed one-shot read-only acquisition from the expected active Facebook group tab without printing raw DOM. Packaged subprocess attribution remains deferred.

Stop conditions: Needs arbitrary script input, unstable target ownership, unbounded DOM, session/profile access, network, forbidden browser technology, source mutation or selector decisions.

Next milestone: Phase 10B2e performs separate redacted rendered-DOM reconnaissance. Phase 10B2b production selectors remain blocked until that evidence identifies stable post-level structure.

## Phase 10B2e - One-page rendered-DOM reconnaissance

Status: complete with STOP/INCONCLUSIVE result after exactly one acquisition.

Exact scope: Run only committed `AcquireSafariActiveTabRenderedDOM` once against one user-prepared active Facebook group tab and analyze private rendered DOM in memory for bounded redacted post-container, permalink, body, author, machine timestamp, group identity and traversal evidence.

Result: Acquisition and strict Facebook group URL guard passed, but the command-output filter retained only the passing test summary and discarded the redacted structural report. Group identity is STRONG. All post-level concepts are NOT FOUND in retained evidence; unavailable counts are not treated as zero or proof of absence. See [FACEBOOK_SAFARI_RENDERED_DOM_RECONNAISSANCE.md](FACEBOOK_SAFARI_RENDERED_DOM_RECONNAISSANCE.md).

Protected areas: No second acquisition, production selector/parser, `RawPost`, Phase 10A/11, private fixture, browser mutation, alternate browser mechanism, dependency, persistence, SQLite, SwiftUI, Xcode or bridge.

Acceptance result: Fail closed. Phase 10B2b remains blocked because the proceed bar requires STRONG retained evidence for every critical post field and sufficient traversal order.

Tests: Existing Go regression/vet/CLI plus documentation and repository privacy audits. No raw rendered DOM or private Facebook value entered the repository.

Stop condition reached: A second acquisition would be required just to recover the first run's missing redacted metadata. Any repeat must be a separately approved reconnaissance milestone, not an automatic retry.

## Phase 10B2f - Deterministic redacted reconnaissance reporting path

Status: complete after automated acceptance. Phase 10B2g subsequently validated the preservation path with one user-guided live result.

Exact scope: Add pure `AnalyzeRenderedDOMStructure(renderedDOM, pageURL)` and typed `RenderedDOMReconnaissanceReport` inside `internal/facebook`. The analyzer consumes one already-acquired bounded DOM plus optional page URL and emits only counts, URL-shape validity, deterministic traversal count, `STRONG`/`TENTATIVE`/`NOT_FOUND` confidence and bounded canonical marker categories.

Protected areas: No Safari acquisition/runtime/script change, production selector/parser, `RawPost`, Phase 10A/11, browser mutation, private fixture, dependency, filesystem/network access in analyzer, persistence, SQLite, SwiftUI, Xcode or bridge.

Acceptance criteria: Synthetic HTML tests prove semantic container/permalink/body/author/machine-time counts, group URL classification, traversal evidence, confidence downgrade, unstable-marker rejection, no private value in JSON, marker bounds and deterministic repeated output. Runtime source audit proves no browser, file, network, clock, selector or pipeline integration.

Preservation procedure for a separately approved manual run:

1. User leaves exactly one Facebook group tab active in Safari's front window.
2. Temporary helper calls `AcquireSafariActiveTabRenderedDOM()` exactly once.
3. Helper passes returned DOM and page URL directly in memory to `AnalyzeRenderedDOMStructure`.
4. Helper JSON-encodes only the typed redacted report.
5. Helper writes only that JSON to `/tmp/scanfb-rendered-dom-reconnaissance.json` with mode `0600`.
6. Helper prints only the temp path and a short success line.
7. Raw rendered DOM is never written to disk.
8. No second acquisition, retry or alternate browser mechanism is allowed.
9. No browser activation, navigation, tab switch, scroll, click, focus, refresh or other mutation is allowed.

Next step completed by Phase 10B2g: one separately user-guided live reconnaissance preserved the typed result. Phase 10B2b remains blocked because retained evidence did not reach its proceed bar.

Tests: Focused synthetic analyzer tests, full Go regression/vet/CLI, diff/status and production-source/privacy audits.

Stop conditions: Report would leak matched values, exceed bounds, require full parser dependency, need acquisition changes/browser mutation or become production selector logic.

## Phase 10B2g - Rendered-DOM live reconnaissance closeout

Status: complete with Phase 10B2b BLOCKED and the current Safari selector investigation closed.

Exact scope: Docs-only record of one successfully preserved Phase 10B2f typed report from one user-guided active Facebook group-page acquisition.

Result: 3,180,722 rendered-DOM bytes; two semantic article candidates; deterministic traversal count two; valid group-page URL shape. Permalink, body, author, machine timestamp, relative-time-only, complete-evidence and group-consistent permalink counts are all zero. Container, group identity and traversal confidence are STRONG; permalink, body, author and machine timestamp are NOT_FOUND. Recognized categories are only `role=article` and `dom-source-order`.

Decision: Phase 10B2b remains BLOCKED. Distinct containers and traversal do not satisfy the proceed bar without approved post identity, body, author and machine-readable timestamp evidence. Selector standards must not be weakened to use obfuscated classes, nth-child, localized/relative text, broad search or private-value heuristics.

Protected areas: No Go/Swift/Xcode/bridge/persistence/browser runtime change, selector/parser, `RawPost`, Phase 10A/11, private fixture, dependency, browser mutation or second acquisition in this closeout.

Next step: none within the current Safari selector approach. Any future investigation requires a separately justified new evidence or acquisition technique that preserves ScanFB's privacy and fail-closed boundaries.

Tests: Documentation consistency plus unchanged Go regression/vet/CLI and repository privacy/status audits.

## Phase 11 - Scan mot group thu cong

Exact scope: Scan mot group voi user-triggered action.

Protected areas: Khong batch 5 group, khong retry lien tuc.

Acceptance criteria: Mot group co attempt state va ket qua domain.

Tests: Manual validation va fixtures regression.

Stop conditions: Mat quyen truy cap group hoac selector quan trong thieu.

## Phase 12 - Scan batch 5 group

Exact scope: Mot explicit Scan action consumes the internal next-five selection, chay lan luot dung 5 group, advances persisted cursor as part of defined real batch progression, tra summary va dung. Explicit Scan action tiep theo moi consumes batch ke.

Protected areas: Khong manual queue-advance control, parallel group, infinite scroll, scheduler hoac cursor advance chi do preview/render.

Acceptance criteria: Summary day du sau batch, cursor progression atomic theo execution policy, batch dung cho user bam Scan tiep va preview khong mutate progression.

Tests: Batch integration tests va manual validation.

Stop conditions: Qua 00:00, CAPTCHA/checkpoint hoac fail-closed batch.

## Phase 13 - Hardening, Dry Run review va manual validation

Exact scope: On dinh UX review, error handling, logs va manual validation checklist.

Protected areas: Khong them he thong moi lon.

Acceptance criteria: Dry Run review giup phuc hoi/xac nhan rule, loi ro rang va khong mat raw record.

Tests: Regression matrix day du va manual validation.

Stop conditions: Can thay doi scope san pham lon.
