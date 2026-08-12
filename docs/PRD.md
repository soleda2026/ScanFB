# PRD - ScanFB

## Muc tieu

ScanFB la app local tren macOS giup nguoi dung tim cac bai Facebook the hien nhu cau **can mua** trong cac group Facebook do nguoi dung phe duyet. Nguoi dung them moi group mot lan, ScanFB luu local va cho phep bat/tat group can theo doi. Automatic enumeration cua moi group ma tai khoan Facebook da tham gia khong phai yeu cau MVP.

MVP tap trung vao built-in MacBook Search Profile: tim nguoi can mua MacBook hoac MacBook Pro. Workflow scan theo batch, phan loai deterministic, gom trung lead va hien thi ket qua co ly do ro rang. Marketplace la nguon phu, khong phai trong tam MVP.

## Nguoi dung muc tieu

Nguoi dung la ca nhan hoac nhom ban MacBook can theo doi nhu cau mua trong cac Facebook group tai Viet Nam. Nguoi dung da dang nhap Facebook trong trinh duyet va tu thuc hien moi tuong tac voi nguoi dang bai.

## Nguyen tac san pham

- Local-first.
- Nguoi dung chu dong bam Scan.
- Khong tu dong tuong tac tren Facebook.
- Khong luu credentials Facebook.
- Ket qua phai giai thich duoc bang reason codes.
- Rule-based la critical classification path cua MVP.
- Dry Run mac dinh bat de nguoi dung xem bai bi loai va dieu chinh rule.
- ScanFB la buyer-only: khong co seller mode, khong tim nguoi can ban va khong co `LeadIntent` buyer/seller.
- MacBook la Search Profile dau tien va la profile duy nhat trong MVP.
- Kien truc khong khoa cung toan bo domain vao MacBook de sau nay co the them buyer Search Profile khac.

## Pham vi MVP

- Them mot lan cac Facebook group duoc nguoi dung phe duyet, luu local va cho phep bat/tat group can theo doi.
- Moi scan batch gom dung 5 group.
- Scan tung group lan luot, khong mo nhieu group dong thoi.
- Chi xet bai duoc tao trong ngay hien tai theo timezone `Asia/Ho_Chi_Minh`, tu 00:00 den thoi diem nguoi dung bam Scan.
- Phan loai buyer intent, target keyword, anonymous author, author name policy, blocklist, geo scope va duplicate.
- Gom nhieu source post thanh mot lead khi cung mot nhu cau.
- Hien thi ket qua sau moi batch.
- Luu local raw record, decision, lead, source va blocklist.
- Cho phep nguoi dung danh dau lead: `new`, `viewed`, `contacted`, `ignored`.

## Ngoai pham vi MVP

- Cloud backend.
- Account system.
- Multi-user.
- Subscription.
- AI model trong critical classification path.
- Tu dong like, comment, inbox, ket ban, dang bai hoac giao dich.
- Tu dong dang nhap Facebook.
- Tu vuot CAPTCHA, checkpoint hoac xac minh.
- Backfill bai cua ngay truoc.
- Cuon vo han hoac retry lien tuc.
- Seller mode, seller tab, seller mode selector.
- `SellerIntentClassifier`, `SellerLead`, `LeadIntent` buyer/seller hoac `SELLER_SCAN`.
- UI tao Search Profile tuy y.
- Plugin system.
- Profile marketplace danh cho seller.

## SearchProfile trong MVP

`SearchProfile` la cau hinh buyer search. Toi thieu gom:

- `id`
- `displayName`
- `targetKeywords`
- `keywordAliases`
- `buyerIntentPhrases`
- `exclusionPhrases`
- `productAttributeDefinitions` tuy chon
- associated watched groups hoac co che lien ket tuong duong
- `isEnabled`

MVP chi cung cap built-in MacBook Search Profile. Moi SearchProfile cua ScanFB deu phuc vu buyer-intent. Rule chung ve thoi gian, geographic scope, author, blocklist va dedup khong thuoc rieng MacBook. Viec trich chip/RAM/SSD la logic chuyen biet cua MacBook profile, khong phai invariant chung cho moi san pham.

Sau MVP co the them buyer Search Profile khac nhu iPhone, may anh hoac laptop, nhung chi sau khi MacBook profile hoat dong on dinh va moi profile moi co fixture/rule deterministic rieng. App tim nguoi can ban, neu co, la du an/app khac dua tren kinh nghiem hoac core co the tai su dung tu ScanFB; khong xay abstraction, enum hoac code seller trong ScanFB hien tai.

## Hanh vi scan

Moi explicit Scan action tu dong lay next five active enrolled groups theo internal persisted cursor, chay lan luot, dung va hien thi ket qua. Nguoi dung khong quan ly cursor va khong can bam mot action rieng de chuyen queue; lan Scan explicit tiep theo moi tiep tuc batch ke tiep. Enrollment chi ghi nhan group nguoi dung muon ScanFB scan; no khong chung minh current Facebook membership, access permission, Safari login validity hoac future post availability. Khi production execution ton tai, access failure phai thanh explicit group attempt/result thay vi am tham xoa enrolled group. Production scan execution van chua duoc implement; automatic joined-group discovery chi la optional future research, khong phai MVP dependency.

Do Phase 11B chua co production Facebook collector duoc approve, Phase 11C0 chon mot transitional input workflow cho moi one-group collection call: ScanFB-owned form nhan cac post field do nguoi dung cung cap ro rang va tao mot bounded prepared snapshot version 1. Group identity va capture time do ScanFB cung cap, con body, author va absolute creation time phai duoc nguoi dung nhap truc tiep; khong infer relative time hoac field bi thieu. Workflow nay chua duoc implement, khong mo khoa Phase 11B va khong thay doi invariant moi full scan batch gom dung 5 group; Phase 12 van so huu viec dieu phoi batch va cursor.

Nguoi dung chon che do dia ly cho tung lan Scan:

1. TP.HCM
2. Ngoai TP.HCM nhung trong Viet Nam
3. Toan Viet Nam

Mac dinh moi ngay va moi lan mo app la TP.HCM. Che do mo rong chi kich hoat khi nguoi dung chu dong chon.

Chi tiet chuan tac nam trong [SCAN_RULES.md](SCAN_RULES.md).

## Ket qua sau batch

Sau moi batch, app hien thi:

- So group thanh cong, loi va chua chay.
- So bai da xem xet.
- So bai co tu khoa.
- So lead phu hop.
- So source post duoc gom trung.
- So bai bi loai theo tung ly do.

Moi lead card gom:

- Diem phu hop.
- Thoi gian dang goc.
- Nguoi dang.
- Noi dung tom tat hoac excerpt.
- Thuoc tinh san pham theo Search Profile. Voi MacBook profile gom model, chip, RAM, SSD neu co.
- Ngan sach neu co.
- Khu vuc.
- Muc tin cay cua phan loai.
- So group da dang.
- Nut Mo bai.
- Da xem.
- Da lien he.
- Bo qua.
- Bo qua account nay.

## Thanh cong cua MVP

MVP duoc coi la thanh cong khi nguoi dung co the scan tung batch 5 group voi MacBook Search Profile, thay buyer lead phu hop trong ngay, thay ly do giu/loai/can kiem tra, gom trung source post dung va khong co automation Facebook ngoai hanh dong doc trang.
