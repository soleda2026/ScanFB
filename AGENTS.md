# ScanFB Agent Instructions

Doc tai lieu truoc khi sua:

1. [README.md](README.md)
2. [docs/PRD.md](docs/PRD.md)
3. [docs/SCAN_RULES.md](docs/SCAN_RULES.md)
4. [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
5. [docs/AI_ENGINEERING_KIT.md](docs/AI_ENGINEERING_KIT.md)
6. [docs/TESTING.md](docs/TESTING.md)

## Preflight bat buoc

- Xac dinh milestone hien tai va muc tieu duy nhat cua milestone.
- Kiem tra scope co phai docs-only, domain, UI, persistence hay Facebook adapter.
- Dung lai neu preflight khong sach, workspace khong ro trang thai hoac yeu cau mau thuan voi tai lieu chuan tac.

## Gioi han

- Khong mo rong scope.
- Khong tao production code khi milestone la docs-only.
- Khong them seller mode, `LeadIntent` buyer/seller, `SellerLead` hoac `SellerIntentClassifier` vao ScanFB.
- Khong thuc hien Git write operations: commit, merge, tag, push, reset, clean, checkout ghi de.
- Khong sua file ngoai project ScanFB.
- Khong truy cap hoac thay doi project khac tren `/Volumes/Apps`.
- Khong cai dependency, package manager, browser extension hoac cau hinh he thong.
- Khong xoa, overwrite hoac dung wildcard destructive command.

## Nguon chuan tac

- [docs/SCAN_RULES.md](docs/SCAN_RULES.md) la nguon chuan tac cao nhat cho hanh vi scan/filter.
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) la nguon chuan tac cho dependency boundaries.
- [docs/AI_ENGINEERING_KIT.md](docs/AI_ENGINEERING_KIT.md) la nguon chuan tac cho workflow agent.

Neu co ambiguity ve Facebook DOM, selector, classification hoac du lieu, dung va bao nguoi dung thay vi doan.
