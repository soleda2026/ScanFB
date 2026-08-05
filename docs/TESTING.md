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
- Phase 8B: empty native SwiftUI app shell tests must include Xcode build, manual app launch, stale process termination before relaunch, verification that the rebuilt app bundle is running, and Go `go test ./...`, `go vet ./...`, CLI build/run output unchanged.
- Phase 8C-8G: fixture UI milestones must include deterministic fixture rendering/manual validation, sample/demo labeling, Vietnamese diacritic preservation, no live Facebook data claims, no direct SQLite access, no hidden networking, no seller mode, stale-process kill/relaunch, rebuilt-bundle verification and Go regression checks.
- Phase 8H: bridge evaluation tests are documentation/prototype checks only if explicitly scoped; bridge selection must not skip deterministic request/response, explicit schema, cancellation, error propagation, packaging, crash isolation and Apple Silicon considerations.
- Phase 8I+: post-bridge integration milestones must add bridge-specific tests for each narrow slice while retaining Xcode build/manual launch, stale-process kill/relaunch, rebuilt-bundle verification and Go regression checks.
- Phase 9: batch state machine tests.
- Phase 10 tro di: manual validation co kiem soat cho Facebook adapter.
