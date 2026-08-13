# Testing

Testing cua ScanFB uu tien deterministic fixtures cho domain, rule engine, geographic classifier, deduplication va lead aggregation. Facebook integration chi duoc them sau khi domain/rule engine da co test rieng.

## Nguyen tac

- Rule engine va deduplication phai test duoc khong can Facebook hoac browser.
- Test phai kiem tra reason codes, khong chi kiem tra ket qua UI.
- Moi production code phai co test phu hop voi risk va blast radius.
- MVP khong dung LLM trong critical classification path nen test phai deterministic.
- Dry Run phai giu raw record bi loai de review.
- Test phai xac nhan ScanFB buyer-only va khong co product behavior cho seller mode.
- Rule thoi gian, dia ly, author, blocklist va dedup la rule chung, khong phu thuoc MacBook.
- Author/time rule tests phai kiem tra exact reason codes va deterministic ordering.
- Buyer-intent rule tests phai dung synthetic posts, active `SearchProfile`, exact reason codes, seller/noise precedence va boundary-aware matching.
- Future native macOS UI milestones phai build app moi, kill stale app process, relaunch rebuilt app bundle va xac minh dung bundle moi dang duoc test truoc khi manual verification.

## Native macOS app-shell checks

Phase 8B verifies the empty native SwiftUI app shell with these checks:

- Build: `xcodebuild -project macos/ScanFBApp/ScanFBApp.xcodeproj -scheme ScanFBApp -configuration Debug -derivedDataPath /tmp/scanfb-phase8b-derived-data CODE_SIGNING_ALLOWED=NO build`
- Test: `xcodebuild -project macos/ScanFBApp/ScanFBApp.xcodeproj -scheme ScanFBApp -configuration Debug -derivedDataPath /tmp/scanfb-phase8b-derived-data -destination 'platform=macOS,arch=arm64' CODE_SIGNING_ALLOWED=NO test`
- App bundle exists at `/tmp/scanfb-phase8b-derived-data/Build/Products/Debug/ScanFB.app`.
- Stale `ScanFB` app processes are terminated before relaunch.
- The rebuilt app bundle is launched and the running executable path resolves inside that rebuilt bundle.
- There must not be multiple `ScanFB` processes after launch verification.
- Automated verification covers build, tests, bundle existence, process start, executable path and cleanup.
- Manual verification covers visible native window, app identity, sidebar navigation and placeholder detail screens.
- Go regressions remain required: `go test ./...`, `go vet ./...`, temporary CLI build, and CLI output `ScanFB foundation ready`.

## Native macOS static Overview fixture checks

Phase 8C verifies the static Overview dashboard with these checks:

- Swift unit tests cover exactly five groups, coherent group totals, fixed date/profile/mode/Dry Run values, non-negative metrics, decision totals equaling reviewed posts, exclusion-reason order, exclusion totals equaling excluded count, unique metric/reason IDs, sample/demo labeling, deterministic repeated fixture access and non-empty user-facing labels.
- Manual Overview checks cover visible sample-data label, disclaimer that data is not connected to Go core and does not come from Facebook, fixed date, geographic mode, Search Profile, Dry Run state, group-run counts, four primary metrics, included/review/excluded counts and five exclusion reasons in required order.
- Manual layout checks cover default window readability, Light and Dark appearance, absence of charts, absence of Scan/refresh controls, absence of Facebook links/data and unchanged placeholders for Leads, Dry Run, Nhóm, Blocklist and Cài đặt.
- Stale-process termination, rebuilt-bundle launch verification and Go regression checks remain required.

## Native macOS fixture Leads checks

Phase 8D verifies the fixture-only Leads screen with these checks:

- Swift unit tests cover exactly four synthetic buyer leads, stable IDs
  `lead-sample-001` through `lead-sample-004`, three eligible leads, one review
  lead, stable tab titles, tab counts, deterministic tab filtering, fixed dates,
  positive source counts, non-empty visible strings, no fixture links,
  deterministic repeated fixture access and repository-existing reason-code
  display strings supplied by fixture data.
- Manual Leads checks cover tabs `Tất cả`, `Đủ điều kiện` and `Cần xem xét`,
  counts 4/3/1, initial all-tab selection, four synthetic buyer cards, sample
  data labeling, disabled placeholder actions, unchanged card content across
  tab changes, and absence of live Facebook links or data.
- Manual layout checks cover default window readability, Light and Dark
  appearance, no clipping, no overlap, full scrolling behavior, no horizontal
  overflow and unchanged placeholders for Dry Run, Nhóm, Blocklist and Cài đặt.
- Stale-process termination, rebuilt-bundle launch verification and Go
  regression checks remain required.

## Native macOS fixture Dry Run checks

Phase 8E verifies the fixture-only Dry Run screen with these checks:

- Swift unit tests cover exactly ten synthetic posts, stable IDs
  `post-sample-001` through `post-sample-010`, three included posts, two review
  posts, five excluded posts, stable tab titles, tab counts, deterministic tab
  filtering, fixed dates, non-empty visible strings, no fixture links,
  deterministic repeated fixture access and repository-existing reason-code
  display strings supplied by fixture data.
- Manual Dry Run checks cover tabs `Được chọn`, `Cần xem xét` and `Đã loại`,
  counts 3/2/5, initial included-tab selection, ten synthetic post cards,
  sample data labeling, disabled placeholder action, unchanged card content
  across tab changes, and absence of live Facebook links or data.
- Manual layout checks cover default window readability, Light and Dark
  appearance, no clipping, no overlap, full scrolling behavior, no horizontal
  overflow and unchanged placeholders for Nhóm, Blocklist and Cài đặt.
- Stale-process termination, rebuilt-bundle launch verification and Go
  regression checks remain required.

## Native macOS fixture Blocklist and Settings checks

Phase 8F verifies the fixture-only Blocklist and Settings screens with these
checks:

- Swift unit tests cover exactly four synthetic blocklist entries, stable IDs
  `block-sample-001` through `block-sample-004`, supported identity kinds only,
  no display-name-only identity kind, deterministic dates, non-empty reasons,
  `.invalid` URL display strings and no real Facebook domain.
- Swift unit tests cover four settings sections in stable order, required
  read-only rows, non-empty labels and values, Dry Run default `Bật`, maximum
  groups `5`, Go bridge initial state `Chưa kiểm tra`, Facebook integration `Chưa triển khai`,
  SwiftUI direct SQLite access `Không`, deterministic repeated construction and
  stable row order.
- Manual checks cover visible `Dữ liệu minh họa` labels, sample-only
  disclaimers, disabled blocklist actions, read-only settings presentation,
  display-name-only identity not treated as authoritative, no live Facebook
  links/data and unchanged `Nhóm` placeholder.
- Build/test verification uses Xcode DerivedData outside the repository.

## Native macOS fixture Leads interaction-state checks

Phase 8G/8H verifies the fixture-only Leads interaction state and browser
handoff action with these checks:

- Swift unit tests cover all four fixture leads starting as `new`, stable lead
  IDs, exact supported states `new/viewed/ignored`, deterministic transitions
  `new -> viewed`, `new -> ignored` and `viewed -> ignored`,
  selected-lead-only mutation, fresh construction reset, and non-empty
  deterministic status labels.
- Swift unit tests verify `Tương tác` does not change interaction state,
  eligibility categories, reason codes or fixture order.
- Swift unit tests verify every lead has a stable synthetic HTTPS source URL,
  source URLs are stable across repeated construction, malformed and non-HTTPS
  URLs are rejected, valid HTTPS URLs are accepted, validation is deterministic,
  browser handoff receives exactly the validated URL, and invalid handoff input
  does not invoke the injected open closure.
- Manual checks, when a launch milestone allows them, cover visible labels
  `Mới`, `Đã xem` and `Bỏ qua`, compact card actions `Đánh dấu đã xem`,
  `Tương tác` and `Bỏ qua`, accessibility labels with current state, no new
  status tab/filter, and unchanged Overview, Dry Run, Nhóm, Blocklist and
  Cài đặt.
- Build/test verification uses Xcode DerivedData outside the repository.

## Native macOS Watched Groups checks

Phase 9E1 verifies the minimal session-only Watched Groups screen and typed
bridge boundary with these checks:

- Go bridge tests cover empty, fewer-than-five, exact-five and more-than-five
  snapshots; add and active-state mutation through Phase 9B; Phase 9C order,
  inactive skipping, cursor continuation, bounded transport and fail-closed
  malformed/duplicate/not-found input.
- Swift tests cover deterministic typed request/response coding, empty and
  insufficient UI state, exact bridge-returned 1-5 order, inactive groups,
  add validation, active toggles, explicit cursor advance and caller-supplied
  local ID/time.
- Source audits confirm Swift does not sort or select groups independently and
  that the slice adds no persistence, SQLite, Facebook/Safari, scan execution,
  scheduler, retry or concurrency behavior.
- Build/test verification uses Xcode DerivedData outside the repository; this
  milestone does not launch the app.
- User-guided closeout verification used the freshly built Debug bundle and
  passed app launch, Groups navigation, empty state, add form, acceptance of a
  real HTTPS Facebook group identity without retaining it in documentation,
  immediate active-row display, both active-toggle directions, and the
  insufficient-active state without a fabricated partial selection.
- Manual selection checks returned exactly five groups in application order
  for five active groups. With six active groups, one explicit
  `Chuyển lượt chọn` action advanced presentation from `1,2,3,4,5` to
  `6,1,2,3,4`, confirming session cursor behavior. No scan execution or
  browser/Safari action occurred.
- The mixed Vietnamese and `Watched Groups`/`Next 5 Groups` labels are a
  non-blocking future UI polish item, not a Phase 9E1 acceptance failure.

## WatchedGroup persistence decision checks

Phase 9E2a is documentation-only. Its checks confirm:

- the selected logical paths are
  `<user-application-support>/com.soleda.ScanFB/completed-batches.sqlite3` and
  `<user-application-support>/com.soleda.ScanFB/watched-groups.sqlite3`, with no
  hardcoded username, repository path or DerivedData path;
- the existing completed-batch schema remains version 1 and the future
  WatchedGroup-state database starts an independent schema version 1;
- the state design covers every existing Phase 9B field, explicit insertion
  position and one exact Phase 9C cursor in the same store/transaction boundary;
- zero groups require cursor zero, while malformed, negative or out-of-range
  cursor values fail closed without modulo repair;
- Go owns path resolution, directory/database creation and restore validation;
  Swift sees no raw path and owns no persistence;
- future tests override storage only through explicit temporary Go repository
  paths; production Application Support is not touched by automated tests;
- no runtime/schema/migration/bridge/Xcode change, database file,
  `UserDefaults`, `@AppStorage`, Phase 11, Facebook/Safari, browser/session data,
  scheduler, worker, network or cloud behavior is introduced.

## WatchedGroup persistence implementation checks

Phase 9E2 verifies the implemented Go-owned persistent flow:

- temporary explicit-path repository tests cover schema v1 bootstrap, empty
  state, full metadata and insertion-order reopen, active toggles, authoritative
  identity conflicts, exact cursor persistence/advance and deterministic reopen;
- malformed rows, schema inventory/version, metadata chronology, insertion
  positions and zero/non-empty cursor bounds fail closed without repair;
- add, set-active and cursor advance are transactional and failed mutations
  preserve the prior state; DELETE journal mode and owner-only directory/database
  and transient sidecar behavior are checked where deterministic;
- bridge v2 tests cover persistent reopen, authoritative refreshed responses,
  bounded typed storage errors and rejection of client-owned groups, cursor and
  raw database path fields;
- Swift store tests cover loading, restored empty/non-empty state, active state,
  exact backend selection order, visible storage failure, one-call authoritative
  add/toggle/advance refresh and absence of independent cursor/selection policy;
- source audits confirm no `UserDefaults`, `@AppStorage`, Swift SQLite/file
  persistence, completed-batch schema change, migration runner, Facebook/Safari,
  Phase 11, scheduler, retry, worker, cloud or network behavior.

Automated repository, bridge and Swift tests verify reopen persistence for groups,
active state, insertion order and the exact cursor. Phase 9E3b user-guided manual
acceptance verified one enrolled group after a full quit/relaunch, then verified
the same group and its inactive state after another full quit/relaunch. This
satisfies the previously deferred Phase 9E2 relaunch-persistence evidence; the
earlier deferment was not a persistence failure.

## Historical group discovery contract correction checks

Phase 9E2b keeps the Phase 9E2 persistence/bridge tests and adds focused Swift
checks that:

- persisted groups, active state, insufficient/exact-five preview order and
  storage-failure presentation remain authoritative;
- startup/render requests only `watched_groups_list`, so preview does not
  invoke `watched_groups_next_five` or advance the cursor;
- the primary Groups view contains no manual Add Group entry point and no
  `Chuyển lượt chọn` action;
- the future `Đồng bộ nhóm đã tham gia` action is visibly disabled and described
  as unavailable, with no fake discovery request, loading animation or data;
- the next-five section is explicitly a read-only preview for a future Scan
  batch, while active/inactive toggles remain functional.

Go regressions confirm schema v1, source-neutral group identity/state,
transactional cursor persistence and existing add/next-five bridge primitives
remain unchanged. Source audits confirm no discovery implementation,
Facebook/Safari change, account/session/browser storage or scan execution.

User-guided verification of the freshly built corrected UI passed: the Groups
screen opened; `+ Thêm nhóm` and `Chuyển lượt chọn` were absent; disabled
`Đồng bộ nhóm đã tham gia` was visible and unavailable; the empty state stated
that discovery is not implemented; and `Next 5 Groups` remained visible and
read-only. The UI did not imply manual population or cursor advancement, and no
discovery, Facebook, Safari or browser action occurred. This closeout does not
relaunch the app.

Phase 9E3b supersedes the Phase 9E2b product assumption and its primary-screen
expectations above. They remain documented only as historical verification of
the UI that existed at the Phase 9E2b closeout.

## Approved-group enrollment correction checks

Phase 9E3b adds focused Swift store/UI checks that:

- the empty state explains that no group has been enrolled, exposes `Thêm nhóm
  theo dõi` and says each group is saved locally and only added once;
- the enrollment sheet uses the existing `WatchedGroupsStore.addGroup` path,
  and a successful call replaces presentation state with the authoritative Go
  response rather than inventing local state;
- persisted groups still load on startup, active toggles still consume the
  authoritative response, and storage failures remain explicit;
- `Đồng bộ nhóm đã tham gia`, `Chuyển lượt chọn` and any other manual cursor
  action are absent from the primary UI;
- next-five remains a read-only preview derived from the list response, and
  viewing/rendering it never invokes `watched_groups_next_five` or advances the
  cursor.

Existing Go persistence/bridge tests continue to cover durable Add, reopen,
active state, insertion order and exact cursor. Source audits must confirm no
new discovery operation, Facebook/Safari runtime, selector, schema, Swift
persistence, cursor advancement or Phase 11/12 execution.

User-guided manual acceptance passed. The fresh Phase 9E3b bundle launched and
opened Groups; `Thêm nhóm theo dõi` and the one-time local-enrollment empty state
were visible, while automatic joined-group sync and manual queue advancement were
absent. One real Facebook group was enrolled through the supported UI and appeared
active. It remained after a full quit/relaunch; after the user changed it inactive,
both the group and its OFF state remained after another full quit/relaunch. Next 5
remained read-only. No Facebook Scan, Safari acquisition, discovery, browser
mutation or cursor advancement occurred. Private group identity is intentionally
not recorded.

## Joined-groups reconnaissance checks

Phase 9E3a used exactly one user-guided call to the committed
`AcquireSafariActiveTabRenderedDOM()` API against one user-prepared Facebook
page visibly listing joined groups. The temporary analyzer kept the 3,208,287
rendered-DOM bytes only in memory and persisted only a small redacted JSON
report under `/tmp` with mode `0600`; no private value or raw DOM entered the
repository or stdout.

The retained evidence contains one tentative semantic section, zero semantic
group-item containers, one canonical group-link/name association, zero
explicit machine-readable joined-membership candidates, one ambiguous group
link and one traversal candidate. Identity and display-name association are
STRONG; section and ordering are TENTATIVE; item and joined-versus-recommended
distinction are NOT_FOUND. The expected result is STOP/INCONCLUSIVE, with no
production discovery code, retry or second acquisition.

## Test matrix toi thieu

| ID | Fixture | Expected |
| --- | --- | --- |
| T01 | Nguoi that can mua MacBook tai HCM | Include buyer lead, `included.buyer_intent`, `included.target_keyword`, `included.location_hcm` |
| T02 | Bai ban co cau "can tien ban MacBook" | Exclude, `excluded.seller_intent` |
| T03 | Shop "can thu mua MacBook" | Review hoac exclude voi `excluded.dealer_or_shop` theo tin hieu shop, khong mac dinh include lead khach ca nhan |
| T04 | Anonymous ro rang | Exclude, `excluded.anonymous_author` |
| T05 | Author `motivatedsalamander3113` | Exclude, `excluded.author_name_has_no_space` |
| T06 | Ten mot tu bi loai theo product policy | Exclude, `excluded.author_name_has_no_space` |
| T07 | Account trong blocklist | Exclude, `excluded.blocked_author` |
| T08 | Mot nguoi dang cung bai o 5 group | Mot lead, 5 `LeadSource`, hien thi so group da dang la 5 |
| T09 | Mot nguoi co hai nhu cau khac nhau | Hai lead rieng |
| T10 | Bai hom qua noi lai vi comment hom nay | Exclude, `excluded.previous_day` |
| T11 | Bai trong ngay truoc thoi diem Scan | Duoc xet neu qua cac rule khac |
| T12 | Bai sau thoi diem bat dau Scan | Exclude hoac skip theo time window, khong include |
| T13 | Scan sau ngay moi khong backfill hom truoc | Khong doc lai ngay truoc, khong tao lead moi tu bai hom truoc |
| T14 | Supported HCM variants: HCM, TPHCM, TP.HCM, Ho Chi Minh, Sai Gon, Saigon | Classify HCM |
| T15 | District/ward/county-only location text | Review, `review.unknown_location`; district/ward/county recognition deferred |
| T16 | Supported outside-HCM Vietnam variants: Hà Nội, Ha Noi, Đà Nẵng, Da Nang, Cần Thơ, Can Tho | Classify Vietnam outside HCM; duoc xet neu qua rule khac |
| T17 | Unmatched or foreign-looking geographic text | Review, `review.unknown_location`; foreign classification deferred |
| T18 | Khong ro dia diem | Review, `review.unknown_location` |
| T19 | Group loi giua batch | Attempt `failed`, batch summary khong danh dau thanh cong gia |
| T20 | Batch di qua ranh gioi 00:00 | Group chua hoan tat `expired_at_day_boundary` |
| T21 | Duplicate source URL | Khong tao duplicate `LeadSource` vo nghia |
| T22 | Cung noi dung nhung khac tac gia | Khong gop chi vi noi dung giong |
| T23 | Dry Run giu bai bi loai | RawPost va FilterDecision duoc luu trong tab Da loai |
| T24 | Reason codes on dinh va deterministic | Cung fixture tao cung ordered reason codes |

## Test bo sung cho SearchProfile

- MacBook profile nhan dung `targetKeywords` va `keywordAliases`.
- Bai co buyer intent nhung khong chua target product cua SearchProfile bi loai voi `excluded.target_keyword_missing`.
- Bai seller intent chua target product bi loai voi `excluded.seller_intent`; reason nay khong tao seller mode.
- Rule thoi gian, dia ly, author, blocklist va dedup dung chung cho SearchProfile, khong phu thuoc MacBook.
- Product behavior khong co seller mode, seller tab, seller mode selector, `SellerLead`, `SellerIntentClassifier`, `LeadIntent` buyer/seller hoac `SELLER_SCAN`.
- SearchProfile khac chi la future architecture note, khong phai MVP acceptance criterion.

## Test types theo milestone

- Phase 1: test harness co the chay fixture rong va bao fail ro.
- Phase 2: model invariants va serialization fixtures.
- Phase 3A: time eligibility, anonymous author va no-space author rule fixtures.
- Phase 3B: product term, buyer-intent term, seller/noise term va composition fixtures.
- Phase 3C: finite MVP geographic vocabulary, unknown, conflict, `GeographicMode` va composition fixtures.
- Phase 3: normalization fixtures.
- Phase 4: BuyerIntentClassifier, target keyword, author va blocklist rules.
- Phase 4A: dedup author key, need key, duplicate comparison va fail-closed identity fixtures.
- Phase 4B: in-memory lead aggregation, source preservation, unaggregated/conflict va deterministic ordering fixtures.
- Phase 4C: blocklist identity key normalization, strongest author identity selection, in-memory list construction, duplicate entry handling, exact same-kind matching, fail-closed insufficient identity va dependency boundary fixtures.
- Phase 4D: application-layer in-memory lead filtering qua blocklist, allowed/blocked/unresolved separation, source preservation, strongest-identity no-fallback, defensive-copy va dependency boundary fixtures.
- Phase 5A: application-layer in-memory evaluation pipeline tu already-collected `RawPost` values qua rules, eligible selection, dedup aggregation va blocklist filtering; tests cover mixed rule outputs, aggregation/unaggregated/conflict preservation, allowed/blocked/unresolved preservation, ordering, source preservation, defensive copies va invalid configuration.
- Phase 5B: application-layer in-memory scan batch model cho mot den nam explicit groups; tests cover group count validation, group identity, post/group consistency, deterministic flattening, single-run pipeline output preservation, batch/per-group count summaries, source preservation, defensive copies va fail-closed behavior.
- Phase 5C: persistence-facing completed batch snapshot contracts; tests cover opaque record ID validation, deterministic conversion tu `ScanBatchInput`/`ScanBatchResult`, source/outcome/reason preservation, summary consistency, malformed snapshot rejection, defensive copies, save-only repository interface va dependency boundary. Khong test SQLite, filesystem I/O, migrations, load/list behavior hoac ID generation.
- Phase 5D: deterministic in-memory `BatchRepository` adapter; tests cover constructor/zero-value use, valid saves, duplicate ID rejection, insertion order, concrete read helpers, validation failure state preservation, defensive stored snapshots, source/outcome preservation va determinism. Khong test durable storage, SQLite, filesystem I/O, SQL, JSON persistence, migrations, goroutines hoac expanded repository interface.
- Phase 5E: thin run-and-save orchestration; tests cover nil repository, caller-supplied record ID validation, application failure without save, save-once behavior, duplicate ID propagation, saved/returned record preservation va dependency boundary. Khong test UI/CLI wiring, Facebook collection, durable storage, generated IDs, retries, concurrency hoac repository load/list behavior.
- Phase 5F: documentation-only durable SQLite schema design; Phase 5F itself had no SQLite runtime tests. Future durable save/load tests must use temporary databases and deterministic fixtures for foreign keys, unique constraints, explicit ordering, transaction rollback, duplicate `BatchRecordID`, schema-version rejection, sequential migrations and fail-closed reconstruction into `BatchRecord`.
- Phase 5G1: SQLite schema-bootstrap foundation tests use temporary databases for explicit-path open/create, foreign-key enable/verify, complete empty schema version 1, metadata validation, representative foreign-key/unique/check/not-null constraints, initialization rollback, deterministic schema inventory, close behavior, invalid paths va dependency boundary. Khong test durable `SaveBatch`, load/list APIs, migrations, production DB path policy, UI/CLI wiring, Facebook collection, generated IDs hoac concurrency.
- Phase 5G2: SQLite `SaveBatch` tests use temporary databases for interface satisfaction, valid one-group/five-group saves, complete rich snapshot table mapping, raw source preservation, explicit positions, decisions/outcomes/evidence/reasons, validation-before-mutation failures, duplicate `BatchRecordID` no-overwrite, induced child-write rollback, safe save-after-close behavior, immutability, determinism va absence of list/update/delete/search APIs. Khong test load reconstruction, migrations, production DB path policy, UI/CLI wiring, Facebook collection, generated IDs, retries hoac concurrency.
- Phase 5G3: SQLite concrete `LoadBatch` tests use temporary databases for rich one-group va five-group round trips, exact accessor equality, scan window/profile/geographic mode/summary preservation, raw-post occurrence order, Vietnamese body preservation, evaluated/bucket reason order, lead/source/outcome/unaggregated/conflict reconstruction, not-found/empty-ID/closed/nil lifecycle behavior, no partial result, defensive reload determinism, save-only `BatchRepository`, va fail-closed corruption of schema, timestamps, booleans, enum-like values, missing references, duplicate/gapped positions, summaries, metadata va required tables/indexes. Khong test list/update/delete/search APIs, migrations, production DB path policy, UI/CLI wiring, Facebook collection, generated IDs, retries hoac concurrency.
- Phase 5: geographic classifier hardening/regression fixtures neu can.
- Phase 6: deduplication va lead aggregation fixtures.
- Phase 7: persistence repository tests voi database local tam.
- Phase 8A: documentation-only native macOS UI architecture decision; checks cover SwiftUI-as-presentation-shell wording, future `macos/ScanFBApp/` location, no existing app code/project, no bridge selected, no business logic in Swift, fixture/privacy policy, stale-process manual validation policy, and Go regression verification.
- Phase 8B: empty native SwiftUI app shell tests must include Swift unit tests for six-section navigation order/labels/icons/default selection, Xcode build, manual app launch, stale process termination before relaunch, verification that the rebuilt app bundle is running, and Go `go test ./...`, `go vet ./...`, CLI build/run output unchanged.
- Phase 8C: static Overview fixture tests cover typed fixture integrity, coherent counts, stable exclusion reason order, sample/demo labeling, Xcode build/test, manual Overview validation, stale-process kill/relaunch, rebuilt-bundle verification and Go regression checks.
- Phase 8D: fixture Leads tests cover typed fixture integrity, exact lead IDs,
  category distribution, stable tab labels/counts, deterministic filtering,
  fixed dates/source counts, repository-existing reason-code display strings,
  no fixture links, Xcode build/test, manual Leads validation, disabled
  placeholder actions, stale-process kill/relaunch, rebuilt-bundle verification
  and Go regression checks.
- Phase 8E: fixture Dry Run tests cover typed fixture integrity, exact post IDs,
  category distribution, stable tab labels/counts, deterministic filtering,
  fixed dates, repository-existing reason-code display strings, no fixture
  links, Xcode build/test, manual Dry Run validation, disabled placeholder
  action, stale-process kill/relaunch, rebuilt-bundle verification and Go
  regression checks.
- Phase 8F: fixture Blocklist and Settings tests cover typed fixture integrity,
  exact blocklist IDs, supported stable identity kinds, no display-name-only
  block identity, fixed dates, `.invalid` synthetic URL display, no real
  Facebook domain, four read-only settings sections, required settings rows,
  no writable setting state, Xcode build/test and no direct SQLite/networking
  or package additions.
- Phase 8G/8H: fixture Leads interaction-state tests cover typed state integrity,
  exact states `new/viewed/ignored`, fresh-session reset, selected lead mutation
  only, unchanged eligibility tabs, categories, reason codes and fixture order,
  stateless `Tương tác`, deterministic source URL fixtures, URL validation,
  injected browser handoff, Xcode build/test and no persistence, direct SQLite,
  AppStorage/UserDefaults, bridge, networking client, WebKit, Facebook
  integration, timestamps or package additions.
- Phase 8I.1: documentation-only bridge decision checks cover
  [BRIDGE_DECISION.md](BRIDGE_DECISION.md), selected local subprocess
  request/response model, rejected IPC/in-process/HTTP/direct-SQLite/browser
  alternatives, schema/error/cancellation principles, first read-only future
  slice and audits that no Go, Swift, Xcode project, dependency, generated code
  or runtime bridge changed.
- Phase 8I.2: first read-only bridge slice tests cover Go readiness
  request/response parsing, exact schema version, exact `core_readiness`
  operation, deterministic response, exact `scanfb-core` identity, malformed
  request rejection, unsupported schema/operation rejection, stdout-only machine
  response, no Facebook/database/persistence access in readiness path, Swift
  deterministic request encoding, response decoding, exact status enum mapping,
  explicit nonzero/timeout/cancellation/helper-missing/malformed-response
  errors, raw stderr not surfaced to UI, no mutation of other settings fixture
  rows, no auto-run on view creation, one subprocess helper round-trip, Xcode
  test/build and Go regression checks.
- Phase 8I.2a: Debug helper packaging tests/checks cover deterministic helper
  name `scanfb-bridge-helper`, bundle-relative lookup at
  `Contents/Helpers/scanfb-bridge-helper`, fail-closed missing helper behavior,
  no PATH or `/tmp` helper fallback, Debug Xcode build packaging, executable bit,
  direct bundled-helper readiness invocation, stdout/stderr separation and no
  protocol/schema changes.
- Phase 8I.3+: later bridge implementation milestones must add
  bridge-specific tests for each narrow slice while retaining Xcode build/manual
  launch when UI changes, stale-process kill/relaunch when launching is in
  scope, rebuilt-bundle verification and Go regression checks.
- Phase 9A: Go-only in-memory lifecycle state machine tests cover production batch exactly five groups, caller-supplied unique attempt IDs, group order preservation, all attempts initially `pending`, one-running-at-a-time, no skip-ahead, valid transitions `pending -> running`, `running -> succeeded`, `running -> failed`, `pending -> skipped`, `pending/running -> expired_at_day_boundary`, terminal attempts refusing further transitions, count-only summary reconciliation to five, explicit next action after failure, no automatic retry, defensive copies, invalid transition no partial mutation, no internal `time.Now`, and exact `Asia/Ho_Chi_Minh` day-boundary behavior.
- Phase 9A T19: a running group can fail deterministically, remains `failed`, never appears `succeeded`, does not auto retry and allows the next group only through an explicit next/start action.
- Phase 9A T20: when supplied time is after local midnight relative to batch scan date, existing succeeded/failed/skipped attempts stay unchanged, running and pending attempts become `expired_at_day_boundary`, the batch becomes terminal and no new attempt can start.
- Phase 9B: Go-only WatchedGroup tests cover facebookGroupId-backed and canonicalUrl-only identity, exact validation errors, facebookGroupId authority when both source identities exist, default active creation, optional metadata, impossible `lastSuccessfulScanAt` chronology, duplicate local/authoritative identity rejection without overwrite, lookup/not-found behavior, idempotent activate/deactivate, identity/createdAt preservation, unlimited collection size, insertion-order inspection, `displayOrder` as metadata only, defensive snapshots, invalid operation no partial mutation, no internal clock/generated ID, no five-group selection and no Phase 9A lifecycle invocation.
- Phase 9C: Go-only selector tests cover explicit initial/bounded collection-position cursor, exact-five active selection, insertion-order continuation, wrap-around, inactive skipping without position compaction, insufficient/all-inactive/empty fail-closed cases, no partial result, duplicate prevention, deterministic repetition, collection/input immutability, defensive selected slices, `displayOrder`/`createdAt`/`lastSuccessfulScanAt` neutrality, deactivate/reactivate behavior, source-identity neutrality, no lifecycle/ID generation, no internal clock, no scan execution and no infrastructure imports.
- Phase 9D: Go-only mapper tests cover exact `FiveGroupSelection` order, caller-supplied batch/attempt IDs and `ScanWindow`, five pending lifecycle attempts, exact attempt-ID count, Phase 9A normalization/error propagation, malformed selection/identity rejection, input/cursor immutability, deterministic repetition, defensive snapshots, no re-selection, no lifecycle transition, no generated ID, no internal clock, no scan execution and no infrastructure imports.
- Phase 9E1: Go bridge and Swift store tests cover authoritative Phase 9B add/active behavior, exact Phase 9C selection order, empty/insufficient/exact/more-than-five active groups, inactive handling, explicit cursor advancement, deterministic typed schemas, session-only ownership and no independent Swift selection logic.
- Phase 9E2a: docs-only checks cover all location/schema candidates, one APPROVE outcome, exact Application Support path shape, Go/helper ownership, separate state schema v1, unchanged completed-batch v1, full WatchedGroup field inventory, cursor transaction/restore/corruption policy, temp-path test override and absence of runtime/schema/bridge/Swift persistence changes.
- Phase 9E2: implemented Go/bridge/Swift tests cover dedicated temporary SQLite state bootstrap, exact full-value/insertion-order restore, authoritative identity conflicts, active metadata, exact cursor persistence/reopen behavior, atomic failures, typed bridge storage errors and Swift authoritative refresh. Phase 9E3b manual acceptance satisfied the remaining quit/relaunch check through the supported enrollment flow for both group membership and active/inactive state.
- Phase 9E2b: historical Swift/source tests covered hidden manual add/queue advance, disabled unavailable joined-group synchronization copy, read-only next-five preview and list-only rendering; Phase 9E3b supersedes those primary-screen expectations without changing schema or authority.
- Phase 9E3a: docs-only one-page reconnaissance records bounded redacted counts and exact confidence after one read-only rendered-DOM acquisition. It accepts canonical group-link and same-link name evidence but rejects broad `/groups/` scraping, localized text, generated classes, DOM depth and positional selectors; missing item/membership distinction keeps Phase 9E3 blocked.
- Phase 9E3b: Swift/store tests cover visible one-time enrollment and accurate empty-state copy, absent automatic-sync/manual-advance controls, existing backend-authoritative Add response, startup restore, active toggle, explicit persistence failures and read-only preview without cursor advancement. Existing Go bridge/persistence regressions passed; user-guided acceptance verified enrolled-group and inactive-state persistence across separate full quit/relaunch cycles.
- Phase 9E+: later persistence, adapter and production scan orchestration tests must be defined by separate narrow milestones.
- Phase 10A: Go-only prepared-page tests cover typed local fixture schema, exact ordered `RawPost` mapping, caller group/name/capture propagation, Vietnamese body preservation, absolute RFC3339 timestamps, optional absolute HTTPS post URLs, supplied stable/display-only author identity, missing-field fail-closed errors, group conflict rejection, determinism, input immutability, no fabricated optional data, no live DOM/browser/network/session/cookie/credential access, no scan/lifecycle execution and no persistence/UI/bridge behavior.
- Phase 10B1: Go-only Safari active-tab acquisition tests cover strict deterministic response parsing, exact URL/title/content and caller timestamp preservation, optional empty title, absolute HTTPS/no-userinfo URL validation, empty/oversized content rejection with a 4 MiB decoded-content bound and finite worst-case JSON transport envelope, direct `/usr/bin/osascript` JXA arguments, front-window current-tab-only reads, bounded separate stdout/stderr, explicit start/nonzero/TCC/not-running/no-window/no-tab/timeout/cancellation errors, result fields without browser secrets, and source audits excluding Chrome, Accessibility/UI scripting, WebKit/extension, network/listener, browser profile stores, `RawPost`/Phase 10A automatic extraction, Phase 11 lifecycle/pipeline, persistence, SQLite, Swift, Xcode and bridge behavior. User-guided manual validation acquired one expected active group page successfully without printing raw HTML; its approximately 1.5-1.6 MB source remained below the 4 MiB bound.
- Phase 10B2a: docs-only one-page reconnaissance records redacted structural counts and confidence without storing live HTML. It confirms page identity from the validated active URL but finds no stable post container, permalink/ID, body, author, absolute timestamp or visible-order evidence in `tab.source()`; generic bootstrap markers are rejected and production selector work stops fail closed.
- Phase 10B2b: production selector implementation remains incomplete and blocked until an approved rendered-DOM acquisition is implemented, manually validated and followed by separate redacted structural evidence.
- Phase 10B2c: docs-only decision checks cover all seven candidate mechanisms, official Apple/Safari sources, rendered-DOM capability, user/developer/TCC permissions, session/privacy and mutation implications, finite-output/process isolation, packaging/signing/App Sandbox impact, fixture-free testability, exact rejection reasons and selection of only fixed read-only Safari Apple Events page-side JavaScript for a future acquisition-only slice. Audits confirm no Go, Swift, Xcode, bridge, entitlement, Safari execution, extension, Accessibility, WebKit, listener, selector, `RawPost` or Phase 11 change.
- Phase 10B2d: Go-only rendered-DOM acquisition tests cover exact URL/title/rendered-document/caller timestamp preservation, deterministic strict parsing, malformed/unsupported/empty/oversized rejection, 8 MiB decoded boundary, 50,397,184-byte worst-case JSON envelope, 16 KiB stderr cap, 5-second timeout, direct `/usr/bin/osascript`, fixed current-tab JXA and fixed outerHTML expression, no caller JavaScript input, Safari/TCC/developer-setting/process/timeout/cancellation errors, bounded stdout/stderr separation, zero snapshot on failure and production source/script audits excluding selectors, `RawPost`, Phase 10A/11, mutation/navigation/scroll/click/focus, cookie/storage/session/profile access, network/listener, Accessibility/System Events, extension/WebKit, persistence, SwiftUI and bridge behavior. Tests use synthetic HTML only. User-guided live validation succeeded on one expected active Facebook group tab without printing raw DOM.
- Phase 10B2e: exactly one production rendered-DOM acquisition and in-memory analyzer run passed on the user-confirmed Facebook group tab, with no raw DOM output or repository fixture. The command-output filter did not retain the redacted structural report, so only group identity is STRONG and every post-level concept remains NOT FOUND in retained evidence. Regression checks must confirm no selector, `RawPost`, Phase 10A/11, private artifact, browser mutation, alternate mechanism, dependency, persistence, SQLite, SwiftUI, Xcode or bridge change. No automatic second acquisition is allowed.
- Phase 10B2f: Go-only pure analyzer tests cover semantic `article`/`role=article` candidates, redacted group-post/profile href shapes, body-preview attributes, machine `time[datetime]`/`data-utime`, optional group-page URL shape, deterministic traversal count, complete/partial/missing coverage, `STRONG`/`TENTATIVE`/`NOT_FOUND`, CSS/depth/nth-child rejection, relative-time non-promotion, exact private body/author/post/group/full-URL non-disclosure, max 16 marker strings, max 64 bytes per marker, sorted deterministic arrays, repeated identical typed/JSON output, empty/oversized fail-closed input and source audits excluding acquisition, browser, filesystem, network, clock, selector, `RawPost`, Phase 10A/11, persistence, SQLite, SwiftUI, Xcode and bridge behavior. Fixtures are synthetic only; no live acquisition belongs to Phase 10B2f automated tests.
- Phase 10B2g: docs-only closeout records one preserved live typed report with 3,180,722 bytes, two `role=article` candidates, traversal count two, valid group URL shape and zero permalink/body/author/machine-time/complete-evidence candidates. Documentation checks must preserve STRONG container/group/traversal confidence, NOT_FOUND critical-field confidence, the rejected unstable-marker list and Phase 10B2b BLOCKED status without adding selector, `RawPost`, Phase 11, code/config/dependency or private fixture changes.
- Phase 11A: Go-only tests cover one active enrolled group, caller-supplied scan/attempt IDs and times, exact group/ScanWindow collector request, exactly one collector call, ordered and defensive `RawPost` handling, existing Phase 5B rules/dedup/blocklist processing, succeeded result, T19-style failed result without retry/fake success, wrapper/post group mismatch, invalid request/application configuration, explicit zero-post success, Phase 9A day-boundary expiration, cancellation and deterministic repetition. Source audits exclude Phase 9C cursor/selection, persistence/save/schema, Facebook/Safari/browser, network, generated IDs, hidden `time.Now`, goroutines and background work.
- Phase 11B0: docs-only checks cover candidates A-H, exact `RawPost` field coverage, timestamp and identity quality, user friction, privacy/security, packaging, fixture-free testability, official Meta/Apple sources, the DEFER outcome and the requirement for materially new source evidence. Audits confirm no collector, selector, importer, API/network client, extension, WebKit/Accessibility, browser runtime, persistence, bridge/UI, Phase 11A or cursor change.
- Phase 11C0: docs-only checks cover local file, ScanFB-owned form, clipboard, CSV/JSON import, external helper and other local candidates; exact field/provenance/friction/privacy comparison; APPROVE outcome; one form action; prepared-snapshot JSON v1 ownership; authoritative group/capture rules; explicit absolute `CreatedAt`; 1-100 ordered posts; 1 MiB aggregate bound; whole-payload failure; Phase 10A/11A replaceable integration; and Phase 11C1 scope. Audits confirm no importer/collector, Swift/UI, bridge, file/clipboard, browser/Safari, persistence/schema, Phase 11A, batch-five or cursor change.
- Phase 11C1: Go-only tests cover interface satisfaction, valid one/multiple/100-post JSON v1, exact order and value preservation, authoritative request group/name, caller ScanWindow capture time, exact 1 MiB acceptance, oversize/zero/101 rejection, body/display-name/username/Facebook-ID/profile-URL/post-URL/post-ID byte boundaries, display-only and stable-ID-only authors, missing/whitespace author rejection, exact `+07:00` RFC3339/RFC3339Nano acceptance, UTC/other-offset/timezone-less/relative/malformed time rejection, malformed/types/unknown top-level/post/author fields, duplicate keys at any object depth, trailing/multiple values, Phase 10A body/HTTPS/no-userinfo reuse, all-or-nothing zero result, constructor/result defensive ownership, deterministic repetition, cancellation and active-request validation. Phase 11A integration proves valid success and invalid-payload T19-style failure. Source audits exclude file/stdin/clipboard, browser/Safari/network, persistence, cursor, generated IDs, `time.Now`, retry, scheduler, concurrency and Phase 12.
- Phase 11C2: Go bridge/helper tests cover the exact `prepared_group_scan` operation, finite 1,114,112-byte request and 16 KiB response bounds, persistent group lookup, active/missing/inactive handling, authoritative group/profile/default-`hcm` request construction, one Phase 11A invocation, success count mapping, invalid-payload fail-closed behavior, generic redacted diagnostics, unchanged WatchedGroup state/cursor and no retry. Swift tests cover active-group availability, one initial row, add/remove/100-row cap, empty body/author validation, exact `+07:00` serialization, row order, fixed nested schema v1, optional post identity/URL, absence of group/capture fields inside the prepared snapshot, exactly one bridge request, count-only success, visible failure and source audits excluding cursor, file/clipboard and Swift persistence APIs. Full Go/macOS regression and build verification precede separately guided manual acceptance; automated tests use synthetic data only.
- Phase 11C2 manual acceptance passed on a fresh built bundle: existing enrollment survived launch; inactive state disabled `Nhập dữ liệu quét` with activation guidance; activation enabled entry; the form showed authoritative group, one initial row, required body/display-name/time, optional post identity fields, collapsed additional author identity, add action, `1/100` counter and session-only notice. Empty submit produced exact `Bài 1 chưa có nội dung.` validation without fake result or crash. One synthetic post without optional Post URL/Post ID completed the full Swift form -> typed bridge -> `PreparedSnapshotCollector` -> Phase 10A -> Phase 11A -> evaluation pipeline path; UI showed `Quét hoàn tất` with collected 1, evaluated 1, included 0, review 1, excluded 0 and allowed lead 0. No five-group execution or visible queue advancement occurred. Full quit/relaunch preserved the enrolled group but restored one fresh empty row and no prior input/result, confirming session-only behavior. The non-blocking stale-validation observation is covered by Phase 11C2a.
- Phase 11C2a: focused Swift tests cover empty-body and empty-author presentation errors, body-edit recomputation to the still-relevant author error, author-edit clearing only the resolved error, valid local form state, unchanged explicit valid submit behavior, preservation of bridge-originated errors, and zero additional bridge request or scan caused by editing. The macOS Debug build verifies integration. Manual acceptance passed after explicit ScanFB termination and fresh relaunch of that bundle: empty submit showed `Bài 1 chưa có nội dung.`, and entering body text without another submit immediately replaced it with the still-relevant `Bài 1 chưa có tác giả.` error. Editing triggered no scan or bridge request. Implementation, focused automated verification, build verification and manual acceptance are complete; Go validation authority and scan/bridge semantics are unchanged. Automated tests and the build were not rerun during this docs-only closeout.
- Phase 11B: production one-group collector remains blocked until a separately approved reliable source contract proves stable post identity, body, author, machine timestamp and group binding. Phase 11A tests use only injected fakes and synthetic normalized `RawPost` fixtures.
