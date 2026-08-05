# ScanFB Claude Instructions

Truoc khi thuc hien bat ky thay doi nao, hay doc:

- [README.md](README.md)
- [docs/PRD.md](docs/PRD.md)
- [docs/SCAN_RULES.md](docs/SCAN_RULES.md)
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [docs/AI_ENGINEERING_KIT.md](docs/AI_ENGINEERING_KIT.md)
- [docs/TESTING.md](docs/TESTING.md)

## Nguyen tac lam viec

- Xac dinh milestone hien tai truoc khi sua.
- Chi lam dung muc tieu duy nhat cua milestone.
- Khong mo rong scope sang cloud backend, account system, multi-user, subscription, AI model hoac browser extension neu milestone khong yeu cau.
- Khong them seller mode, `LeadIntent` buyer/seller, `SellerLead` hoac `SellerIntentClassifier`. ScanFB hien tai la buyer-only.
- Khong chon ngon ngu lap trinh cuoi cung trong Phase 0. Python, Rust va Go chi la ung vien can danh gia sau.
- Khong cai dependency va khong thay doi cau hinh he thong.
- Khong thuc hien Git write operations. Nguoi dung tu lam Git.
- Khong sua file ngoai project ScanFB va khong cham vao project khac tren `/Volumes/Apps`.

## Khi can dung lai

Dung va bao nguoi dung khi:

- Preflight khong sach hoac trang thai workspace khong ro.
- Yeu cau hien tai mau thuan voi [docs/SCAN_RULES.md](docs/SCAN_RULES.md), [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) hoac [docs/AI_ENGINEERING_KIT.md](docs/AI_ENGINEERING_KIT.md).
- Facebook DOM, selector hoac du lieu dau vao khong du de ket luan deterministic.
- Co nguy co xoa, overwrite, reset hoac clean ngoai scope.
