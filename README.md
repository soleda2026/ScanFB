# ScanFB

ScanFB la ung dung local tren macOS ho tro nguoi dung tim cac bai Facebook the hien nhu cau **can mua**. MVP trien khai duy nhat built-in MacBook Search Profile de tim nguoi can mua MacBook hoac MacBook Pro.

Trang thai hien tai: Go core da co deterministic domain/rules/dedup/blocklist/application/orchestration foundations, SQLite save/load foundations cho completed batch snapshots, Phase 9A in-memory lifecycle state machine cho mot production-shaped batch dung 5 group, va Phase 9B in-memory WatchedGroup management voi so group khong gioi han. Phase 8B da tao native SwiftUI macOS app shell buildable tai `macos/ScanFBApp/`. Phase 8C them static sample Overview dashboard cho mot completed batch minh hoa bang du lieu synthetic. Phase 8D thay placeholder `Leads` bang fixture-only buyer lead tabs/cards voi du lieu synthetic va action disabled. Phase 8E thay placeholder `Dry Run` bang fixture-only review tabs/cards cho bai duoc chon, can xem xet va da loai. Phase 8F thay placeholder `Blocklist` va `Cài đặt` bang fixture-only presentation screens voi du lieu synthetic, read-only va action disabled. Phase 8G them in-memory interaction state cho fixture Leads, va Phase 8H gioi han states con `Mới`, `Đã xem`, `Bỏ qua` voi action `Tương tác` handoff URL nguon mau sang trinh duyet macOS mac dinh. Phase 8I.2 them bridge slice dau tien chi cho Go core readiness bang local subprocess helper, voi response gioi han `schema_version`, `readiness_status` va `core_identity`; Phase 8I.2a package helper vao Debug app bundle tai `Contents/Helpers/scanfb-bridge-helper`. Chua co production scan workflow, five-group queue policy, browser extension, Facebook integration, watched-group persistence/UI/bridge, lead/search bridge hoac production database path.

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

## Quy trinh phat trien

Moi milestone chi co mot muc tieu chinh, scope nho va acceptance criteria ro rang. Truoc khi sua, agent phai doc README, PRD, SCAN_RULES, ARCHITECTURE, AI_ENGINEERING_KIT va TESTING, sau do xac dinh milestone hien tai.

Nguoi dung tu quan ly Git. Agent khong duoc commit, merge, tag, push, reset, clean hoac xoa ngoai scope.

## Go core va UI direction

Module Go `github.com/soleda2026/ScanFB` hien giu core authoritative cho domain invariant, SearchProfile MacBook MVP, deterministic rules, geographic classification, deduplication, blocklist behavior, application batch evaluation, Phase 9A in-memory group-attempt lifecycle, Phase 9B in-memory WatchedGroup management, orchestration va SQLite snapshot persistence. Phase 9B preserve insertion order va active/inactive state nhung khong chon next five groups hoac dinh nghia queue ordering.

Native macOS UI direction cho Phase 8 la SwiftUI app shell trong `macos/ScanFBApp/`. Phase 8B app shell co mot main window va placeholder navigation. Phase 8C chi thay placeholder `Tổng quan` bang dashboard sample/demonstration data. Phase 8D chi thay placeholder `Leads` bang ba tab fixture `Tất cả`, `Đủ điều kiện`, `Cần xem xét` va bon buyer lead cards synthetic. Phase 8E chi thay placeholder `Dry Run` bang ba tab fixture `Được chọn`, `Cần xem xét`, `Đã loại` va muoi post cards synthetic. Phase 8F chi thay placeholder `Blocklist` va `Cài đặt` bang fixture-only read-only screens; display-name-only identity khong phai block identity authoritative va settings khong ghi persistence. Phase 8G/8H chi giu state tuong tac trong memory cho Leads fixture voi `new/viewed/ignored`; state reset khi app restart va khong phai persisted domain status. `Tương tác` la action browser handoff, khong phai status va khong chung minh nguoi dung da like, comment, message hay thuc hien buoc nao tren Facebook. Fixture URL la synthetic, khong phai production Facebook data; trong tuong lai URL nguon authoritative den tu Go-backed lead data. Phase 8I.2 chi ket noi `Cài đặt` -> Integration status -> Go bridge voi nut `Kiểm tra kết nối` de chay helper readiness theo yeu cau nguoi dung; khong auto-run, khong polling va khong expose product data. Cac fixture UI con lai khong ket noi Go core, khong lay tu Facebook, khong doc/ghi database va khong co production behavior. Mo Xcode project bang:

```bash
open macos/ScanFBApp/ScanFBApp.xcodeproj
```

SwiftUI chi la presentation layer; khong reimplement business rules, reason codes, summaries, deduplication, blocklist outcomes hoac SQLite access. Phase 8I.2 implements only `core_readiness` through a local subprocess helper. Debug builds package the helper at `Contents/Helpers/scanfb-bridge-helper`; app runtime resolves only that bundle-relative executable and fails closed when it is absent. No PATH search, shell fallback, hidden network, Facebook, persistence write or broad bridge API exists.

## Luu y ve Facebook integration

Facebook integration chi la adapter de doc trang dang mo trong moi truong nguoi dung da dang nhap. Adapter nay de thay doi theo DOM Facebook va khong phai source of truth cua domain. Domain layer chi tin vao `RawPost`, rule engine, geographic classifier, deduplication va lead aggregation duoc test bang fixture.

Trong UI hien tai, `Tương tác` chi dua URL nguon hop le cho macOS default browser. Browser so huu dang nhap/session/cookies; ScanFB khong luu credential, cookie, browser session va khong tu dong like/comment/message hay inspect Facebook.

## Search Profile

MacBook la Search Profile dau tien va la profile duy nhat duoc trien khai trong MVP. Kien truc duoc giu du trung lap de sau nay co the them buyer Search Profile khac nhu iPhone, may anh hoac laptop, nhung chi sau khi MacBook profile hoat dong on dinh va co fixture/rule deterministic rieng. App tim nguoi can ban, neu co trong tuong lai, la du an/app khac va khong nam trong roadmap ScanFB.
