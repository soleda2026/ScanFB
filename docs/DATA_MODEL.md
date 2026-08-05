# Data Model

Tai lieu nay dinh nghia entity toi thieu va quan he muc domain. Kieu du lieu cu the se duoc khoa trong milestone sau. ScanFB hien tai la buyer-only: khong co `LeadIntent` buyer/seller, `SellerLead` hoac seller mode.

Phase 2 Go implementation chi trien khai subset domain toi thieu cho normalized post input va cau hinh mot scan: `RawPost`, `AuthorIdentity`, `SearchProfile`, `GeographicMode`, `ScanWindow` va `ScanRequest`. Phase 4C Go implementation chi them in-memory blocklist identity primitives trong `internal/blocklist`. Phase 4D Go implementation chi them application-layer in-memory filtering cua aggregated leads qua blocklist. Phase 5A Go implementation them application-layer deterministic pipeline cho already-collected `RawPost` values qua rules, eligible selection, in-memory aggregation va blocklist filtering. Phase 5B Go implementation them application-layer in-memory batch model cho mot den nam explicit groups, deterministic flattening va count summaries. Phase 5C Go implementation them persistence-facing completed `BatchRecord` snapshot contract va save-only repository interface. Phase 5D Go implementation them `InMemoryBatchRepository` adapter chi trong memory. Phase 5E Go implementation them thin `internal/orchestration` use case de run completed batch, convert thanh `BatchRecord` va save qua repository contract. Phase 5F them durable SQLite schema design tai [PERSISTENCE_SCHEMA.md](PERSISTENCE_SCHEMA.md). Phase 5G1 them SQLite schema-bootstrap foundation trong `internal/persistence`. Phase 5G2 them transactional SQLite `SaveBatch` cho completed `BatchRecord`; migration execution, load/list API tren interface, UI/CLI wiring va ID generation van chua co.

## WatchedGroup

Purpose: Luu group Facebook nguoi dung chu dong theo doi.

Required fields: `id`, `facebookGroupId` hoac `canonicalUrl`, `name`, `createdAt`, `isActive`.

Optional fields: `notes`, `lastSuccessfulScanAt`, `lastError`, `displayOrder`.

Identity/key: `facebookGroupId` neu co, nguoc lai `canonicalUrl`.

Lifecycle: Tao khi nguoi dung them group, cap nhat khi doi ten/URL, vo hieu hoa khi nguoi dung xoa khoi danh sach theo doi.

Invariants:

- Chi group nguoi dung chu dong them moi duoc scan.
- Khong gioi han tong so group.
- Mot batch chi lay dung 5 group.

## ScanSession

Purpose: Dai dien mot lan nguoi dung bam Scan.

Required fields: `id`, `startedAt`, `scanDate`, `timezone`, `geographicMode`, `searchProfileId`, `status`.

Optional fields: `completedAt`, `lastSuccessfulScanAtUsed`, `summary`.

Identity/key: `id`.

Lifecycle: Tao khi nguoi dung bam Scan, ket thuc khi batch tra ket qua hoac fail-closed.

Invariants:

- `timezone` la `Asia/Ho_Chi_Minh`.
- `startedAt` la upper bound cua post time.
- Khong lay bai sau `startedAt`.
- `searchProfileId` trong MVP tro den built-in MacBook Search Profile.

## ScanBatch

Purpose: Nhom dung 5 group trong mot scan session.

Required fields: `id`, `scanSessionId`, `groupIds`, `status`, `createdAt`.

Optional fields: `completedAt`, `summary`.

Identity/key: `id`.

Lifecycle: Tao tu queue group, chay tung group, tra ket qua va cho batch tiep theo.

Invariants:

- `groupIds` co dung 5 group.
- Neu khong con du 5 group active de tao batch tiep theo, app khong bat dau batch moi va bao ro cho nguoi dung.
- Khong mo nhieu group dong thoi.
- Neu qua 00:00, group chua hoan tat duoc ghi `expired_at_day_boundary`.
- Phase 5B implementation chi la in-memory model cho already-collected posts va chap nhan mot den nam explicit groups de test/summary deterministic; chua phai production batch queue, adapter hoac persistence behavior.

## BatchRecord

Purpose: Completed immutable snapshot cua mot `ScanBatch` da co application result, de persistence adapter co the save nguyen batch.

Phase 5C Go implementation nam trong `internal/persistence` va chi tao value contract: `BatchRecordID`, `BatchRecord`, structural `Validate()` va `BatchRepository.SaveBatch(record)`. Phase 5D them `InMemoryBatchRepository` de save snapshots trong memory, preserve insertion order va reject duplicate ID ma khong overwrite. Phase 5E orchestration dung caller-supplied `BatchRecordID`, tao completed `BatchRecord` tu successful application result va chi tra ve sau khi repository save thanh cong. Phase 5G2 them `SQLiteBatchRepository.SaveBatch` de persist complete snapshot vao SQLite schema version 1 trong mot transaction.

Required content: scan window, SearchProfile snapshot, GeographicMode, ordered groups, flattened posts, evaluated/included/review/excluded posts voi exact decisions/reasons, aggregated leads, source posts, unaggregated posts, source conflicts, blocklist outcomes/reasons, application reasons, batch summary va group summaries.

Invariants:

- `BatchRecordID` opaque, non-empty va duoc supplied tu future boundary; persistence khong generate ID.
- Snapshot chi dai dien completed batch; SQLite save la atomic, khong co partial save, update, delete, search, paging, migration hoac public transaction API.
- Validation chi structural, khong recompute rule, geography, dedup hoac blocklist decisions.
- Orchestration khong generate ID va khong mo rong `BatchRepository` ngoai `SaveBatch`.
- SQLite schema version 1 dung `BatchRecord` lam root aggregate, `BatchRecordID` van authoritative, moi ordered collection co position explicit, va moi source post reference mot preserved post occurrence thay vi rely vao `PostID`.
- Batched group IDs phai non-empty, unique va co 1 den 5 group trong Phase 5C contract.
- Count summaries phai khop preserved collections.
- In-memory adapter inspection methods (`Count`, `Records`, `RecordByID`) chi la concrete-adapter helpers va tra defensive snapshots; `BatchRepository` van save-only.
- `SQLiteBatchRepository` open/create explicit local SQLite path, enable foreign keys, initialize/validate schema metadata, close, va satisfy `BatchRepository` bang transactional `SaveBatch`; reconstruction/load/list van deferred.

## GroupScanAttempt

Purpose: Luu ket qua scan cua mot group trong batch.

Required fields: `id`, `scanBatchId`, `watchedGroupId`, `status`, `startedAt`.

Optional fields: `completedAt`, `errorCode`, `errorMessage`, `postsConsidered`, `keywordMatches`.

Identity/key: `id`.

Lifecycle: `pending` -> `running` -> `succeeded` hoac `failed`/`skipped`/`expired_at_day_boundary`.

Invariants:

- Khong danh dau thanh cong neu adapter gap loi fail-closed.
- Moi attempt thuoc dung mot batch.

## RawPost

Purpose: Ban ghi raw da doc tu Facebook truoc khi normalize va filter.

Phase 2 Go fields: `PostID`, `GroupID`, `GroupName`, `PostURL`, `Author`, `Body`, `CreatedAt`, `CapturedAt`.

Required fields: `id`, `sourcePostUrl`, `canonicalPostUrl`, `groupId`, `author`, `rawText`, `createdAt`, `capturedAt`.

Optional fields: `facebookPostId`, `permalink`, `rawTimestampText`, `attachmentsMetadata`, `readerMetadata`.

Identity/key: `facebookPostId` neu co, nguoc lai `canonicalPostUrl`.

Lifecycle: Tao boi Facebook adapter, luu local ke ca khi bi loai trong Dry Run.

Invariants:

- Khong chua secret, cookie hoac token.
- `createdAt` phai la thoi diem tao bai goc, khong phai thoi diem activity moi.

## AuthorIdentity

Purpose: Dinh danh nguoi dang bai.

Phase 2 Go fields: `FacebookUserID`, `CanonicalProfileURL`, `Username`, `DisplayName`. Khong co password, cookie, access token, session token, email, phone hoac profile image.

Required fields: `displayName`.

Optional fields: `facebookUserId`, `canonicalProfileUrl`, `username`, `profileUrl`, `isAnonymousSignal`.

Identity/key: Uu tien `facebookUserId`, roi `canonicalProfileUrl`, roi `username`.

Lifecycle: Gan vao RawPost, dung cho blocklist va dedup.

Invariants:

- Ten hien thi chi la thong tin phu, khong duoc dung lam khoa dedup duy nhat.
- Author anonymous hoac ten khong co dau cach bi loai theo [SCAN_RULES.md](SCAN_RULES.md).

## SearchProfile

Purpose: Dinh nghia cau hinh buyer search cho mot product/category muc tieu.

Phase 2 Go implementation co built-in `MacBookSearchProfile()` va term slices duoc copy khi nhan/tra de tranh mutation leak.

Required fields: `id`, `displayName`, `targetKeywords`, `keywordAliases`, `buyerIntentPhrases`, `exclusionPhrases`, `isEnabled`.

Optional fields: `productAttributeDefinitions`, `associatedWatchedGroupIds` hoac co che lien ket tuong duong, `description`.

Identity/key: `id`.

Lifecycle: MVP tao san built-in MacBook Search Profile va khong co UI tao profile tuy y. Sau MVP co the them buyer Search Profile khac bang milestone rieng.

Invariants:

- Moi SearchProfile cua ScanFB deu phuc vu buyer-intent.
- MVP chi enable MacBook Search Profile.
- Khong co profile marketplace danh cho seller.
- Khong co seller mode hoac `LeadIntent` buyer/seller.
- Rule chung ve time window, geographic scope, author, blocklist va dedup khong thuoc rieng MacBook.
- `productAttributeDefinitions` la tuy chon theo profile; chip/RAM/SSD la attribute rieng cua MacBook profile, khong phai invariant chung.

## FilterDecision

Purpose: Ket qua rule engine cho mot RawPost.

Required fields: `id`, `rawPostId`, `decision`, `reasons`, `createdAt`.

Optional fields: `score`, `confidence`, `reviewNotes`, `searchProfileId`.

Identity/key: `id`.

Lifecycle: Tao sau normalization/rule/geographic classification; co the duoc user override trong Dry Run review.

Invariants:

- Review/exclude decision phai preserve exact `FilterReason`; include decision preserve exact rule result va co the khong co reason trong Go implementation hien tai.
- Reason codes phai deterministic va machine-readable.
- Include decision trong ScanFB luon la buyer include cho SearchProfile dang bat.
- Phase 5A pipeline giu rule-evaluated, eligible, review va excluded posts explicit trong memory; chua ghi `FilterDecision` vao persistence.

## FilterReason

Purpose: Ma ly do giu, loai hoac can kiem tra.

Required fields: `code`, `message`.

Optional fields: `evidence`, `ruleVersion`.

Identity/key: `code` trong context decision.

Lifecycle: Tao boi rule engine hoac geographic classifier.

Invariants:

- `code` on dinh theo contract.
- `message` de hieu cho nguoi dung.

## GeographicClassification

Purpose: Ket qua phan loai dia ly.

Required fields: `mode`, `classification`, `confidence`, `reasons`.

Optional fields: `matchedTerms`, `normalizedLocation`, `rawLocationText`.

Identity/key: Theo `rawPostId` hoac `leadId` tuy context.

Lifecycle: Tao sau normalization, dung trong filter va lead display.

Invariants:

- Foreign-location classification va foreign exclusion deferred trong MVP.
- Geography khong match approved vocabulary trong [SCAN_RULES.md](SCAN_RULES.md) la unknown.
- Unknown geography vao Can kiem tra va khong duoc coi la foreign.
- Mac dinh mode moi ngay va moi lan mo app la TP.HCM.

## Lead

Purpose: Dai dien mot buyer lead sau dedup va aggregation.

Phase 4B Go implementation chi cung cap in-memory aggregation primitive trong `internal/dedup`: logical lead identity, stable author identity, deterministic need identity, full preserved source posts, explicit unaggregated posts va source conflicts. Chua co persistence schema, repository, database ID hoac workflow status implementation.

Phase 4D Go implementation them in-memory application filtering cho aggregated leads: allowed, blocked va unresolved collections explicit. Filtering dung stable author identity san co tren lead va local blocklist primitives; blocked leads khong bi xoa va sources van duoc preserve. Chua co persistence, scan orchestration, raw-post rule evaluation, UI hoac CLI behavior.

Phase 5A Go implementation ket noi already-collected `RawPost` values vao rules, in-memory aggregation va local blocklist filtering trong application layer. Ket qua explicit gom allowed, blocked, unresolved, unaggregated va source conflicts; chua co persistence schema, repository, Facebook adapter, UI hoac CLI behavior.

Required fields: `id`, `searchProfileId`, `authorIdentityKey`, `status`, `createdAt`, `updatedAt`, `firstPostCreatedAt`, `sources`.

Optional fields: `score`, `confidence`, `summary`, `productAttributes`, `budget`, `area`.

Identity/key: `id`, duoc suy ra tu author identity va fingerprint nhu cau.

Lifecycle: Tao khi post duoc include hoac restore tu Dry Run, cap nhat khi them source, doi status khi nguoi dung thao tac.

Invariants:

- Khong gop hai nhu cau khac nhau cua cung author.
- Mot lead co the co nhieu `LeadSource`.
- Lead trong ScanFB la buyer-only. Khong tao `SellerLead`.
- `productAttributes` phu thuoc SearchProfile. Voi MacBook profile co the gom model, chip, RAM, SSD.

## LeadSource

Purpose: Lien ket lead voi post/group goc.

Required fields: `id`, `leadId`, `rawPostId`, `groupId`, `canonicalPostUrl`, `postCreatedAt`.

Optional fields: `sourceExcerpt`, `matchedKeywords`, `searchProfileId`.

Identity/key: `canonicalPostUrl` hoac `rawPostId` trong context lead.

Lifecycle: Tao khi aggregation them source vao lead.

Invariants:

- Khong xoa source trung; duplicate source URL phai duoc nhan dien va khong tao source trung lap vo nghia.
- Phai giu URL goc de nguoi dung mo bai.

## BlockedAuthor

Purpose: Luu account nguoi dung bo qua.

Phase 4C Go implementation chi cung cap in-memory identity primitives: `Entry`, `IdentityKey`, `List` va deterministic author matching. Chua co persistence schema, database ID, timestamp, repository, scan filtering orchestration, UI hoac CLI behavior.

Required fields: `id`, `identityKind`, `identityValue`, `reason`, `createdAt`.

Optional fields: `displayName`, `note`, `unblockedAt`.

Identity/key: Cap `identityKind` va `identityValue`.

Lifecycle: Tao tu thao tac `Bo qua account nay`, co the bo block trong Settings.

Invariants:

- Uu tien identity theo Facebook user ID, canonical profile URL, username; display name chi la metadata va khong duoc tu minh authorize block.
- Matching dung strongest available stable identity va khong fallback sang identity yeu hon khi identity manh hon ton tai nhung khong match.
- Matching la exact, same-kind, deterministic; thieu stable author identity thi fail closed va khong block.
- Account bi block phai bi bo qua tren moi group va lan scan tuong lai.

## LeadStatus

Purpose: Trang thai workflow cua lead.

Required values: `new`, `viewed`, `contacted`, `ignored`.

Optional fields: Khong ap dung neu la enum/value object.

Identity/key: Gia tri enum.

Lifecycle: Mac dinh `new`, doi theo thao tac nguoi dung.

Invariants:

- Tab UI phai phan anh trang thai nay.
- `ignored` khac voi `BlockedAuthor`; ignored chi ap dung cho lead, block ap dung cho account.

## AppSettings

Purpose: Luu cau hinh local cua app.

Required fields: `id`, `defaultGeographicMode`, `defaultSearchProfileId`, `dryRunEnabled`, `timezone`.

Optional fields: `lastOpenedAt`, `scanOverlapSeconds`, `uiPreferences`.

Identity/key: Singleton local settings.

Lifecycle: Tao luc first run, cap nhat qua Settings.

Invariants:

- `defaultGeographicMode` moi ngay va moi lan mo app la TP.HCM.
- `defaultSearchProfileId` trong MVP la built-in MacBook Search Profile.
- `dryRunEnabled` mac dinh bat trong MVP.
- `timezone` la `Asia/Ho_Chi_Minh`.
