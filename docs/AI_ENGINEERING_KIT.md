# AI Engineering Kit

Tai lieu nay la nguon chuan tac cho workflow agent trong ScanFB.

## Nguyen tac milestone

- Mot milestone chi co mot muc tieu chinh.
- Scope phai nho, co protected areas va acceptance criteria ro.
- Doc truoc khi viet: README, PRD, SCAN_RULES, ARCHITECTURE, AI_ENGINEERING_KIT va TESTING.
- Khong sua ngoai milestone.
- Khong tu mo rong scope.
- Khong them seller mode, seller tab, seller classifier, seller lead hoac `LeadIntent` buyer/seller vao ScanFB.
- Khong bien MacBook MVP thanh framework da san pham trong Phase 0; SearchProfile khac chi la future note cho buyer search.
- Khong commit, merge, tag hoac push. Nguoi dung tu lam Git.
- Khong xoa, reset, clean hoac overwrite ngoai scope.
- Khong dung wildcard destructive command.
- Khong tu cai dependency.
- Khong bia build/test result.

## Preflight

Truoc khi sua:

- Xac dinh milestone hien tai.
- Xac dinh muc tieu duy nhat.
- Doc cac tai lieu chuan tac lien quan.
- Kiem tra workspace co file chua ro nguon goc hay khong.
- Dung lai neu trang thai khong ro hoac co mau thuan scope.

## Trong khi lam

- Giu thay doi nho va dung boundary.
- Production code phai kem test phu hop.
- Facebook integration lam sau domain/rule engine.
- Khong de Facebook selector ro vao domain.
- Neu lam voi SearchProfile, mac dinh MVP chi co built-in MacBook profile va moi profile ScanFB deu la buyer-intent.
- Neu gap ambiguity hoac Facebook DOM thay doi, dung va bao thay vi doan.
- Khong tu cai package manager, runtime, extension hoac cau hinh he thong.

## Test va xac minh

- Chi noi da chay test/build khi thuc te da chay.
- Neu milestone docs-only, khong build vi chua co code.
- Khi test app thu cong ve sau, phai kill app process cu va relaunch ban build moi truoc khi xac nhan UI.
- Rule engine va deduplication phai test bang fixture, khong can Facebook hoac browser.

## Report sau moi milestone

Bao cao:

1. Files changed.
2. Decisions.
3. Tests run va ket qua that.
4. Failures neu co.
5. Remaining risks.
6. Xac nhan co hay khong production code.
7. Xac nhan co hay khong Git write.

## Stop conditions

Dung khi:

- Yeu cau hien tai doi hoi vuot milestone.
- Can chon ngon ngu lap trinh cuoi cung khi milestone chua cho phep.
- Can cloud backend, account system, multi-user, subscription hoac AI model ngoai scope.
- Facebook bat CAPTCHA, checkpoint, login, verification hoac DOM khong xac dinh.
- Co nguy co mat du lieu do xoa/reset/clean/overwrite.
