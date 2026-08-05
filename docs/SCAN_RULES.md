# Scan Rules

Tai lieu nay la nguon chuan tac cao nhat cho hanh vi scan, filter, SearchProfile, geographic classification, author filtering, blocklist, deduplication va reason codes.

## Nguon va batch

- Nguon chinh la Facebook group ma nguoi dung da tham gia va chu dong them vao danh sach theo doi.
- Khong gioi han tong so group luu trong app.
- Moi scan batch gom dung 5 group.
- Sau moi batch, app dung, tra ket qua va cho nguoi dung bam Scan batch tiep theo.
- Khong mo nhieu group dong thoi.
- Khong cuon vo han.
- Khong retry lien tuc.
- Marketplace la nguon phu va khong phai trong tam MVP.

## Time window

Timezone chuan: `Asia/Ho_Chi_Minh`.

Moi lan Scan chi xet bai:

- Duoc tao trong ngay hien tai.
- Tu 00:00 hom nay den dung thoi diem nguoi dung bam Scan.
- Khong co thoi diem tao sau thoi diem bat dau Scan.

Quy tac ngay:

- Qua ngay moi, khong quay lai tim hoac backfill bai ngay truoc.
- Lead da luu tu ngay truoc van duoc giu.
- Bai cu noi len do comment, reaction hoac activity moi khong duoc coi la bai moi.
- Phai dua vao thoi diem tao bai goc, khong dua vao thu tu feed.
- Co the dung thoi diem Scan thanh cong gan nhat trong cung ngay de giam doc lai, nhung phai co overlap nho de tranh bo sot do Facebook lam tron thoi gian.
- Duplicate trong overlap phai xu ly bang post ID hoac canonical URL.

Neu batch di qua 00:00:

- Dung phan ngay cu.
- Khong tiep tuc quet bai ngay cu.
- Ghi trang thai group chua hoan tat la `expired_at_day_boundary`.
- Ngay moi bat dau cua so moi tu 00:00.

## Geographic scope

Ba che do do nguoi dung chon thu cong cho tung lan Scan:

1. `hcm`: TP.HCM.
2. `non_hcm_vietnam`: Ngoai TP.HCM nhung trong Viet Nam.
3. `all_vietnam`: Toan Viet Nam.

Mac dinh moi ngay va moi lan mo app la `hcm`.

Che do mo rong:

- Chi kich hoat khi nguoi dung chu dong chon.
- Khong tu ghi de mac dinh TP.HCM.
- Van chi xet bai trong ngay hien tai.
- Foreign-location classification is deferred trong MVP; chua co foreign exclusion rules.
- Bai khong xac dinh duoc dia diem dua vao Can kiem tra.

### MVP geographic vocabulary

MVP geographic vocabulary la danh sach huu han, implementation-neutral. MVP khong co comprehensive city/province database. Moi mo rong vocabulary sau nay phai explicit va documented.

Approved HCM terms:

- HCM
- TPHCM
- TP.HCM
- Ho Chi Minh
- Sai Gon
- Saigon

District, ward va county names khong duoc support trong MVP; recognition cho cac cap dia danh nay deferred beyond MVP.

Approved outside-HCM Vietnam terms:

- Hà Nội
- Ha Noi
- Đà Nẵng
- Da Nang
- Cần Thơ
- Can Tho

Do not add more cities or provinces trong MVP.

Unknown geographic text:

- Bat ky post nao khong match approved HCM terms hoac approved outside-HCM Vietnam terms phai giu la geographically unknown va dua vao Can kiem tra.
- Unknown khong tuong duong foreign.
- Khong infer dia ly tu unmatched text, shipping-based location phrases, fuzzy matching, geocoding hoac accent stripping ngoai cac bien the co dau/khong dau da duoc liet ke explicit.

Foreign classification:

- Foreign-location classification is deferred.
- MVP khong define foreign country hoac city indicators.
- Chua add foreign exclusion rules.

## SearchProfile

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

Quy dinh:

- MVP chi cung cap built-in MacBook Search Profile.
- Chua co UI tao profile tuy y trong MVP.
- Chua co plugin system.
- Chua co profile marketplace danh cho seller.
- Moi SearchProfile cua ScanFB deu phuc vu buyer-intent.
- Rule chung ve thoi gian, geographic scope, author, blocklist va dedup khong thuoc rieng MacBook.
- Viec trich chip/RAM/SSD la logic chuyen biet cua MacBook profile, khong phai invariant chung cho moi san pham.

## Buyer intent

ScanFB chi giu bai the hien mot nguoi dang co nhu cau mua hoac tim mua san pham muc tieu cua SearchProfile dang bat. ScanFB khong co seller mode va khong tim nguoi can ban.

Voi built-in MacBook Search Profile, vi du buyer intent:

- can mua
- tim mua
- muon mua
- can tim
- co ai ban
- can may
- can MacBook gap

Bat buoc co target keyword hoac alias cua SearchProfile. Voi MacBook profile, bai phai co lien quan den MacBook, MacBook Pro hoac thong tin model/cau hinh ro rang lien quan den MacBook.

## Exclusion rules

Loai:

- Nguoi dang ban may.
- Shop quang cao ban may.
- Bai tuyen cong tac vien.
- Bai dich vu, ky gui hoac quang cao.
- Bai khong lien quan.
- Bai ngoai pham vi dia ly da chon.
- Bai cu.
- Bai tu account bi block.
- Bai tu nguoi dang an danh.
- Bai co buyer intent nhung khong chua target product cua SearchProfile.

Cau de nham:

- "Can tien nen ban MacBook" la bai ban, khong phai can mua.
- "Shop can thu mua MacBook" co the la shop mua lai de kinh doanh, khong mac dinh la lead khach ca nhan. Mac dinh dua vao Can kiem tra hoac loai theo rule ro rang neu co tin hieu shop/kinh doanh.

`excluded.seller_intent` chi la ly do loai bai ban khoi buyer search, khong phai ho tro seller mode.

## Author rules

Tu dong loai author:

- Anonymous.
- Thanh vien an danh.
- Nguoi tham gia an danh.
- Cac bien the tuong duong.
- Ten hien thi sau khi trim khong chua dau cach.

Quy tac khong co dau cach phai loai ca cac alias nhu:

- motivatedsalamander3113
- bubblyapricot6241
- productiveshark5829

Quyet dinh san pham: chap nhan kha nang bo sot nguoi dung that co ten mot tu.

Khong duoc tu mo hang loat profile trong luc Scan de xac minh account.

## Blocklist local

Moi lead co chuc nang `Bo qua account nay`.

Blocklist uu tien dinh danh theo:

1. Facebook user ID
2. Canonical profile URL
3. Username
4. Ten hien thi chi la thong tin phu

Khi da block:

- Bo qua account do tren moi group.
- Bo qua trong cac lan Scan tuong lai.
- Van cho phep xem va bo block trong Settings.
- Luu `reason`, `createdAt` va `note` tuy chon.

## Deduplication

Khong xoa mat bai trung. Phai gom thanh mot lead co nhieu nguon.

Cung mot nguoi dang cung nhu cau tai nhieu group:

- Hien thi mot lead.
- Giu tat ca source post, group va URL goc.
- Cho biet so group da dang.

Khong gop neu:

- Cung nguoi nhung la nhu cau thuc su khac.
- Noi dung, model, ngan sach, khu vuc hoac thoi diem cho thay mot lead moi.

Uu tien nhan dien:

1. Author Facebook ID
2. Profile URL hoac username
3. Noi dung da normalize
4. Model/cau hinh/ngan sach/khu vuc
5. Khoang thoi gian

Khong chi dua vao ten hien thi.

## Dry Run

MVP phai co Dry Run mac dinh bat:

- Bai bi loai van duoc luu local trong tab Da loai.
- Phai thay ly do bi loai.
- Nguoi dung co the phuc hoi hoac xac nhan rule.
- Khong xoa vinh vien raw record chi vi rule phan loai.

## Reason log

Moi quyet dinh giu, loai hoac can kiem tra phai co machine-readable reason codes va mo ta de hieu.

Reason code phai on dinh va deterministic. Doi ten reason code la thay doi contract va phai co migration hoac release note.

Vi du:

- `included.buyer_intent`
- `included.target_keyword`
- `included.location_hcm`
- `excluded.anonymous_author`
- `excluded.author_name_has_no_space`
- `excluded.blocked_author`
- `excluded.seller_intent`
- `excluded.dealer_or_shop`
- `excluded.target_keyword_missing`
- `excluded.outside_scope`
- `excluded.previous_day`
- `review.unknown_location`
- `review.ambiguous_buyer_intent`

## Batch state

Moi group trong batch co trang thai rieng:

- `pending`
- `running`
- `succeeded`
- `failed`
- `skipped`
- `expired_at_day_boundary`

Neu gap CAPTCHA, checkpoint, yeu cau dang nhap lai, trang xac minh, DOM khong nhan dien duoc, selector quan trong bi thieu hoac mat quyen truy cap group:

- Dung group hien tai hoac toan batch tuy muc do.
- Khong tu vuot qua.
- Khong retry vo han.
- Khong danh dau thanh cong gia.
- Hien thi huong dan de nguoi dung xu ly thu cong.
