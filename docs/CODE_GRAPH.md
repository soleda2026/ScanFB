# Code Graph

Tai lieu nay mo ta graph kien truc muc tieu, module boundaries va dependency direction. Phase 2 da co Go domain types toi thieu nhung chua gia dinh codebase-memory-mcp/codegraph da duoc cau hinh va khong tu cai tool.

## Mermaid dependency graph

```mermaid
flowchart TD
    UI["App/UI"] --> APP["Application services"]
    UI --> ORCH["Use-case orchestration"]
    ORCH --> APP
    ORCH --> PERSIST_CONTRACT
    APP --> DOMAIN["Domain"]
    PERSIST_CONTRACT["Persistence-facing contracts"] --> APP
    PERSIST_CONTRACT --> DOMAIN
    PERSIST_IMPL["Persistence implementation"] --> PERSIST_CONTRACT
    SQLITE["SQLite schema bootstrap"] --> PERSIST_CONTRACT
    FB["Facebook adapter"] --> APP
    FB --> RAW["RawPost mapping contract"]
    RAW --> APP
    PROFILE["SearchProfile"] --> DOMAIN
    DOMAIN --> TYPES["Domain types and reason codes"]
    BLOCKLIST["Blocklist identity primitives"] --> DOMAIN
```

## Module ownership

- `cmd/scanfb`: binary entry point toi thieu.
- `internal/domain`: Go package cho domain ownership.
- `internal/application`: Go package cho application service ownership.
- `internal/blocklist`: Go package cho deterministic local blocklist identity primitives.
- `internal/rules`: Go package cho deterministic buyer rules ownership.
- `internal/dedup`: Go package cho deduplication ownership.
- `internal/persistence`: Go package cho persistence-facing completed-batch contract, in-memory adapter va SQLite schema-bootstrap ownership.
- `internal/orchestration`: Go package cho thin synchronous use-case orchestration ownership.
- `internal/facebook`: Go package cho Facebook/browser adapter boundary ownership.
- `internal/ui`: Go package cho presentation boundary ownership.
- App/UI: owner cua views, tabs, lead cards, settings va user actions.
- Application services: owner cua deterministic in-memory scan batch model, batch state va time window.
- Use-case orchestration: owner cua glue logic giua completed application result, `BatchRecord` conversion va repository save boundary.
- Domain: owner cua normalization contracts, SearchProfile, BuyerIntentClassifier, rule engine, geographic classifier, deduplication, lead aggregation va reason codes.
- Persistence-facing contracts: owner cua completed batch snapshot contracts.
- Persistence implementation: owner cua local storage implementation.
- Facebook adapter: owner cua page reading, DOM parsing va fail-closed adapter errors.

Phase 2 files trong `internal/domain` gom minimal models cho `RawPost`, `AuthorIdentity`, `SearchProfile`, `GeographicMode`, `ScanWindow` va `ScanRequest`. Domain package chi duoc import Go standard library.

Phase 3A files trong `internal/rules` gom deterministic primitives cho post-time eligibility va author exclusion. Phase 3B them deterministic buyer-intent va seller/noise text matching dua tren active `SearchProfile`. Rules package chi duoc import Go standard library va `internal/domain`.

Phase 3C them `internal/rules/geography.go` va `internal/rules/geography_test.go` cho finite MVP geographic classification, `GeographicMode` evaluation va composed buyer-search-with-geography evaluation. Khong co geocoder, location database, foreign classifier hoac foreign exclusion.

Phase 4A them `internal/dedup/identity.go`, `internal/dedup/compare.go` va `internal/dedup/identity_test.go` cho stable author key, buyer need key, candidate key va deterministic duplicate comparison primitives. Chua co lead aggregation, persistence hoac source-post merging.

Phase 4B them `internal/dedup/aggregate.go` va `internal/dedup/aggregate_test.go` cho deterministic in-memory lead aggregation. Moi source post duoc preserve bang full `RawPost`; post khong auto-aggregated va source conflicts duoc tra ve explicit. Chua co persistence, repository, Facebook adapter, UI hoac scan orchestration.

Phase 4C them `internal/blocklist` cho local deterministic blocklist identity primitives. Supported stable identity kinds la Facebook user ID, canonical profile URL va username. Matching dung strongest available stable author identity, exact same-kind normalized key va fail closed khi thieu stable identity. Display name chi la metadata, khong duoc dung de block. Chua co persistence, scan orchestration, application integration, UI hoac CLI.

Phase 4D them `internal/application/lead_filter.go` cho in-memory filtering cua aggregated buyer leads bang local blocklist. Ket qua tach explicit allowed, blocked va unresolved leads; blocked/unresolved leads van giu nguyen source posts. Chua co persistence, scan orchestration, raw-post rule evaluation, UI hoac CLI wiring.

Phase 5A them `internal/application/evaluation_pipeline.go` cho deterministic in-memory pipeline tu already-collected `RawPost` values qua rules, eligible selection, dedup aggregation va blocklist filtering. Pipeline preserve evaluated, eligible, review, excluded, unaggregated, conflicts, allowed, blocked va unresolved outputs. Chua co Facebook adapter, persistence, UI, CLI behavior, scheduling, concurrency hoac network behavior.

Phase 5B them `internal/application/scan_batch.go` cho deterministic in-memory manual batch model gom mot den nam explicit groups. Batch validate group identity va post/group consistency, flatten posts theo group order roi post order, goi Phase 5A pipeline mot lan va tao batch/per-group count summaries. Chua co Facebook collection, persistence, UI, CLI behavior, scheduling, retries, progress reporting, concurrency hoac network behavior.

Phase 5C them `internal/persistence/batch_record.go` cho completed scan batch snapshot contract. Contract gom opaque `BatchRecordID`, immutable-style `BatchRecord`, structural validation, deterministic converter tu `application.ScanBatchInput`/`ScanBatchResult` va save-only `BatchRepository.SaveBatch`. Chua co SQLite, schema, migration, file I/O, load/list/update/delete/search/paging API, ID generation, Facebook adapter, UI/CLI, concurrency hoac network behavior.

Phase 5D them `internal/persistence/in_memory_batch_repository.go` cho deterministic in-memory adapter satisfy `BatchRepository`. Adapter validate `BatchRecord`, reject duplicate ID without overwrite, preserve insertion order bang slice, dung map chi cho lookup, va expose concrete helpers `Count`, `Records`, `RecordByID`. `BatchRepository` van save-only; chua co durable storage, SQLite, SQL, schema, migration, JSON/file I/O, goroutine, network hoac ID generation.

Phase 5E them `internal/orchestration/run_and_save_scan_batch.go` cho synchronous use case `RunAndSaveScanBatch`. Use case validate repository boundary, chap nhan caller-supplied `BatchRecordID`, chay `application.RunScanBatch`, convert successful result bang `persistence.NewBatchRecord`, save dung mot lan qua `persistence.BatchRepository`, va chi tra result/record sau khi save thanh cong. Chua co UI/CLI wiring, Facebook collection, durable storage, concurrency, retry, generated ID hoac rule moi.

Phase 5F them documentation-only SQLite schema design tai `docs/PERSISTENCE_SCHEMA.md`. Phase 5G1 them `internal/persistence/sqlite_repository.go` va `internal/persistence/sqlite_schema.go` cho SQLite foundation: `modernc.org/sqlite` driver import chi trong persistence, explicit-path open/create, foreign-key enable/verify, transactional empty schema v1 creation, schema metadata validation va `Close`. `SQLiteBatchRepository` chua co `SaveBatch`, khong satisfy `BatchRepository`, va chua co durable snapshot write/load/list/migration behavior.

## Allowed dependencies

- App/UI duoc goi Application services.
- Application services duoc goi Domain, Rules, Dedup va Blocklist.
- Persistence-facing contracts duoc goi Application services va Domain de copy completed batch snapshots.
- Persistence implementation duoc implement Persistence-facing contracts.
- SQLite schema bootstrap duoc implement trong `internal/persistence`; future durable save adapter se implement Persistence-facing contracts trong cung package.
- Facebook adapter duoc goi Application services bang adapter boundary.
- Tests duoc import Domain truc tiep de chay fixture deterministic.
- `internal/rules` duoc import `internal/domain`.
- `internal/dedup` duoc import `internal/domain`.
- `internal/blocklist` duoc import `internal/domain`.
- `internal/application` duoc import `internal/domain`, `internal/rules`, `internal/dedup` va `internal/blocklist`.
- `internal/persistence` duoc import `internal/application` va `internal/domain`.
- `modernc.org/sqlite` chi duoc import trong `internal/persistence`.
- `internal/orchestration` duoc import `internal/application` va `internal/persistence`.

## Forbidden dependencies

- Domain khong duoc import App/UI.
- Domain khong duoc import Facebook adapter.
- Domain khong duoc chua CSS selector, DOM traversal, browser automation hoac Facebook-specific parsing.
- Facebook adapter khong duoc chua buyer business rules hoac product-specific extraction.
- UI khong duoc tu quyet dinh include/exclude thay rule engine.
- Persistence implementation khong duoc dieu khien browser.
- Application, Domain, Rules, Dedup va Blocklist khong duoc import `internal/persistence`.
- Application, Domain, Rules, Dedup, Blocklist va Persistence khong duoc import `internal/orchestration`.
- `internal/orchestration` khong duoc import Facebook adapter, UI hoac CLI package.
- `internal/persistence` khong duoc import Facebook adapter, UI, CLI package hoac persistence implementation package.
- Khong package nao ngoai `internal/persistence` duoc import `modernc.org/sqlite`.
- Khong module nao trong ScanFB duoc them seller mode, `SellerLead`, `SellerIntentClassifier`, `LeadIntent` buyer/seller hoac `SELLER_SCAN`.

## Entry points

- User bam Scan: App/UI goi Application service tao `ScanSession` va `ScanBatch`.
- User mo lead: App/UI mo canonical post URL tu `LeadSource`.
- User danh dau status: App/UI goi Application service cap nhat `LeadStatus`.
- User bo qua account: App/UI goi Application service tao `BlockedAuthor`.
- Facebook adapter: application service kich hoat adapter doc group hien tai trong batch.

## Data flow cho Scan batch

```text
User Scan
-> App/UI
-> Application services tao ScanSession voi startedAt va geographicMode
-> Gan built-in MacBook SearchProfile trong MVP
-> Chon dung 5 WatchedGroup cho ScanBatch
-> Voi tung group: GroupScanAttempt pending -> running
-> Facebook adapter doc page va tao RawPost
-> Application services dua RawPost vao domain pipeline
-> Luu RawPost, FilterDecision, Lead/LeadSource
-> GroupScanAttempt succeeded/failed/skipped/expired_at_day_boundary
-> Batch summary tra ve UI
```

## Data flow cho filter/dedup

```text
RawPost
-> In-memory ScanBatch group validation
-> Deterministic flattening theo group order va post order
-> Rule evaluation: time, author, SearchProfile target keyword, BuyerIntentClassifier va geographic classification
-> Eligible/review/excluded separation
-> In-memory buyer lead aggregation
-> Local blocklist lead filtering
-> Explicit allowed, blocked, unresolved, unaggregated va conflict outputs
-> Count-only batch summary va per-group rule-stage summaries
-> Persistence-facing completed BatchRecord snapshot contract
-> Optional in-memory BatchRepository adapter cho deterministic inspection/testing
-> Optional thin RunAndSaveScanBatch orchestration cho explicit caller-supplied record ID
-> Optional SQLite schema-bootstrap foundation neu caller mo explicit local DB path
```

## SearchProfile va buyer-only boundary

MVP chi co built-in MacBook Search Profile. Graph duoc giu trung lap cho phan scan/post/author/geo/dedup de sau nay co the them buyer Search Profile khac, nhung khong bien Phase hien tai thanh framework da san pham.

App tim nguoi can ban, neu co trong tuong lai, la du an/app khac. ScanFB hien tai khong co seller tab, seller mode selector, seller classifier hay enum intent buyer/seller.

## Cap nhat code graph khi code thay doi

Khi bat dau co code, moi thay doi module boundary phai cap nhat tai lieu nay neu:

- Them module moi.
- Doi dependency direction.
- Them entry point moi.
- Doi pipeline scan/filter/dedup.
- Them persistence hoac adapter implementation moi.
- Thay doi SQLite schema bootstrap, dependency boundary hoac schema version.

Khong cap nhat graph de hop thuc hoa dependency bi cam. Neu code can dependency bi cam, phai sua design hoac xin quyet dinh san pham/kien truc truoc.

## Huong dan dung codebase-memory-mcp/codegraph ve sau

Neu project sau nay duoc cau hinh codebase-memory-mcp hoac CodeGraph:

- Dung graph tools de tim symbol, module boundary va call path truoc khi grep.
- Dung grep/file search cho string literal, config, markdown hoac khi graph chua co ket qua.
- Khong tu cai, khoi tao hoac rebuild tool trong milestone khong cho phep.
- Khong gia dinh graph da dung neu `.codegraph/` hoac cau hinh MCP khong ton tai.
