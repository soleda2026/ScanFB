# Security And Privacy

## Nguyen tac

- Local-first.
- Thu thap it nhat co the.
- Khong luu Facebook password.
- Khong export cookie ra khoi browser profile.
- Khong gui noi dung bai len dich vu cloud trong MVP.
- Khong tai anh/video neu khong can.
- Khong tu mo profile hang loat.
- Log khong duoc chua secret, cookie hoac token.

## Du lieu local

Du lieu co the luu local:

- Watched groups nguoi dung chu dong them.
- RawPost toi thieu can cho Dry Run va audit.
- FilterDecision va FilterReason.
- Lead va LeadSource.
- BlockedAuthor.
- AppSettings.
- SearchProfile built-in va lien ket watched groups neu can.

Nguoi dung phai co kha nang export va xoa du lieu trong milestone phu hop sau nay.

## Facebook credentials va session

ScanFB gia dinh Facebook da duoc dang nhap san trong trinh duyet. App khong tu dang nhap, khong yeu cau username/password va khong trich xuat cookie/session de luu rieng.

Neu Facebook yeu cau dang nhap lai, CAPTCHA, checkpoint hoac xac minh, app phai dung fail-closed va huong dan nguoi dung xu ly thu cong.

## Network va cloud

MVP khong gui noi dung bai, author, group, cookie, token hoac lead len cloud service. Neu sau nay co tinh nang export/sync/cloud, phai co milestone rieng, threat model rieng va user consent ro rang.

SearchProfile khac sau MVP, neu co, van phai theo local-first va buyer-only boundaries cua ScanFB.

## Logs

Log chi dung de chan doan trang thai app va loi adapter. Log phai redact:

- Cookie.
- Token.
- Secret.
- Header nhay cam.
- Noi dung private khong can thiet.

## Blocklist va quyen rieng tu

Blocklist la local user preference. Dinh danh uu tien user ID, canonical profile URL va username; display name chi la thong tin phu. Nguoi dung phai xem va bo block duoc trong Settings.
