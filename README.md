# ScanFB

ScanFB la ung dung local tren macOS ho tro nguoi dung tim cac bai Facebook the hien nhu cau **can mua**. MVP trien khai duy nhat built-in MacBook Search Profile de tim nguoi can mua MacBook hoac MacBook Pro.

Trang thai hien tai: Go core da co deterministic domain/rules/dedup/blocklist/application/orchestration foundations va SQLite save/load foundations cho completed batch snapshots. Phase 8B da tao native SwiftUI macOS app shell buildable tai `macos/ScanFBApp/`. Chua co production scan workflow, browser extension, Facebook integration, Go bridge, UI fixtures, persistence wiring hoac production database path.

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

## Quy trinh phat trien

Moi milestone chi co mot muc tieu chinh, scope nho va acceptance criteria ro rang. Truoc khi sua, agent phai doc README, PRD, SCAN_RULES, ARCHITECTURE, AI_ENGINEERING_KIT va TESTING, sau do xac dinh milestone hien tai.

Nguoi dung tu quan ly Git. Agent khong duoc commit, merge, tag, push, reset, clean hoac xoa ngoai scope.

## Go core va UI direction

Module Go `github.com/soleda2026/ScanFB` hien giu core authoritative cho domain invariant, SearchProfile MacBook MVP, deterministic rules, geographic classification, deduplication, blocklist behavior, application batch evaluation, orchestration va SQLite snapshot persistence.

Native macOS UI direction cho Phase 8 la SwiftUI app shell trong `macos/ScanFBApp/`. Phase 8B app shell co mot main window va placeholder navigation, chua ket noi Go core va khong co Facebook/production behavior. Mo Xcode project bang:

```bash
open macos/ScanFBApp/ScanFBApp.xcodeproj
```

SwiftUI chi la presentation layer; khong reimplement business rules, reason codes, summaries, deduplication, blocklist outcomes hoac SQLite access. Bridge SwiftUI-Go van deferred va phai co milestone rieng.

## Luu y ve Facebook integration

Facebook integration chi la adapter de doc trang dang mo trong moi truong nguoi dung da dang nhap. Adapter nay de thay doi theo DOM Facebook va khong phai source of truth cua domain. Domain layer chi tin vao `RawPost`, rule engine, geographic classifier, deduplication va lead aggregation duoc test bang fixture.

## Search Profile

MacBook la Search Profile dau tien va la profile duy nhat duoc trien khai trong MVP. Kien truc duoc giu du trung lap de sau nay co the them buyer Search Profile khac nhu iPhone, may anh hoac laptop, nhung chi sau khi MacBook profile hoat dong on dinh va co fixture/rule deterministic rieng. App tim nguoi can ban, neu co trong tuong lai, la du an/app khac va khong nam trong roadmap ScanFB.
