# ScanFB

ScanFB la ung dung local tren macOS ho tro nguoi dung tim cac bai Facebook the hien nhu cau **can mua**. MVP trien khai duy nhat built-in MacBook Search Profile de tim nguoi can mua MacBook hoac MacBook Pro.

Trang thai hien tai: Go core da co deterministic domain/rules/dedup/blocklist/application/orchestration foundations, SQLite save/load foundations cho completed batch snapshots, Phase 9A lifecycle, Phase 9B WatchedGroup management, Phase 9C active-only round-robin selector va Phase 9D selection-to-lifecycle mapping. Phase 9E1 Watched Groups UI da qua automated va user-guided manual verification; Phase 9E2 Go-owned persistence va Phase 9E2b product-contract correction da complete automated verification, va corrected Groups UI da qua user-guided manual verification. Full ordered WatchedGroup state va exact Phase 9C cursor duoc luu trong dedicated `watched-groups.sqlite3`, voi Swift chi giu presentation snapshot. Primary future group source la joined-group discovery tu authenticated Facebook/Safari context; Phase 9E3a one-page reconnaissance tim thay canonical group-link/name association nhung khong co strong item boundary hoac joined-versus-recommended distinction, nen Phase 9E3 discovery implementation hien BLOCKED. Current UI chi hien read-only next-five preview; discovery va production scan execution chua duoc implement. Phase 10A co fixture-only prepared-page extraction boundary; Phase 10B1 Safari active-tab acquisition da manual validate; Phase 10B2d rendered-DOM acquisition va Phase 10B2f typed redacted preservation hoat dong. Phase 10B2g ghi nhan 2 semantic article candidates nhung 0 complete-post evidence, nen Phase 10B2b production selector van BLOCKED va current Safari selector investigation da dong. Phase 8B-8H cung cap native SwiftUI shell va fixture presentation; Phase 8I.2/8I.2a cung cap bounded local subprocess readiness bridge va Debug helper packaging. Chua co production scan workflow/execution, production Facebook DOM selectors, browser extension hoac lead/search bridge.

## ScanFB khong lam gi

- Khong tu chay theo lich. Ung dung chi hoat dong khi nguoi dung bam Scan.
- Khong luu username, password Facebook va khong tu dang nhap.
- Khong tu like, comment, inbox, ket ban, dang bai hoac thuc hien giao dich.
- Khong tu vuot CAPTCHA, checkpoint, trang xac minh hoac yeu cau dang nhap lai.
- Khong gui noi dung bai len dich vu cloud trong MVP.
- Khong co seller mode, khong tim nguoi can ban va khong co `LeadIntent` buyer/seller.
- Khong co UI tao Search Profile tuy y, plugin system hoac profile marketplace trong MVP.

## Tai lieu

- [docs/PRD.md](docs/PRD.md): muc tieu san pham, pham vi MVP va hanh vi nguoi dung.
- [docs/SCAN_RULES.md](docs/SCAN_RULES.md): nguon chuan tac cao nhat cho scan, filter, geo, author, blocklist, dedup va reason code.
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): nguon chuan tac cho module boundary va dependency direction.
- [docs/DATA_MODEL.md](docs/DATA_MODEL.md): entity, field, key, lifecycle va invariant.
- [docs/AI_ENGINEERING_KIT.md](docs/AI_ENGINEERING_KIT.md): workflow bat buoc cho coding agents.
- [docs/TESTING.md](docs/TESTING.md): test strategy va test matrix MVP.
- [docs/ROADMAP.md](docs/ROADMAP.md): roadmap milestone nho.
- [docs/SECURITY_AND_PRIVACY.md](docs/SECURITY_AND_PRIVACY.md): yeu cau bao mat va quyen rieng tu.
- [docs/CODE_GRAPH.md](docs/CODE_GRAPH.md): graph kien truc muc tieu va cach cap nhat graph.
- [docs/MACOS_UI_ARCHITECTURE.md](docs/MACOS_UI_ARCHITECTURE.md): quyet dinh SwiftUI native macOS shell va plan Phase 8.
- [docs/BRIDGE_DECISION.md](docs/BRIDGE_DECISION.md): Phase 8I bridge decision va readiness-only SwiftUI-Go subprocess slice.
- [docs/WATCHED_GROUP_PERSISTENCE_DECISION.md](docs/WATCHED_GROUP_PERSISTENCE_DECISION.md): Phase 9E2a production location, dedicated state schema va cursor persistence decision.
- [docs/FACEBOOK_JOINED_GROUPS_RECONNAISSANCE.md](docs/FACEBOOK_JOINED_GROUPS_RECONNAISSANCE.md): Phase 9E3a one-page joined-groups structural evidence va STOP/INCONCLUSIVE decision.
- [docs/SAFARI_RENDERED_DOM_ACQUISITION_DECISION.md](docs/SAFARI_RENDERED_DOM_ACQUISITION_DECISION.md): Phase 10B2c rendered-DOM acquisition decision va future boundary.
- [docs/FACEBOOK_SAFARI_RENDERED_DOM_RECONNAISSANCE.md](docs/FACEBOOK_SAFARI_RENDERED_DOM_RECONNAISSANCE.md): Phase 10B2e one-page rendered-DOM reconnaissance va fail-closed result.

## Quy trinh phat trien

Moi milestone chi co mot muc tieu chinh, scope nho va acceptance criteria ro rang. Truoc khi sua, agent phai doc README, PRD, SCAN_RULES, ARCHITECTURE, AI_ENGINEERING_KIT va TESTING, sau do xac dinh milestone hien tai.

Nguoi dung tu quan ly Git. Agent khong duoc commit, merge, tag, push, reset, clean hoac xoa ngoai scope.

## Go core va UI direction

Module Go `github.com/soleda2026/ScanFB` hien giu core authoritative cho domain invariant, SearchProfile MacBook MVP, deterministic rules, geographic classification, deduplication, blocklist behavior, application batch evaluation, Phase 9A in-memory group-attempt lifecycle, Phase 9B in-memory WatchedGroup management, Phase 9C deterministic five-group selection, Phase 9D selection-to-lifecycle mapping, orchestration va SQLite snapshot persistence. Phase 9C traverses active groups circularly theo insertion order tu caller-managed cursor; `displayOrder`, `createdAt` va `lastSuccessfulScanAt` khong anh huong selection, va selector khong tao lifecycle hoac chay scan. Phase 9D preserve exact selected-group order, dung caller-supplied batch/attempt IDs va `ScanWindow`, va chi tao lifecycle voi tat ca attempt `pending`; mapper khong advance cursor, re-select, chay scan, goi Facebook, persistence, UI hoac bridge.

Native macOS UI direction cho Phase 8 la SwiftUI app shell trong `macos/ScanFBApp/`. Phase 8B app shell co mot main window va placeholder navigation. Phase 8C chi thay placeholder `Tổng quan` bang dashboard sample/demonstration data. Phase 8D chi thay placeholder `Leads` bang ba tab fixture `Tất cả`, `Đủ điều kiện`, `Cần xem xét` va bon buyer lead cards synthetic. Phase 8E chi thay placeholder `Dry Run` bang ba tab fixture `Được chọn`, `Cần xem xét`, `Đã loại` va muoi post cards synthetic. Phase 8F chi thay placeholder `Blocklist` va `Cài đặt` bang fixture-only read-only screens; display-name-only identity khong phai block identity authoritative va settings khong ghi persistence. Phase 8G/8H chi giu state tuong tac trong memory cho Leads fixture voi `new/viewed/ignored`; state reset khi app restart va khong phai persisted domain status. `Tương tác` la action browser handoff, khong phai status va khong chung minh nguoi dung da like, comment, message hay thuc hien buoc nao tren Facebook. Fixture URL la synthetic, khong phai production Facebook data; trong tuong lai URL nguon authoritative den tu Go-backed lead data. Phase 8I.2 chi ket noi `Cài đặt` -> Integration status -> Go bridge voi nut `Kiểm tra kết nối` de chay helper readiness theo yeu cau nguoi dung; khong auto-run, khong polling va khong expose product data. Cac fixture UI con lai khong ket noi Go core, khong lay tu Facebook, khong doc/ghi database va khong co production behavior. Mo Xcode project bang:

```bash
open macos/ScanFBApp/ScanFBApp.xcodeproj
```

SwiftUI chi la presentation layer; khong reimplement business rules, reason codes, summaries, deduplication, blocklist outcomes hoac SQLite access. Phase 8I.2 implements `core_readiness`; Phase 9E2 giu bon operation hep `watched_groups_list`, `watched_groups_add`, `watched_groups_set_active` va `watched_groups_next_five`, nhung Go persistent store nay la authority. Request khong gui full collection, cursor hay raw database path; moi response thanh cong tra authoritative groups, exact-five selection va current cursor. `WatchedGroupsStore` chi hien thi snapshot do, voi loading va storage-failure state rieng. Debug builds package helper tai `Contents/Helpers/scanfb-bridge-helper`; runtime chi resolve executable bundle-relative va fail closed neu vang mat. Khong co PATH search, shell fallback, hidden network, Facebook, lead/search bridge hoac broad command bus.

Primary future Groups flow se discover cac group da tham gia tu Facebook account dang authenticated trong Safari, synchronize chung vao source-neutral WatchedGroup persistence, va de nguoi dung bat/tat eligibility. Manual add bridge/domain support duoc giu lam fallback/scaffolding nhung bi an khoi primary UI. Cursor la internal scan-progression state: xem preview khong advance; future explicit Scan action se consume next five, chay sequentially, advance cursor va dung voi results. UI khong co manual queue-advance action.

Phase 9E2 implements approved Phase 9E2a architecture. Go helper tu resolve `<user-application-support>/com.soleda.ScanFB/watched-groups.sqlite3`; completed batches van dung database rieng `completed-batches.sqlite3`. WatchedGroup database co schema v1 doc lap va DELETE journal; restore malformed schema/data/cursor fail closed. Swift khong thay path va khong so huu persistence. Existing completed-batch schema v1 khong doi.

## Luu y ve Facebook integration

Facebook integration chi la adapter de doc trang dang mo trong moi truong nguoi dung da dang nhap. Adapter nay de thay doi theo DOM Facebook va khong phai source of truth cua domain. Domain layer chi tin vao `RawPost`, rule engine, geographic classifier, deduplication va lead aggregation duoc test bang fixture.

Phase 10A chi chung minh typed local `PreparedPageSnapshot` -> `RawPost` extraction contract bang deterministic fixtures. Phase 10B1 them `AcquireSafariActiveTab`: khi caller kich hoat, adapter goi truc tiep `/usr/bin/osascript` bang JXA de doc URL, title va bounded page source cua dung current tab trong front Safari window, voi `capturedAt` do caller cung cap. Nguoi dung tu mo Safari, dang nhap, dieu huong va de tab mong muon active; ScanFB khong auto-login, navigate, switch tab, scroll, click, poll hay doc cookie/session/credential/profile data. Apple Events Automation permission co the duoc macOS yeu cau va loi permission duoc fail closed. Manual validation da acquire thanh cong mot user-prepared Facebook group page co source khoang 1.5-1.6 MB, duoi bound 4 MiB. Phase 10B2a reconnaissance thay source chi la HTML/bootstrap shell va khong co du stable post container/body/author/permalink/timestamp evidence. Phase 10B2c approve acquisition-only route, va Phase 10B2d them `AcquireSafariActiveTabRenderedDOM`: fixed expression `document.documentElement ? document.documentElement.outerHTML : ""` chay qua Safari Apple Events tren exactly one current tab, voi decoded bound 8 MiB, finite stdout envelope, caller-supplied `capturedAt` va fail-closed errors. Probe khong co caller-supplied JavaScript, selector, `RawPost`, browser mutation, cookie/storage/session/profile access, networking, UI/bridge hoac persistence. User-guided live validation da succeed; Phase 10B2e redacted reconnaissance dung fail closed vi structural report khong duoc retain.

Trong UI hien tai, `Tương tác` chi dua URL nguon hop le cho macOS default browser. Browser so huu dang nhap/session/cookies; ScanFB khong luu credential, cookie, browser session va khong tu dong like/comment/message hay inspect Facebook.

## Search Profile

MacBook la Search Profile dau tien va la profile duy nhat duoc trien khai trong MVP. Kien truc duoc giu du trung lap de sau nay co the them buyer Search Profile khac nhu iPhone, may anh hoac laptop, nhung chi sau khi MacBook profile hoat dong on dinh va co fixture/rule deterministic rieng. App tim nguoi can ban, neu co trong tuong lai, la du an/app khac va khong nam trong roadmap ScanFB.
