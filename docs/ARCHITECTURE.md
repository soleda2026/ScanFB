# Architecture

Tai lieu nay la nguon chuan tac cho dependency boundaries cua ScanFB.

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

### App/UI

Hien thi group, batch state, lead tabs, settings, blocklist va Dry Run review. UI goi application services, khong import Facebook adapter truc tiep.

### Application services

Dieu phoi scan batch, time window, state machine, persistence transaction va domain services. Day la noi noi adapter voi domain.

### Domain

Chua entity, value object, SearchProfile, BuyerIntentClassifier, rule engine, geographic classifier, deduplication va lead aggregation. Domain khong import UI, persistence implementation, browser automation hoac Facebook selector.

Domain duoc giu trung lap o phan tai su dung nhu `ScanSession`, `ScanBatch`, `RawPost`, `AuthorIdentity`, `GeographicClassification`, `FilterDecision`, `LeadSource`, `PostDeduplicator` va `SearchProfile`. Phan lead/filter duoc phep ghi ro buyer-only bang `BuyerIntentClassifier`, `BuyerLead` hoac `Lead` voi invariant buyer-only.

### Persistence interfaces

Dinh nghia repository contract cho group, session, raw post, decision, lead, source va blocklist. Domain co the phu thuoc vao interface neu can, nhung khong phu thuoc implementation.

### Persistence implementation

Luu du lieu local. SQLite la ung vien cho Phase 7, nhung Phase 0 chua khoa cong nghe.

### Facebook adapter

Doc trang Facebook ma nguoi dung da dang nhap trong browser profile. Adapter chuyen DOM Facebook thanh `RawPost` va bao loi fail-closed. Adapter khong chua rule domain, khong biet MacBook-specific extraction va khong biet seller behavior.

## Dependency direction

```text
App/UI -> Application services -> Domain
App/UI -> Application services -> Persistence interfaces
Persistence implementation -> Persistence interfaces
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
