# Checklists

## Preflight

- Da doc README, PRD, SCAN_RULES, ARCHITECTURE, AI_ENGINEERING_KIT va TESTING.
- Da xac dinh milestone hien tai.
- Da xac dinh muc tieu duy nhat.
- Da xac dinh protected areas.
- Workspace ro trang thai.
- Khong can Git write.
- Khong can dependency install.

## Docs-only milestone

- Chi tao/sua tai lieu.
- Khong tao production code.
- Khong chon ngon ngu lap trinh cuoi cung.
- Khong build/test code.
- Kiem tra internal Markdown links.
- Tim cac marker placeholder pho bien trong noi dung tai lieu.
- Kiem tra mau thuan batch size, day boundary, geographic modes va author filtering.
- Kiem tra ScanFB van buyer-only, MacBook-first va khong co seller mode.

## Production-code milestone

- Doc tai lieu chuan tac lien quan.
- Them test phu hop.
- Khong de Facebook selector vao domain.
- Rule engine/dedup co fixture deterministic.
- SearchProfile trong MVP chi la built-in MacBook profile.
- Khong them `LeadIntent` buyer/seller, seller tab, seller mode selector, `SellerLead` hoac `SellerIntentClassifier`.
- Khong bia test result.
- Report files changed, decisions, tests, failures va risks.

## Facebook adapter milestone

- Nguoi dung chu dong mo/dang nhap Facebook.
- Khong luu password.
- Khong export cookie.
- Khong auto login.
- Khong auto like/comment/inbox/add friend/post.
- Fail-closed khi CAPTCHA, checkpoint, login required, verification, DOM unknown hoac missing selector.
- Log khong chua secret.
