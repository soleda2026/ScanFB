# Data Model

Tai lieu nay dinh nghia entity toi thieu va quan he muc domain. Kieu du lieu cu the se duoc khoa trong milestone sau. ScanFB hien tai la buyer-only: khong co `LeadIntent` buyer/seller, `SellerLead` hoac seller mode.

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

Required fields: `id`, `sourcePostUrl`, `canonicalPostUrl`, `groupId`, `author`, `rawText`, `createdAt`, `capturedAt`.

Optional fields: `facebookPostId`, `permalink`, `rawTimestampText`, `attachmentsMetadata`, `readerMetadata`.

Identity/key: `facebookPostId` neu co, nguoc lai `canonicalPostUrl`.

Lifecycle: Tao boi Facebook adapter, luu local ke ca khi bi loai trong Dry Run.

Invariants:

- Khong chua secret, cookie hoac token.
- `createdAt` phai la thoi diem tao bai goc, khong phai thoi diem activity moi.

## AuthorIdentity

Purpose: Dinh danh nguoi dang bai.

Required fields: `displayName`.

Optional fields: `facebookUserId`, `canonicalProfileUrl`, `username`, `profileUrl`, `isAnonymousSignal`.

Identity/key: Uu tien `facebookUserId`, roi `canonicalProfileUrl`, roi `username`.

Lifecycle: Gan vao RawPost, dung cho blocklist va dedup.

Invariants:

- Ten hien thi chi la thong tin phu, khong duoc dung lam khoa dedup duy nhat.
- Author anonymous hoac ten khong co dau cach bi loai theo [SCAN_RULES.md](SCAN_RULES.md).

## SearchProfile

Purpose: Dinh nghia cau hinh buyer search cho mot product/category muc tieu.

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

- Moi decision phai co it nhat mot `FilterReason`.
- Reason codes phai deterministic va machine-readable.
- Include decision trong ScanFB luon la buyer include cho SearchProfile dang bat.

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

- Bai ro rang ngoai Viet Nam bi loai.
- Bai khong ro dia diem vao Can kiem tra.
- Mac dinh mode moi ngay va moi lan mo app la TP.HCM.

## Lead

Purpose: Dai dien mot buyer lead sau dedup va aggregation.

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

Required fields: `id`, `identityKind`, `identityValue`, `reason`, `createdAt`.

Optional fields: `displayName`, `note`, `unblockedAt`.

Identity/key: Cap `identityKind` va `identityValue`.

Lifecycle: Tao tu thao tac `Bo qua account nay`, co the bo block trong Settings.

Invariants:

- Uu tien identity theo user ID, canonical profile URL, username, display name phu.
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
