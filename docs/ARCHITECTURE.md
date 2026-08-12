# Architecture

Tai lieu nay la nguon chuan tac cho dependency boundaries cua ScanFB.

Phase 1 da chon Go lam ngon ngu chinh va tao skeleton package toi thieu. Tai lieu nay van la nguon chuan tac cho boundary.

## Nguyen tac

- Tach Facebook adapter khoi domain.
- Rule engine va deduplication phai test duoc hoan toan bang fixture, khong can Facebook hoac trinh duyet.
- Khong de selector DOM, browser automation hoac Facebook-specific parsing ro vao domain layer.
- MVP uu tien rule-based. Khong dung LLM hoac AI model trong critical classification path.
- Co the bo sung model local sau nay chi de assist/ranking, khong thay deterministic rules va khong duoc bia du lieu.
- ScanFB la buyer-only. Khong co seller mode, `LeadIntent` buyer/seller, `SellerLead`, `SellerIntentClassifier` hoac `SELLER_SCAN`.
- MacBook la Search Profile duy nhat trong MVP, nhung rule chung ve thoi gian, dia ly, author, blocklist va dedup phai nam ngoai logic rieng cua MacBook.

## Pipeline bat buoc

```text
Facebook page reader
-> RawPost
-> Normalization
-> SearchProfile target keyword matching
-> BuyerIntentClassifier
-> Geographic classification
-> Deduplication
-> Lead aggregation
-> Local persistence
-> UI
```

## Layers

Trong Go skeleton, cac layer duoc anh xa toi package:

- `cmd/scanfb`: CLI entry point toi thieu.
- `internal/domain`: entity, value object va invariant thuan; Phase 9B them `WatchedGroup` value model.
- `internal/application`: application services/use cases; phu thuoc domain, rules, dedup va blocklist. Phase 9A them Go-only in-memory lifecycle state machine cho mot batch production-shaped dung 5 group; Phase 9B them deterministic in-memory `WatchedGroupCollection`; Phase 9C them pure five-group round-robin selection policy; Phase 9D them narrow mapping tu mot approved `FiveGroupSelection` sang `ScanBatchLifecycle` inputs.
- `internal/orchestration`: thin synchronous use cases ket noi completed application result voi persistence-facing contract.
- `internal/rules`: deterministic buyer-intent, author, time va geographic rules.
- `internal/dedup`: duplicate detection va lead aggregation.
- `internal/persistence`: persistence-facing contracts, completed-batch SQLite implementation, va independent Phase 9E2 WatchedGroup-state SQLite schema v1/repository. WatchedGroup repository co narrow load/add/set-active/advance API, DELETE journal va khong co generic CRUD, migration execution hay completed-batch schema change.
- `internal/facebook`: adapter bien ngoai cho Facebook/browser; domain khong duoc import.
- `internal/ui`: Go-layer documentation/package placeholder. Native macOS UI implementation lives outside `internal/` using SwiftUI.
- `internal/bridge`: bridge-facing Go package for bounded typed `core_readiness` and Phase 9E2 persistent watched-group operations. Watched-group handlers open the Go-owned repository, apply Phase 9B/9C policy and return authoritative refreshed state; requests expose no raw path, SQL, collection or cursor authority. It does not access Facebook.
- `cmd/scanfb-bridge-helper`: local one-request subprocess helper for the bounded typed bridge; Phase 8I.2a packages it into Debug app bundles at `Contents/Helpers/scanfb-bridge-helper`.
- `macos/ScanFBApp`: native SwiftUI macOS app shell from Phase 8B plus the Phase 8 fixture screens, Phase 8I.2 readiness status and Phase 9E2 persistent Watched Groups presentation. Swift owns no storage path, SQLite access, cursor policy or durable state. It still has no lead/search bridge, Facebook integration or production scan workflow.

### App/UI

SwiftUI la approved native macOS presentation direction cho Phase 8. Phase 8B tao app shell tai `macos/ScanFBApp/`, nam ngoai Go `internal/` package tree. Phase 8C-8H them fixture presentation va session-only Leads interaction state. Phase 8I.2 them `CoreReadinessBridgeClient` cho explicit user-triggered readiness check. Phase 9E1 thay placeholder `Nhóm` bang Watched Groups UI; Phase 9E2 chuyen durable group/cursor authority sang Go. `WatchedGroupsStore` request state khi screen khoi tao, phan biet loading/loaded/failure, va chi ap dung authoritative response sau list/add/toggle/advance. Swift khong resolve path, mo SQLite, persist file, gui full collection/cursor nhu authority, sort hay chon group doc lap. Fixture values chi la display sample data; lead va Dry Run tab filtering khong infer business outcomes. Shell van khong co lead/search bridge, Facebook SDK/API, networking client, WebKit, browser automation, blocklist writes, lead status writes hoac production settings writes. UI khong import Facebook adapter truc tiep.

### Application services

Dieu phoi scan batch, time window, state machine va domain services. Day la noi noi adapter voi domain. Application khong import `internal/persistence` trong contract hien tai.

Phase 9A them `ScanBatchLifecycle` trong Go application layer de model state transition cho mot production-shaped batch gom dung 5 watched groups. Lifecycle nay chi o in-memory, su dung caller-supplied batch/attempt IDs, preserve order, cho phep toi da mot `running` attempt, va khong goi `RunScanBatch`, Facebook adapter, persistence, bridge, UI, scheduler, retry hoac concurrency. Day-boundary handling dung supplied time, compare theo `Asia/Ho_Chi_Minh`, va expire unfinished attempts fail-closed.

Phase 9B them `WatchedGroup` domain value va `WatchedGroupCollection` application service chi trong memory. Collection ho tro caller-supplied identity/time, metadata update, active/inactive state, lookup va stable insertion-order inspection cho so group khong gioi han. Phase nay khong chon next five groups, khong sort candidate cho scan, khong tao queue cursor, va khong goi Phase 9A lifecycle, Facebook, persistence, bridge, UI, scheduler, retry hoac concurrency.

Phase 9C them `SelectNextFiveActiveGroups` de doc deterministic WatchedGroup snapshot ma khong mutate collection. Policy traverses circular theo insertion order tu explicit caller-managed collection-position cursor, skip inactive, tra dung 5 distinct active groups va cursor tai collection position ngay sau group thu nam; neu thieu 5 active groups thi fail closed khong co partial result. `displayOrder`, `createdAt` va `lastSuccessfulScanAt` khong tham gia ordering. Cursor chi o memory cua caller; selector khong tao `ScanBatchLifecycle`, ID, scan execution, persistence, Facebook, bridge, UI, scheduler, retry hoac concurrency.

Phase 9D them `NewScanBatchLifecycleFromSelection` de map mot already-selected exact-five `FiveGroupSelection` sang `GroupScanAttemptInput` theo dung selected-group order. Caller cung cap batch ID, `ScanWindow` va dung 5 attempt ID; final Phase 9A validation van do `NewScanBatchLifecycle` so huu. Mapper khong goi selector, khong doc broader collection, khong advance/persist cursor, khong start transition, khong chay scan va khong goi Facebook, persistence, UI hoac bridge.

Phase 9E1 them bon bridge operations hep cho list, add, set-active va next-five. Vi helper la one-request subprocess, Swift gui full current-session snapshot va cursor tren moi call; Go tai tao `WatchedGroupCollection`, ap dung Phase 9B authoritative identity/URL/state rules va goi Phase 9C selector. Add chi nhan display name va canonical HTTPS URL tu UI, voi local ID va `createdAt` do Swift caller cung cap. Snapshot/cursor khong persisted; slice nay khong goi lifecycle, scan execution, Facebook, SQLite, scheduler, retry hoac concurrency.

Phase 9E2 giu cung bon operation nhung thay the Phase 9E1 session authority. Helper resolve va mo dedicated Go-owned store tren moi call; list doc state, add/set-active commit mutation, va next-five atomically persist exact Phase 9C next cursor. Response thanh cong tra full ordered WatchedGroups, current selection va current cursor; request khong mang authoritative collection/cursor hay filesystem path. Phase 9E3b dung `watched_groups_add` lam primary one-time enrollment path va giu `watched_groups_next_five` lam internal progression primitive; primary UI chi doc preview tu list response. Slice khong goi lifecycle, scan execution, Facebook, scheduler, retry hoac concurrency.

### Use-case orchestration

Ket noi application services voi persistence-facing contract o boundary mong. Phase 5E chi co `RunAndSaveScanBatch`: nhan caller-supplied `BatchRecordID`, goi `application.RunScanBatch`, convert successful result bang `persistence.NewBatchRecord`, save dung mot lan qua `BatchRepository`, va chi tra completed result/record sau khi save thanh cong. Orchestration khong import Facebook adapter, UI hoac CLI.

### Domain

Chua entity, value object, SearchProfile, BuyerIntentClassifier, rule engine, geographic classifier, deduplication va lead aggregation. Domain khong import UI, persistence implementation, browser automation hoac Facebook selector.

Domain duoc giu trung lap o phan tai su dung nhu `ScanSession`, `ScanBatch`, `RawPost`, `AuthorIdentity`, `GeographicClassification`, `FilterDecision`, `LeadSource`, `PostDeduplicator` va `SearchProfile`. Phan lead/filter duoc phep ghi ro buyer-only bang `BuyerIntentClassifier`, `BuyerLead` hoac `Lead` voi invariant buyer-only.

### Persistence interfaces

Dinh nghia persistence-facing contract cho completed scan batch snapshot. Phase 5C chi co opaque `BatchRecordID`, completed `BatchRecord`, structural validation va save-only `BatchRepository.SaveBatch`; khong co list/update/delete/search/paging/schema/migration/transaction API. Phase 5D them `InMemoryBatchRepository` lam concrete adapter chi trong memory; inspection methods nam tren adapter, khong mo rong `BatchRepository`. Phase 5G3 them concrete-only `SQLiteBatchRepository.LoadBatch`, khong them method vao `BatchRepository`. `internal/persistence` duoc phu thuoc `internal/application` va `internal/domain` de copy completed batch result; application, domain, rules, dedup va blocklist khong phu thuoc persistence.

### Persistence implementation

Phase 5G2 them durable SQLite `SaveBatch` cho mot completed `BatchRecord` snapshot. Phase 5G3 them fail-closed `SQLiteBatchRepository.LoadBatch(id BatchRecordID) (BatchRecord, error)` de reconstruct mot complete snapshot tu schema version 1 trong read transaction. In-memory adapter van chi validate/save completed snapshot trong process de test va future wiring. Phase 5F chon SQLite la local durable storage technology va ghi schema design tai [PERSISTENCE_SCHEMA.md](PERSISTENCE_SCHEMA.md). Phase 5G1 them `SQLiteBatchRepository` bootstrap trong `internal/persistence`: open/create explicit local SQLite path, enable va verify foreign keys, create empty schema version 1 transactionally, validate schema metadata, va `Close`.

`SQLiteBatchRepository` satisfy `BatchRepository` bang `SaveBatch(record BatchRecord) error`. Method nay validate `BatchRecord` truoc khi mutate database, ghi root va toan bo child collections trong mot transaction, translate duplicate `BatchRecordID` thanh `ErrBatchRecordAlreadyExists`, rollback moi write failure, va khong retry hay partial write. Concrete `LoadBatch` verify schema version 1, load child rows bang explicit positions, decode canonical timestamps/booleans, reject malformed stored enum-like values, run `BatchRecord.Validate`, va return zero record on failure. Khong co list/update/delete/search/paging/schema/migration/transaction API tren public contract.

Phase 9E2 implements [WATCHED_GROUP_PERSISTENCE_DECISION.md](WATCHED_GROUP_PERSISTENCE_DECISION.md): Go-owned production storage nam trong OS-provided user Application Support root, thu muc `com.soleda.ScanFB`, voi hai database `completed-batches.sqlite3` va `watched-groups.sqlite3`. Existing completed-batch schema v1 giu nguyen; WatchedGroup-state database co schema v1 rieng, DELETE journal, explicit insertion positions va exact Phase 9C cursor. Helper resolve production path, Swift khong thay path; tests inject explicit temporary path. Khong co migration runner.

WatchedGroup rows la source-neutral. Phase 9E3b enrolls tung user-approved group mot lan qua existing Add operation; Go persistence giu identity/metadata/active/order qua cac lan mo app, con Swift chi consume authoritative response. Active/inactive la user-controlled eligibility. Enrollment khong verify hoac luu Facebook membership, access permission, account, cookie, session hay browser identity. Cursor chi duoc advance boi future real batch progression, khong boi render/read preview va khong boi mot primary UI queue control.

Phase 9E3a performs one docs-only, user-guided reconnaissance through the existing Phase 10B2d rendered-DOM acquisition. The active HTTPS Facebook `/groups/...` page and one canonical group-link/name association were structurally plausible, but no semantic group-item boundary or machine-readable joined-membership discriminator was found; traversal evidence was only tentative. [FACEBOOK_JOINED_GROUPS_RECONNAISSANCE.md](FACEBOOK_JOINED_GROUPS_RECONNAISSANCE.md) therefore records STOP/INCONCLUSIVE. No discovery runtime edge exists and automatic joined-group discovery remains blocked historical research, but it is not an MVP dependency. Phase 9E4 synchronization is optional future research rather than a gate for enrollment or production scan work.

Phase 9E3b restores the existing Add sheet as a setup/configuration flow. The user supplies the canonical HTTPS Facebook group URL and the display name required by the current domain contract; `WatchedGroupsStore` sends the existing add request, then renders only the authoritative Go response. The disabled discovery affordance is removed, empty-state copy explains one-time local enrollment, toggles retain active eligibility semantics, and next-five remains read-only. No new bridge operation, schema, Swift persistence, cursor advancement, Facebook access or scan execution is added. A future scan access failure must become an explicit group attempt/result rather than silently deleting the enrolled group.

Phase 11A adds application-facing one-group request/collector/result contracts in `internal/application` and a Go-only `RunOneGroupScan` use case in `internal/orchestration`, preserving the existing rule that orchestration imports application rather than domain/blocklist directly. The injected `GroupPostCollector` receives exactly one immutable-style enrolled `WatchedGroup` plus the existing `ScanWindow`, and returns an exact group identity plus ordered already-normalized `RawPost` values. The use case rejects inactive groups, calls the collector once, maps the caller-supplied scan/attempt IDs into the same Phase 9A lifecycle transitions, and passes successful collection to Phase 5B `RunScanBatch` as exactly one `GroupBatch`. It does not import `internal/facebook` or persistence, does not select five groups, does not advance a cursor, and has no retry, goroutine, scheduler, network, Swift or bridge edge. Phase 11B remains blocked because no reliable production post collector/source contract exists.

Phase 11B0 adds only [PRODUCTION_GROUP_COLLECTOR_SOURCE_DECISION.md](PRODUCTION_GROUP_COLLECTOR_SOURCE_DECISION.md). The decision evaluates Safari DOM and structured page-side output, bounded user-prepared input, the official Meta API, Safari extension, Accessibility and alternate automation paths, then DEFERs because none currently proves every `RawPost` field with trustworthy provenance. It adds no dependency or runtime edge: no collector, selector, API client, extension, WebKit/Accessibility path, persistence, bridge/UI, Phase 11A or cursor behavior changes. A future implementation decision requires materially new source evidence rather than another wrapper around the same incomplete rendered DOM.

Phase 11C0 adds only [MVP_SCAN_INPUT_WORKFLOW_DECISION.md](MVP_SCAN_INPUT_WORKFLOW_DECISION.md) and does not reopen Phase 11B. It approves a transitional per-group input boundary: one ScanFB-owned manual form produces one versioned JSON DTO capped at 1 MiB and 100 ordered posts. Authoritative Go state supplies the active enrolled-group identity, the caller supplies `capturedAt` from the same instant as `ScanWindow.ScanStarted()`, and the user explicitly supplies body, author, absolute creation time and available post identity/permalink. A future Go adapter may strictly decode that DTO, construct Phase 10A `PreparedPageSnapshot`, and satisfy the unchanged Phase 11A collector interface. Phase 11C0 adds no runtime/dependency edge, UI/bridge operation, file/clipboard access, persistence, browser behavior, batch execution or cursor movement.

### Facebook adapter

Phase 10A nhan typed local `PreparedPageSnapshot`, validate fail-closed va map deterministic sang ordered `RawPost` values. Phase 10B1 them mot Safari-only acquisition boundary: explicit caller action goi truc tiep `/usr/bin/osascript` bang JXA, doc chi `currentTab` cua front Safari window, va tra URL/title/page source gioi han cung caller-supplied `capturedAt`. Nguoi dung so huu viec mo Safari, dang nhap, dieu huong va chon active tab. Adapter khong dung shell, Accessibility/UI scripting, WebKit, extension, network/listener, khong auto-login/navigation/tab switching/scrolling/polling, va khong doc cookie, credential, session, Keychain, profile, history hoac cache. stdout machine response va bounded stderr diagnostics tach rieng; raw stderr chi dung noi bo de classify loi va khong duoc tra ve caller. Timeout/cancellation chi ket thuc process acquisition do call do so huu. Apple Events Automation permission la dependency macOS explicit va bi tu choi thi fail closed.

Phase 10B1 chi acquire mot bounded page snapshot; no khong claim production Facebook DOM selectors, khong map sang `RawPost`, khong goi Phase 10A extraction, application pipeline, lifecycle, persistence, UI hoac bridge. Production selector validation thuoc Phase 10B2. Facebook boundary khong chua rule domain, khong biet MacBook-specific extraction va khong biet seller behavior.

Phase 10B1 manual validation da thanh cong tren mot user-prepared Facebook group page voi source khoang 1.5-1.6 MB, duoi decoded bound 4 MiB. Phase 10B2a read-only reconnaissance tren dung mot page cho thay `tab.source()` chi expose HTML/bootstrap shell: khong co stable post container, permalink/ID, body, author hoac absolute timestamp marker de tao fail-closed selector. Vi vay Phase 10B2a la blocked/inconclusive va khong co edge moi tu Safari snapshot toi `RawPost`; Phase 10B2b production selector khong duoc bat dau tren acquisition source hien tai.

Phase 10B2c documentation decision tai [SAFARI_RENDERED_DOM_ACQUISITION_DECISION.md](SAFARI_RENDERED_DOM_ACQUISITION_DECISION.md) approve duy nhat mot future acquisition-only mechanism: fixed read-only page-side JavaScript duoc Safari execute qua Apple Events tren current tab cua front window, voi user-trigger, exact HTTPS URL validation, bounded output va fail-closed permission/process errors. User phai explicit enable Safari developer setting `Allow JavaScript from Apple Events` va grant macOS Automation permission. `NSAppleEventsUsageDescription` la user-facing privacy purpose string bat buoc khi application dung API gui Apple Events; Phase 10B2d phai validate exact TCC/signing attribution cua direct `osascript` subprocess va final Hardened Runtime/App Sandbox configuration. Neu configuration do yeu cau `com.apple.security.automation.apple-events`, entitlement phai chi scope cho Safari automation; arbitrary-app automation va broad temporary exception khong duoc approve. Hien tai khong co rendered-DOM code, entitlement, Xcode change, selector, `RawPost` edge, browser mutation, extension, Accessibility, WebKit hoac listener.

Phase 10B2d implements `AcquireSafariActiveTabRenderedDOM` trong `internal/facebook`. Mot explicit caller action owns mot direct `/usr/bin/osascript` process, JXA targets `windows[0].currentTab()` va executes duy nhat fixed page-side expression `document.documentElement ? document.documentElement.outerHTML : ""`. Result chi gom exact HTTPS URL, optional title, rendered document toi da 8 MiB va caller-supplied `CapturedAt`; stdout envelope la 50,397,184 bytes, stderr van 16 KiB va timeout van 5 giay. API khong nhan caller JavaScript, khong co selector/`RawPost` edge, browser mutation, cookie/storage/session/profile access, network/listener, Accessibility/System Events, extension/WebKit, persistence, SwiftUI hoac bridge. Automated acceptance va user-guided live validation da pass; packaged TCC/signing attribution van deferred, va selector work van blocked cho den khi separate redacted rendered-DOM reconnaissance dat proceed bar.

Phase 10B2e executes exactly one production rendered-DOM acquisition for docs-only reconnaissance. The expected Facebook group URL guard passed, but the command-output filter did not retain the analyzer's redacted structural report. No post-level runtime edge is therefore approved: container, permalink, body, author, machine timestamp and traversal evidence remain unestablished. [FACEBOOK_SAFARI_RENDERED_DOM_RECONNAISSANCE.md](FACEBOOK_SAFARI_RENDERED_DOM_RECONNAISSANCE.md) records STOP/INCONCLUSIVE; Phase 10B2b remains blocked and Phase 11 is unchanged.

Phase 10B2f adds pure `AnalyzeRenderedDOMStructure` inside `internal/facebook` as reconnaissance tooling, not product extraction. Input is only one rendered-DOM string plus optional page URL; output is a bounded typed report containing counts, group-page URL-shape validity, `STRONG`/`TENTATIVE`/`NOT_FOUND` confidence and canonical redacted marker names. The analyzer keeps no text-node value in its report, returns no URL/ID/body/author data, uses no browser/filesystem/network/clock/global state and has no edge to acquisition, `RawPost`, Phase 10A/11, persistence, SwiftUI or bridge. A temporary manual helper, outside runtime, may JSON-encode only this report to a mode-0600 `/tmp` file.

Phase 10B2g records the preserved live Phase 10B2f report and closes the current Safari rendered-DOM selector investigation. Acquisition and redacted preservation work; one page yielded two `role=article` candidates in deterministic DOM order. Zero candidates had approved permalink, body, author or machine timestamp evidence, and complete evidence count was zero. No selector/`RawPost` edge is added, Phase 10B2b remains blocked, and further Safari selector work requires a separately justified evidence or acquisition technique rather than weaker markers.

## Dependency direction

```text
App/UI -> Application services -> Domain
App/UI -> Use-case orchestration -> Application services
Use-case orchestration -> Persistence-facing contracts
Persistence-facing contracts -> Application services
Persistence-facing contracts -> Domain
Persistence implementation -> Persistence-facing contracts
Facebook adapter -> Application services
Facebook adapter -> RawPost mapping contract
```

Facebook adapter khong duoc domain import nguoc lai. Domain khong duoc import adapter.

Application, domain, rules, dedup, blocklist va persistence khong duoc import `internal/orchestration`. `internal/orchestration` chi duoc import `internal/application`, `internal/persistence` va standard library trong production code.

Native SwiftUI code phai song ngoai `internal/`. Go application/domain packages khong duoc phu thuoc Swift hoac macOS UI code. Phase 8I.1 selected local subprocess request/response between SwiftUI and a bundled Go helper, documented in [BRIDGE_DECISION.md](BRIDGE_DECISION.md). Phase 8I.2 implements only `core_readiness`; future slices must not use HTTP by default, cloud API, direct SQLite access from Swift, arbitrary JSON maps or a broad command bus.

Chi tiet Phase 8 nam trong [MACOS_UI_ARCHITECTURE.md](MACOS_UI_ARCHITECTURE.md).

## Search Profile boundary

`SearchProfile` dinh nghia buyer target cua mot lan scan. MVP chi co built-in MacBook Search Profile. Cac profile buyer khac nhu iPhone, may anh hoac laptop chi la future architecture note sau MVP, khong phai acceptance criterion cua MVP.

Khong xay abstraction, enum hoac code cho app tim nguoi can ban trong ScanFB hien tai. Neu sau nay can san pham tim nguoi can ban, do la du an/app khac co the hoc tu ScanFB hoac tai su dung core da tach dung boundary.

## Loi fail-closed

Khi adapter gap CAPTCHA, checkpoint, login required, verification page, DOM unknown, missing critical selector hoac lost group access, application service phai danh dau group/batch that bai dung muc va dung workflow theo [SCAN_RULES.md](SCAN_RULES.md).

## UI technology

SwiftUI is the approved native macOS presentation technology for Phase 8. Phase 8C adds only static fixture-driven Overview presentation at `macos/ScanFBApp`; Phase 8D adds only fixture-driven buyer Leads presentation; Phase 8E adds only fixture-driven Dry Run review presentation; Phase 8F adds only fixture-driven Blocklist and Settings presentation; Phase 8G adds only session-memory Leads interaction state; Phase 8H adds only a stateless fixture source URL handoff through SwiftUI `openURL`. The Go core remains authoritative for rules, deduplication, blocklist behavior, batch evaluation, lead aggregation, persistence snapshots, reason codes, settings semantics and deterministic decisions.

## Bridge decision

Phase 8I.1 selected local subprocess request/response as the bridge model.
Phase 8I.2 implements the first read-only slice: Swift launches one explicitly
resolved helper process for one `core_readiness` request, writes the bounded
typed request to stdin, reads one bounded machine-readable response from stdout
and maps transport errors explicitly. The response contains only
`schema_version`, `readiness_status` and `core_identity`. Diagnostics are
stderr-only, bounded and not user-visible raw output. The slice has a 2.0 second
timeout and terminates the owned helper on timeout/cancellation, with a 0.5
second force-kill grace. Phase 8I.2a adds Debug-only helper packaging at
`Contents/Helpers/scanfb-bridge-helper`; Release distribution, signing,
notarization and hardened-runtime policy remain deferred. Runtime resolution
fails closed when the bundled helper is absent.

Phase 9E1 introduced that one-request subprocess boundary for only
`watched_groups_list`, `watched_groups_add`, `watched_groups_set_active` and
`watched_groups_next_five` with session-owned Swift state.

Phase 9E2 persistent watched-group calls resolve the
dedicated Application Support store internally in the helper. No bridge request
or response carries a raw filesystem path, SQL, database handle or database-local
ID. Requests no longer carry the Swift snapshot/cursor as authority. Responses
return authoritative Go-restored state; the finite `watched_groups_next_five`
operation is the only explicit cursor-advance intent. This is not a broad command
bus, lead/search bridge or production scan orchestration.
