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
- `internal/domain`: entity, value object va invariant thuan.
- `internal/application`: application services/use cases; phu thuoc domain, rules, dedup va blocklist.
- `internal/orchestration`: thin synchronous use cases ket noi completed application result voi persistence-facing contract.
- `internal/rules`: deterministic buyer-intent, author, time va geographic rules.
- `internal/dedup`: duplicate detection va lead aggregation.
- `internal/persistence`: persistence-facing contracts, deterministic in-memory adapter cho completed batch snapshots, va SQLite schema-bootstrap/transactional `SaveBatch`/concrete `LoadBatch` implementation; chua co list/update/delete/search/paging API hoac migration execution.
- `internal/facebook`: adapter bien ngoai cho Facebook/browser; domain khong duoc import.
- `internal/ui`: Go-layer documentation/package placeholder. Native macOS UI implementation is planned outside `internal/` using SwiftUI.

### App/UI

SwiftUI la approved native macOS presentation direction cho Phase 8. Future app root duoc du kien la `macos/ScanFBApp/`, nam ngoai Go `internal/` package tree. SwiftUI se hien thi group, batch state, lead tabs, settings, blocklist va Dry Run review, nhung khong reimplement Go business logic. UI goi future narrow application/orchestration adapter sau khi bridge duoc quyet dinh, va khong import Facebook adapter truc tiep.

### Application services

Dieu phoi scan batch, time window, state machine va domain services. Day la noi noi adapter voi domain. Application khong import `internal/persistence` trong contract hien tai.

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

### Facebook adapter

Doc trang Facebook ma nguoi dung da dang nhap trong browser profile. Adapter chuyen DOM Facebook thanh `RawPost` va bao loi fail-closed. Adapter khong chua rule domain, khong biet MacBook-specific extraction va khong biet seller behavior.

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

Native SwiftUI code, khi duoc tao trong milestone sau, phai song ngoai `internal/`. Go application/domain packages khong duoc phu thuoc Swift hoac macOS UI code. Bridge SwiftUI-Go chua duoc chon; candidate bridge phai duoc danh gia trong milestone rieng va khong duoc ngam dinh HTTP, cloud API, direct SQLite access tu Swift hoac arbitrary JSON maps.

Chi tiet Phase 8 nam trong [MACOS_UI_ARCHITECTURE.md](MACOS_UI_ARCHITECTURE.md).

## Search Profile boundary

`SearchProfile` dinh nghia buyer target cua mot lan scan. MVP chi co built-in MacBook Search Profile. Cac profile buyer khac nhu iPhone, may anh hoac laptop chi la future architecture note sau MVP, khong phai acceptance criterion cua MVP.

Khong xay abstraction, enum hoac code cho app tim nguoi can ban trong ScanFB hien tai. Neu sau nay can san pham tim nguoi can ban, do la du an/app khac co the hoc tu ScanFB hoac tai su dung core da tach dung boundary.

## Loi fail-closed

Khi adapter gap CAPTCHA, checkpoint, login required, verification page, DOM unknown, missing critical selector hoac lost group access, application service phai danh dau group/batch that bai dung muc va dung workflow theo [SCAN_RULES.md](SCAN_RULES.md).

## UI technology

SwiftUI is the approved native macOS presentation technology for Phase 8. The Go core remains authoritative for rules, deduplication, blocklist behavior, batch evaluation, persistence snapshots, reason codes and deterministic decisions.
