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

Exact scope: Add deterministic fixture lead tabs and lead cards for buyer-only lead presentation.

Protected areas: Khong seller tab, seller mode, Go bridge, SQLite direct access, Facebook data, status persistence, scoring recomputation hoac reason-code generation in Swift.

Acceptance criteria: Lead tabs and cards present sample buyer leads, source counts, reason-code display and disabled/placeholder actions without claiming live data.

Tests: Xcode build, manual fixture validation, accessibility spot checks where practical, stale-process kill before relaunch, Go regression checks.

Stop conditions: Need real lead loading, bridge, status workflow, persistence hoac Facebook adapter.

## Phase 8E - Fixture Dry Run review

Exact scope: Add deterministic fixture Dry Run review screen for included/review/excluded sample posts.

Protected areas: Khong rule recomputation, reason-code inference, restore behavior backed by persistence, Facebook data, bridge, network hoac seller review mode.

Acceptance criteria: Dry Run is presented as default-on product behavior; sample/demo labels are clear; rejected/review sample reasons preserve exact codes and diacritics.

Tests: Xcode build, manual fixture validation, stale-process kill before relaunch, Go regression checks.

Stop conditions: Need real filter decisions, persistence, bridge, user edits hoac production review workflow.

## Phase 8F - Fixture settings and blocklist presentation

Exact scope: Add deterministic fixture settings and blocklist screens.

Protected areas: Khong persistence, SQLite direct access, real blocklist import/export, Facebook identity lookup, network, bridge, credentials, cookies hoac production settings writes.

Acceptance criteria: Settings and blocklist are sample/demo only, local-first/privacy copy is visible where appropriate, display name is not represented as authoritative block identity.

Tests: Xcode build, manual fixture validation, stale-process kill before relaunch, Go regression checks.

Stop conditions: Need real settings storage, blocklist mutation, database path, bridge hoac identity resolution.

## Phase 8G - Fixture UI interaction state

Exact scope: Add in-memory fixture-only UI state for viewed/contacted/ignored interactions.

Protected areas: Khong persistence, database, bridge, production lead status, Facebook action, network, business logic recomputation hoac seller workflow.

Acceptance criteria: Interactions affect only the current in-memory fixture session and reset on relaunch; UI does not imply production persistence.

Tests: Xcode build, Swift tests if state reducers/helpers exist, manual interaction validation after stale-process kill/relaunch, Go regression checks.

Stop conditions: Need status persistence, Go integration, SQLite, import/export hoac real lead workflow.

## Phase 8H - Bridge evaluation and architecture decision

Exact scope: Documentation/prototype-decision milestone to evaluate C-compatible exported Go library, subprocess with structured stdin/stdout, and local IPC boundary, then select exactly one narrow integration model.

Protected areas: Khong production bridge, UI wiring, direct Swift SQLite access, HTTP server by default, cloud API, arbitrary JSON maps, credential transfer, database-local ID exposure hoac business logic in Swift.

Acceptance criteria: Decision compares deterministic request/response behavior, no hidden network, no credential transfer, explicit schemas, error propagation, cancellation, build complexity, packaging, testing, crash isolation and Apple Silicon support.

Tests: Documentation consistency and any approved tiny prototype checks only if explicitly scoped.

Stop conditions: Need production integration, broad API, live Facebook data, database path policy hoac bridge implementation beyond decision scope.

## Phase 8I+ - Narrow Go integration after bridge decision

Exact scope: Only after Phase 8H, add tiny bridge-specific integration milestones for one request/response slice at a time.

Protected areas: Khong broad bridge API, arbitrary JSON maps, direct Swift SQLite access, hidden networking, Facebook automation, seller mode, migration execution, search/list expansion without separate milestone hoac business-rule duplication.

Acceptance criteria: Each slice has explicit schema, deterministic fixture-backed behavior, error propagation, cancellation policy, tests on both sides as needed, and no regression to Go core checks.

Tests: Bridge-specific unit/integration tests, Xcode build/manual launch, stale-process kill before relaunch, Go `go test ./...`, `go vet ./...`, CLI output unchanged.

Stop conditions: Need broad production workflow, Facebook adapter, database path policy, packaging/signing/notarization hoac unsupported bridge behavior.

## Phase 9 - Group management va batch state machine

Exact scope: Quan ly group, queue batch 5 group, attempt state.

Protected areas: Khong doc Facebook that.

Acceptance criteria: Batch state dung khi success, failed, skipped va day boundary.

Tests: T19-T20 va state machine tests.

Stop conditions: Can adapter Facebook de tiep tuc logic domain.

## Phase 10 - Browser integration thu nghiem voi mot trang do nguoi dung mo

Exact scope: Adapter doc mot trang/group dang mo do nguoi dung chuan bi.

Protected areas: Khong auto login, khong mass profile open, khong batch automation.

Acceptance criteria: Adapter tao `RawPost` hoac fail-closed ro rang.

Tests: Manual validation co log khong secret.

Stop conditions: CAPTCHA, checkpoint, login required, DOM unknown.

## Phase 11 - Scan mot group thu cong

Exact scope: Scan mot group voi user-triggered action.

Protected areas: Khong batch 5 group, khong retry lien tuc.

Acceptance criteria: Mot group co attempt state va ket qua domain.

Tests: Manual validation va fixtures regression.

Stop conditions: Mat quyen truy cap group hoac selector quan trong thieu.

## Phase 12 - Scan batch 5 group

Exact scope: Chay lan luot dung 5 group va tra summary.

Protected areas: Khong parallel group, khong infinite scroll, khong scheduler.

Acceptance criteria: Summary day du sau batch, batch dung cho user bam tiep.

Tests: Batch integration tests va manual validation.

Stop conditions: Qua 00:00, CAPTCHA/checkpoint hoac fail-closed batch.

## Phase 13 - Hardening, Dry Run review va manual validation

Exact scope: On dinh UX review, error handling, logs va manual validation checklist.

Protected areas: Khong them he thong moi lon.

Acceptance criteria: Dry Run review giup phuc hoi/xac nhan rule, loi ro rang va khong mat raw record.

Tests: Regression matrix day du va manual validation.

Stop conditions: Can thay doi scope san pham lon.
