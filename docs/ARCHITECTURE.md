# Architecture

Tai lieu nay la nguon chuan tac cho dependency boundaries cua ScanFB.

Phase 1 da chon Go lam ngon ngu chinh va tao skeleton package toi thieu. Tai lieu nay van la nguon chuan tac cho boundary; skeleton hien tai chua trien khai domain model, rule engine, persistence, Facebook adapter hoac UI framework.

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
- `internal/domain`: entity, value object va invariant thuan.
- `internal/application`: orchestration/use cases; phu thuoc domain.
- `internal/rules`: deterministic buyer-intent, author, time va geographic rules.
- `internal/dedup`: duplicate detection va lead aggregation.
- `internal/persistence`: persistence-facing contracts cho completed batch snapshots; chua co SQLite, file I/O hoac storage adapter.
- `internal/facebook`: adapter bien ngoai cho Facebook/browser; domain khong duoc import.
- `internal/ui`: presentation layer; chua chon UI framework.

### App/UI

Hien thi group, batch state, lead tabs, settings, blocklist va Dry Run review. UI goi application services, khong import Facebook adapter truc tiep.

### Application services

Dieu phoi scan batch, time window, state machine va domain services. Day la noi noi adapter voi domain. Application khong import `internal/persistence` trong contract hien tai.

### Domain

Chua entity, value object, SearchProfile, BuyerIntentClassifier, rule engine, geographic classifier, deduplication va lead aggregation. Domain khong import UI, persistence implementation, browser automation hoac Facebook selector.

Domain duoc giu trung lap o phan tai su dung nhu `ScanSession`, `ScanBatch`, `RawPost`, `AuthorIdentity`, `GeographicClassification`, `FilterDecision`, `LeadSource`, `PostDeduplicator` va `SearchProfile`. Phan lead/filter duoc phep ghi ro buyer-only bang `BuyerIntentClassifier`, `BuyerLead` hoac `Lead` voi invariant buyer-only.

### Persistence interfaces

Dinh nghia persistence-facing contract cho completed scan batch snapshot. Phase 5C chi co opaque `BatchRecordID`, completed `BatchRecord`, structural validation va save-only `BatchRepository.SaveBatch`; khong co load/list/update/delete/search/paging/schema/migration/transaction API. `internal/persistence` duoc phu thuoc `internal/application` va `internal/domain` de copy completed batch result; application, domain, rules, dedup va blocklist khong phu thuoc persistence.

### Persistence implementation

Luu du lieu local. SQLite la ung vien cho Phase 7, nhung Phase 0 chua khoa cong nghe.

### Facebook adapter

Doc trang Facebook ma nguoi dung da dang nhap trong browser profile. Adapter chuyen DOM Facebook thanh `RawPost` va bao loi fail-closed. Adapter khong chua rule domain, khong biet MacBook-specific extraction va khong biet seller behavior.

## Dependency direction

```text
App/UI -> Application services -> Domain
Persistence-facing contracts -> Application services
Persistence-facing contracts -> Domain
Persistence implementation -> Persistence-facing contracts
Facebook adapter -> Application services
Facebook adapter -> RawPost mapping contract
```

Facebook adapter khong duoc domain import nguoc lai. Domain khong duoc import adapter.

## Search Profile boundary

`SearchProfile` dinh nghia buyer target cua mot lan scan. MVP chi co built-in MacBook Search Profile. Cac profile buyer khac nhu iPhone, may anh hoac laptop chi la future architecture note sau MVP, khong phai acceptance criterion cua MVP.

Khong xay abstraction, enum hoac code cho app tim nguoi can ban trong ScanFB hien tai. Neu sau nay can san pham tim nguoi can ban, do la du an/app khac co the hoc tu ScanFB hoac tai su dung core da tach dung boundary.

## Loi fail-closed

Khi adapter gap CAPTCHA, checkpoint, login required, verification page, DOM unknown, missing critical selector hoac lost group access, application service phai danh dau group/batch that bai dung muc va dung workflow theo [SCAN_RULES.md](SCAN_RULES.md).

## Ung vien cong nghe

Python, Rust va Go la ung vien can danh gia sau. Phase 0 khong chon ngon ngu lap trinh cuoi cung.
