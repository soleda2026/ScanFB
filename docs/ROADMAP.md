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
