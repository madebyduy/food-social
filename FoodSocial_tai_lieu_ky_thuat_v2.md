# FoodSocial — Tài liệu kỹ thuật đầy đủ (v2 — đã rà soát & bổ sung)

> Mạng xã hội review đồ ăn địa phương · Backend **Go thuần** (`net/http` + `database/sql`, không framework, không ORM) + **PostgreSQL** · Thiết kế lại theo chuẩn **mid-level**.

Tài liệu này gồm 7 phần: Tổng quan → **Quy tắc sản phẩm v1** → **Bảo mật & quyền riêng tư** → Kiến trúc & quy tắc code → Cơ sở dữ liệu → Đặc tả API → Lộ trình & quy trình. Đọc tuần tự từ trên xuống.

---

## Changelog bản rà soát này (so với bản trước)

Bản trước đã khá đầy đủ về kiến trúc/DB/API/roadmap, nhưng rà soát lại phát hiện các mảng còn thiếu hoặc chưa đủ chặt. Đây là danh sách đã bổ sung/sửa trong bản này:

**Phần mới hoàn toàn**

- **Quy tắc sản phẩm v1**: công khai hay riêng tư, bài bị xóa/ẩn thì comment/like/suggestion cũ hiển thị ra sao, tác giả có xem/khiếu nại được bài bị ẩn không, block/mute, place do user tạo có cần duyệt không, ai sửa được place, nhãn nội dung quảng cáo/tài trợ.
- **Bảo mật & quyền riêng tư (tài khoản, dữ liệu, pháp lý)**: xác thực email/SĐT trước khi đăng nhiều bài, quên/đặt lại mật khẩu, chính sách lưu giữ/xóa dữ liệu (IP, user agent, ảnh, log), xóa tài khoản, Privacy Policy/Terms/Community Guidelines.
- **Ma trận phân quyền xem nội dung** (guest / tác giả / admin) cho bài VISIBLE, HIDDEN_BY_COMMUNITY, HIDDEN_BY_ADMIN, DELETED.
- **Phụ lục Google Places API**: giới hạn về lưu trữ, hiển thị, attribution.

**Đã sửa vì chưa đủ an toàn/chặt chẽ**

- **Luồng upload ảnh**: đổi từ "client tự gửi `public_url` khi tạo post" sang luồng xác minh key tạm thời → server kiểm MIME/size/chủ sở hữu → chuyển trạng thái `usable` → post chỉ tham chiếu `media_id`. Thêm bước bỏ EXIF/GPS, quét file độc hại, báo cáo ảnh vi phạm.
- **Idempotency-Key**: bổ sung bảng `idempotency_keys` thật sự (lưu request hash + response), không chỉ là mô tả API.
- **`province_id` vs `place_id`**: ghi rõ luật ưu tiên khi đã có địa điểm.
- **Canonical place merge**: ghi rõ luật "luôn merge vào canonical, không merge dây chuyền" (chặn multi-hop).
- **`map_url`**: bỏ nhận URL tự do từ client, chỉ sinh từ toạ độ/`google_place_id` đã xác thực.
- **Rate limit**: ghi rõ giới hạn của in-memory limiter khi scale nhiều instance, và mốc chuyển sang Redis.
- **Full-text search**: đổi mô tả từ "full-text search tiếng Việt" (dễ hiểu lầm là hiểu ngữ nghĩa) thành đúng bản chất: "tìm kiếm không dấu cơ bản", ghi rõ semantic search/đồng nghĩa món ăn là việc của tương lai.
- **Quyền hạn Block/Report**: ghi rõ nên làm trước hệ thống xếp hạng feed nâng cao.

---

# FoodSocial — Mạng xã hội review đồ ăn địa phương

> Backend **Go thuần** (`net/http` + `database/sql`, không framework, không ORM) + **PostgreSQL**.
> Dự án được thiết kế lại theo chuẩn **mid-level**: layered architecture, interface-based repository, transaction đúng, concurrency an toàn, cursor pagination, tìm kiếm không dấu cơ bản (chưa phải semantic search tiếng Việt).

Tài liệu này là bản **chuẩn hóa và nâng cấp** từ đề bài gốc. Mục tiêu không chỉ là "làm chạy được", mà là viết code như một backend engineer đi làm thật: có ranh giới tầng rõ ràng, có test, có migration versioning, có quy trình release.

---

## Mục lục

1. **Tổng quan** — sản phẩm là gì, ràng buộc công nghệ, những gì đã nâng cấp so với đề gốc, nguyên tắc xuyên suốt.
2. **Quy tắc sản phẩm v1** [MỚI] — công khai/riêng tư, xử lý bài xóa/ẩn, block/mute, place approval, nhãn tài trợ, ma trận phân quyền.
3. **Bảo mật & quyền riêng tư** [MỚI] — xác thực email/SĐT, quên/đặt lại mật khẩu, xóa tài khoản, chính sách lưu giữ dữ liệu, văn bản pháp lý.
4. **Kiến trúc & Quy tắc code** — 3 tầng, DI thủ công, repository qua `Querier`, error/envelope/cursor, checklist code Go.
5. **Cơ sở dữ liệu** — schema đã nâng cấp + lý do từng quyết định, index, concurrency, migration order.
6. **Đặc tả API v1** — endpoint, envelope, phân quyền, luồng lõi (accept suggestion, vote).
7. **Lộ trình & Quy trình** — 7 giai đoạn + Definition of Done, testing, CI/CD, deploy, mobile → web.

Nếu chỉ muốn code ngay: đọc phần 4 → 5 → nhảy vào Giai đoạn 0 ở phần 7 (nhưng nên chốt phần 2–3 trước khi code Giai đoạn 2 trở đi, vì chúng ảnh hưởng schema).

| File / Phần                         | Nội dung                                                                                                                   |
| ----------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `README.md` (file này)              | Tổng quan, tính năng, những gì đã nâng cấp, tech stack, cách chạy                                                          |
| Phần 00a (Quy tắc sản phẩm)         | Vòng lặp sản phẩm, công khai/riêng tư, xử lý nội dung xóa/ẩn, block/mute, place approval, nhãn tài trợ, ma trận phân quyền |
| Phần 00b (Bảo mật & quyền riêng tư) | Xác thực email/SĐT, quên/đặt lại mật khẩu, xóa tài khoản, chính sách dữ liệu, văn bản pháp lý                              |
| Phần 01 (Kiến trúc)                 | Kiến trúc phân tầng, cấu trúc thư mục, các pattern (repository, service, DI thủ công), quy tắc code Go chuẩn               |
| Phần 02 (Cơ sở dữ liệu)             | Schema nâng cấp (đầy đủ + giải thích quyết định), migration strategy, index, concurrency                                   |
| Phần 03 (API)                       | Đặc tả API v1, response envelope, error contract, pagination, auth                                                         |
| Phần 04 (Lộ trình)                  | Lộ trình 7 giai đoạn, definition of done, testing, CI/CD, deploy, mobile-first workflow                                    |

Đọc theo thứ tự trên. Nếu bạn chỉ muốn bắt tay code ngay: đọc `00a` → `00b` (chốt quyết định sản phẩm) → `01` → `02` → nhảy vào Giai đoạn 1 ở `04`.

---

## 1. Sản phẩm là gì

Mạng xã hội chia sẻ trải nghiệm ăn uống, vận hành như Threads nhưng có một cơ chế đặc thù: **mỗi bài review có thể gắn với một địa điểm ăn uống**, và **địa điểm không bắt buộc phải biết ngay lúc đăng**.

Người khác có thể **đề xuất địa điểm** cho bài chưa rõ địa chỉ; chủ bài **xác nhận hoặc từ chối**. Khi được xác nhận, tất cả bài cùng địa điểm được gom vào một **trang địa điểm** chung.

Cơ chế đặc thù thứ hai: **vote độ tin cậy có trọng số** theo tuổi tài khoản — để chống seeding/review giả mà không phạt oan bài ít tương tác.

Triển khai đầu tiên tại **Hải Phòng**, nhưng data model hỗ trợ mở rộng nhiều tỉnh/thành ngay từ đầu.

**Thứ tự làm client: Mobile trước, Web sau.** Vì vậy API được thiết kế mobile-first (xem mục 4).

---

## 2. Ràng buộc công nghệ (giữ nguyên tinh thần đề gốc)

**Backend**

- Go thuần, `net/http`, `http.ServeMux` (Go 1.22+ đã hỗ trợ pattern routing `GET /api/v1/posts/{id}` — dùng luôn, không cần thư viện router).
- `database/sql` + driver `github.com/lib/pq` (hoặc `jackc/pgx` dạng `database/sql` — khuyến nghị `pgx/v5/stdlib`, xem lý do ở `02`).
- **Không** dùng Gin/Fiber/Echo. **Không** ORM. SQL viết tay 100%, luôn dùng parameter `$1, $2`.

**Thư viện ngoài — danh sách tối thiểu đã chốt**

```text
github.com/jackc/pgx/v5           # driver, dùng qua database/sql stdlib
golang.org/x/crypto/bcrypt        # hash mật khẩu
github.com/golang-jwt/jwt/v5      # optional — mình chọn session token trong DB thay vì JWT (lý do bên dưới)
github.com/golang-migrate/migrate/v4  # chạy migration (tool CLI, không phải ORM)
github.com/google/uuid            # request ID, idempotency key
```

**Chuẩn stdlib khai thác tối đa**: `log/slog` (structured logging, có sẵn Go 1.21+), `context`, `errors`, `net/http`, `encoding/json`, `crypto/rand`, `crypto/sha256`, `database/sql`, `time`.

> **Quyết định: Session token trong DB thay vì JWT.**
> JWT stateless nghe hấp dẫn nhưng không revoke được ngay (phải chờ hết hạn hoặc dựng blacklist — mà blacklist thì lại là state, mất điểm cộng của JWT). Với mạng xã hội có admin khóa tài khoản, logout mọi thiết bị, ban user — bạn **cần revoke tức thì**. Session token trong DB (`sessions` table) làm việc này bằng một `UPDATE ... SET revoked_at`. Đây cũng là luồng auth rõ ràng hơn để luyện. JWT giữ lại như optional cho access token ngắn hạn nếu sau này cần scale.

---

## 3. Những gì đã NÂNG CẤP so với đề gốc

Đây là phần quan trọng. Đề gốc đã tốt; dưới đây là các thay đổi để "hoàn thiện" và đạt chuẩn mid-level. Chi tiết kỹ thuật nằm trong các phần bên dưới.

### 3.1. Data & correctness

1. **`TIMESTAMP` → `TIMESTAMPTZ`**: đề gốc dùng `TIMESTAMP` (không timezone). Đây là bug kinh điển — server và client lệch giờ, so sánh `expires_at` sai. Toàn bộ đổi sang `TIMESTAMPTZ`, lưu UTC.
2. **Bổ sung bảng còn thiếu**: `wards` (phường/xã), `hashtags` + `post_hashtags`, `place_merge_history`, `admin_actions` (audit log). Đề gốc nhắc tới các khái niệm này nhưng không có bảng.
3. **Optimistic locking cho `posts`**: thêm cột `version`. Chống lost-update khi tác giả sửa bài từ 2 thiết bị.
4. **Counter đồng bộ đúng cách**: `like_count`, `comment_count`, `vote_count`... là denormalized. Đề gốc không nói cách giữ đồng bộ — đây chính là nguồn bug atomicity/cache. Mình chốt: **cập nhật counter trong cùng transaction** với thao tác gốc (không dùng trigger để giữ logic ở tầng application dễ test; có phương án trigger dự phòng trong `02`).
5. **Vote weight snapshot**: giữ nguyên `weight_at_vote` (đề gốc đã đúng), nhưng bổ sung rõ luồng **đổi vote** phải cập nhật lại aggregate trong transaction có `SELECT ... FOR UPDATE` trên post.
6. **Case-insensitive username/email**: dùng unique index trên `lower(...)` thay vì tin vào input.

### 3.2. Đọc dữ liệu / hiệu năng

7. **Cursor-based pagination** thay cho `OFFSET`. Offset chậm ở trang sâu và bị "nhảy dòng" khi có bài mới chèn vào — tệ cho feed mobile infinite-scroll. Dùng keyset `(created_at, id) < (cursor_created_at, cursor_id)`.
8. **Tìm kiếm không dấu cơ bản** (không gọi là "full-text search tiếng Việt hoàn chỉnh"): dùng `tsvector` + `unaccent` + GIN index thay vì `LIKE '%...%'`. Tìm "pho" ra cả "phở". Đây **chưa** phải hiểu ngữ nghĩa tiếng Việt, tên món đồng nghĩa, hay tự sửa lỗi chính tả — semantic search để làm sau, không phải cam kết của bản này.
9. **Partial index cho soft-delete**: `WHERE deleted_at IS NULL` và `WHERE status = 'VISIBLE'` để feed query nhanh.

### 3.3. An toàn & vận hành

10. **Rate limiting middleware** (token bucket in-memory cho MVP). Áp cho `register`, `login`, `create post`, `comment`, `vote`. **[NÂNG CẤP]** in-memory limiter chỉ đúng khi chạy **một instance duy nhất** — chạy N instance sau load balancer thì giới hạn thực tế bị nhân N lần. Ngay khi scale ngang (>1 instance), bắt buộc chuyển sang giới hạn tập trung: Redis (`INCR` + `EXPIRE`, hoặc Lua script token bucket) hoặc rate limit ở API gateway/CDN.
11. **Idempotency**: like/save dùng `INSERT ... ON CONFLICT DO NOTHING`; các thao tác tạo tài nguyên nhạy cảm nhận header `Idempotency-Key`.
12. **Media upload qua presigned URL** (S3/Cloudflare R2) thay vì đẩy file qua API Go. Mobile client upload thẳng lên storage — API chỉ lưu URL. Nhẹ server, hợp mobile.
13. **Graceful shutdown, server timeouts, body size limit, request ID, recovery, structured logging** — bộ middleware "người lớn".
14. **Outbox pattern cho notification** (mức nhẹ): notification ghi vào bảng trong cùng transaction nghiệp vụ; một worker đơn giản đẩy đi. MVP có thể đọc trực tiếp bảng, nhưng cấu trúc để không mất notification khi lỗi.
15. **API versioning** `/api/v1/...` ngay từ đầu.
16. **Admin audit log**: mọi hành động admin (ẩn bài, khóa user, gộp place) ghi vào `admin_actions`.

### 3.4. Chuẩn code

17. **Interface-based repository** (dependency inversion) → service test được bằng mock, không cần DB.
18. **Custom error type + HTTP status mapping** tập trung, không leak lỗi nội bộ.
19. **DTO/mapping layer** tách entity DB khỏi response API (không bao giờ trả `password_hash`, và có thể đổi shape API mà không đụng DB).
20. **Table-driven tests** (idiom Go) cho service và các hàm thuần như `CalculateVoteWeight`, `ShouldHidePost`.

---

## 4. Thiết kế mobile-first — vì sao ảnh hưởng tới API

Làm app mobile trước nghĩa là API phải chịu được mạng yếu, offline-friendly, và tiết kiệm pin/data:

- **Cursor pagination** (đã nói) → infinite scroll mượt, không trùng/mất bài.
- **Response envelope nhất quán** (`{ data, meta, error }`) → client parse một kiểu duy nhất.
- **`ETag` / `If-None-Match`** cho các GET tĩnh (profile, place) → tiết kiệm data.
- **Trả về đủ dữ liệu trong một call** cho màn feed (bài + tác giả + counters + trạng thái like/save/vote của người xem) → tránh N+1 request từ mobile. Dùng JOIN + subquery, không để client gọi thêm.
- **Push notification** để dành: MVP dùng bảng `notifications` + polling/GET; khi lên app thật thì gắn FCM/APNs qua worker đọc outbox. Data model không đổi.
- **Idempotency-Key** cho POST → mobile retry khi mất mạng không tạo trùng.

Web làm sau **dùng lại y hệt API v1** — chỉ khác client. Không cần backend riêng.

---

## 5. Chạy dự án (sẽ chi tiết ở Phần 04)

```bash
# 1. Khởi động Postgres (docker-compose có sẵn ở phần Lộ trình)
docker compose up -d db

# 2. Cấu hình
cp .env.example .env   # sửa DATABASE_URL, PORT, ...

# 3. Chạy migration
make migrate-up

# 4. Chạy server
make run        # hoặc: go run ./cmd/api

# 5. Test
make test
```

API sống ở `http://localhost:8080/api/v1`. Health check: `GET /healthz`.

---

## 6. Nguyên tắc xuyên suốt (đọc trước khi code)

1. **Không tin client.** User ID luôn lấy từ session, không lấy từ body/param. Ai sửa được tài nguyên → kiểm ở service, không ở client.
2. **Mọi thao tác nhiều bước = 1 transaction.** Vote, accept suggestion, follow+notify... phải all-or-nothing.
3. **Repository chỉ biết SQL, Service chỉ biết nghiệp vụ, Handler chỉ biết HTTP.** Không lẫn tầng. Handler không viết SQL. Service không đọc `*http.Request`.
4. **Lỗi thì wrap, đừng nuốt.** `fmt.Errorf("...: %w", err)`. Sentinel error ở tầng service để handler map ra HTTP status.
5. **Soft delete, không xóa vật lý** (trừ dữ liệu rác kỹ thuật).
6. **Migration chỉ tiến, không sửa file cũ.** Đã apply thì viết migration mới để đổi.

---

_Tiếp theo: phần Quy tắc sản phẩm v1 bên dưới._

---

# 00a — Quy tắc sản phẩm v1

Đây là phần **quyết định sản phẩm**, không phải kỹ thuật — nhưng thiếu phần này thì kỹ thuật ở dưới không biết implement thế nào cho đúng. Mọi quyết định ở đây nên chốt **trước khi** code Giai đoạn 2 trở đi, vì chúng ảnh hưởng trực tiếp tới schema và luồng API.

## 1. Vòng lặp sản phẩm cốt lõi (nhắc lại cho rõ)

```
Người dùng thấy một trải nghiệm ăn uống
  → đăng bài kèm ảnh/món/địa điểm (địa điểm có thể chưa biết)
  → người khác bình luận hoặc đề xuất địa điểm
  → chủ bài xác nhận địa điểm
  → địa điểm có thêm nội dung đáng tin để người khác khám phá
```

Đây là điểm khác biệt thực sự của FoodSocial so với "Threads có thêm ảnh đồ ăn": giá trị nằm ở **trải nghiệm ăn uống được cộng đồng xác thực và gom theo địa điểm**, không nằm ở việc feed đẹp hay nhiều tính năng. Mọi quyết định ưu tiên tính năng nên kiểm lại với vòng lặp này trước.

## 2. Công khai hay riêng tư?

- **v1: bài viết mặc định công khai** (🟢 guest xem được). Đây là mạng xã hội khám phá địa điểm — nội dung cần công khai để trang place có giá trị SEO/khám phá (đã nêu ở Phần E — Web).
- Không làm chế độ "chỉ người theo dõi" ở v1. Nếu sau này cần, thêm `posts.visibility` (`PUBLIC` | `FOLLOWERS_ONLY`) — nhưng đây là "future feature", không phải MVP.
- Profile cũng mặc định công khai (để trang place hiển thị được tác giả).

## 3. Bài bị xóa/ẩn: nội dung liên quan hiển thị thế nào?

| Tình huống                                | Comment/like cũ                       | Location suggestion cũ                                                    | Bài xuất hiện ở đâu                                                                                           |
| ----------------------------------------- | ------------------------------------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| Tác giả tự xóa (`DELETED_BY_AUTHOR`)      | Ẩn cùng bài (không truy cập được nữa) | Giữ nguyên trạng thái đã resolve trong DB (lịch sử), nhưng không hiển thị | Biến mất khỏi feed/place/search                                                                               |
| Cộng đồng vote ẩn (`HIDDEN_BY_COMMUNITY`) | Vẫn còn trong DB, ẩn theo bài         | Giữ nguyên                                                                | Biến mất khỏi feed/place/search công khai; **tác giả và admin vẫn xem được** (xem ma trận phân quyền ở mục 8) |
| Admin ẩn (`HIDDEN_BY_ADMIN`)              | Vẫn còn trong DB, ẩn theo bài         | Giữ nguyên                                                                | Biến mất khỏi feed/place/search công khai; **tác giả xem được kèm banner lý do**, có nút khiếu nại            |

Nguyên tắc chung: **không xóa vật lý** dữ liệu liên quan khi bài bị ẩn/xóa — chỉ ẩn theo trạng thái của bài cha, để admin khôi phục được nếu ẩn nhầm (đã có endpoint `restore` ở Phần 03).

## 4. Bài bị admin ẩn: tác giả xem được không? Khiếu nại được không?

- **Có, tác giả luôn xem được bài của chính mình** dù trạng thái gì, kèm banner giải thích ngắn (ví dụ "Bài bị ẩn do vi phạm cộng đồng — mã ADM-2201").
- Tác giả có nút "Khiếu nại" → tạo một `report` ngược với `target_type = POST`, `reason = 'APPEAL'`, admin xử lý qua hàng đợi report như bình thường (không cần bảng riêng).
- Bài bị **cộng đồng** vote ẩn (`HIDDEN_BY_COMMUNITY`) cũng cho khiếu nại tương tự — admin xem lại và có thể `restore`.

## 5. Block / Mute

- **Có, và nên làm trước hệ thống xếp hạng feed nâng cao.** Mạng xã hội có nội dung do người dùng tạo mà thiếu block sẽ rất mệt để vận hành ngay từ vài trăm user đầu tiên.
- Hai cơ chế tách biệt:
  - **Block**: hai chiều — A block B thì A không thấy bài/comment của B và ngược lại; B không follow/comment/vote được bài của A.
  - **Mute**: một chiều, chỉ ảnh hưởng gì A thấy trong feed của A; B không biết bị mute.
- Đưa vào roadmap ở **MVP-2** (ngay sau feed following), trước khi làm feed ranking/gợi ý thông minh.

## 6. Place do người dùng tạo: có cần duyệt không? Ai được sửa?

- **Place tạo mới không cần duyệt trước (pre-moderation)** — nếu bắt duyệt thì luồng "đề xuất → chấp nhận" sẽ bị chậm và cản trở đúng vòng lặp cốt lõi. Thay vào đó **hậu kiểm (post-moderation)**: report + admin xử lý như nội dung khác.
- Place trùng lặp được gom bằng `canonical_place_id` (đã có ở Phần 02) thay vì chặn tạo mới.
- **Ai sửa được thông tin place**: bất kỳ user đã đăng nhập nào cũng đề xuất sửa được (tên, địa chỉ, giờ mở cửa) qua một bảng `place_edit_suggestions` đơn giản (tương tự location_suggestions, để sau MVP-3); admin duyệt trước khi áp dụng, vì place là dữ liệu dùng chung nhiều bài.
- **Chủ quán** (chưa có xác thực danh tính chủ quán ở v1): có thể gửi yêu cầu sửa/xóa qua kênh report/hỗ trợ thủ công. Không xây flow "xác minh chủ quán" ở MVP — chi phí xác minh cao, để version sau.

## 7. Nội dung quảng cáo / seeding / review nhận tài trợ

- Thêm cột `posts.is_sponsored BOOLEAN DEFAULT FALSE` (đã phản ánh trong schema ở Phần 02, mục posts). Khi `TRUE`, feed/place hiển thị nhãn "Nội dung được tài trợ".
- v1 **tự khai báo** (tác giả tick khi đăng), không có xác minh tự động — nhưng vi phạm khai báo sai là một `reason` riêng trong `reports` để cộng đồng gắn cờ.
- Đây cũng là một tín hiệu đầu vào tốt cho cơ chế vote độ tin cậy sau này (bài tự nhận tài trợ có thể tính trọng số khác), nhưng để sau MVP.

## 8. Ma trận phân quyền xem nội dung (bổ sung cho Phần 03)

| Trạng thái bài        | Guest (🟢)                               | Tác giả                                     | Người dùng khác | Admin (🔴)                                                 |
| --------------------- | ---------------------------------------- | ------------------------------------------- | --------------- | ---------------------------------------------------------- |
| `VISIBLE`             | Xem đầy đủ                               | Xem đầy đủ                                  | Xem đầy đủ      | Xem đầy đủ                                                 |
| `HIDDEN_BY_COMMUNITY` | 404/không thấy trong feed, search, place | Xem đầy đủ + banner + nút khiếu nại         | Không thấy      | Xem đầy đủ + nút restore                                   |
| `HIDDEN_BY_ADMIN`     | 404/không thấy                           | Xem đầy đủ + banner lý do + nút khiếu nại   | Không thấy      | Xem đầy đủ + lý do + nút restore                           |
| `DELETED_BY_AUTHOR`   | 404                                      | 404 (đã xóa, không khôi phục qua UI thường) | 404             | Xem được qua admin panel (audit), không hiển thị công khai |

Áp dụng y hệt cho `comments` (thay `posts` bằng `comments`, không có "khiếu nại" ở v1 cho comment — chỉ ẩn/khôi phục bởi admin).

---

_Tiếp theo: phần Bảo mật & quyền riêng tư bên dưới._

---

# 00b — Bảo mật & quyền riêng tư (tài khoản, dữ liệu, pháp lý)

Phần backend/API đã tốt ở kỹ thuật (transaction, session revoke...), nhưng thiếu các luồng tài khoản thật và chính sách dữ liệu. Bổ sung ở đây; chi tiết endpoint tương ứng nằm ở Phần 03 (đã cập nhật).

## 1. Xác thực email/số điện thoại

- Sau khi `register`, tài khoản ở trạng thái `status = 'ACTIVE'` nhưng thêm cờ `email_verified_at TIMESTAMPTZ NULL` trên `users`.
- Cho phép đăng **tối đa 1 bài** khi chưa verify (đủ để dùng thử), giới hạn còn lại (comment, vote, đăng thêm bài, upload nhiều ảnh) yêu cầu verify — chống spam đăng ký hàng loạt.
- Luồng: gửi OTP/email link → `POST /api/v1/auth/verify` với `code` → set `email_verified_at = now()`.

## 2. Quên / đặt lại mật khẩu

- `POST /api/v1/auth/forgot-password` → nhận `email`, luôn trả `200` chung chung (không tiết lộ email có tồn tại hay không), sinh token một lần dùng, hạn 15–30 phút, lưu **hash** của token (giống session token) trong bảng `password_reset_tokens (user_id, token_hash, expires_at, used_at)`.
- `POST /api/v1/auth/reset-password` → nhận `token`, `new_password` → verify hash + hạn + chưa dùng → đổi `password_hash`, đánh dấu `used_at`, và **revoke toàn bộ session hiện có** của user (buộc đăng nhập lại ở mọi thiết bị) — đây là bước hay bị quên nhưng quan trọng về bảo mật.

## 3. Quản lý thiết bị đăng nhập

- Đã có `GET /auth/sessions` và `DELETE /auth/sessions/{id}` ở bản trước — giữ nguyên.
- Bổ sung: `DELETE /api/v1/auth/sessions` (không kèm id) = **đăng xuất toàn bộ thiết bị trừ thiết bị hiện tại**, dùng khi nghi ngờ lộ tài khoản.

## 4. Xóa tài khoản

- Chốt rõ: **xóa mềm trước, xóa cứng sau 30 ngày**. `DELETE /api/v1/users/me` → set `users.status='DELETED'`, `deleted_at=now()`, revoke toàn bộ session, ẩn toàn bộ bài/comment công khai (không xóa dữ liệu ngay).
- Một job định kỳ (cron/worker) sau 30 ngày: xóa/anonymize thật (xóa `email`, `phone`, `password_hash`, thay `display_name` thành "Người dùng đã xóa") — giữ lại `id` để không vỡ khóa ngoại của bài/comment cũ (bài vẫn hiện tác giả là "Người dùng đã xóa" thay vì biến mất hoàn toàn).
- Trong 30 ngày đó, user đăng nhập lại được thì **khôi phục** tài khoản (hủy lịch xóa cứng).

## 5. Chính sách lưu giữ / xóa dữ liệu

| Loại dữ liệu                                      | Thời gian lưu                                                 | Ghi chú                                                        |
| ------------------------------------------------- | ------------------------------------------------------------- | -------------------------------------------------------------- |
| `sessions.ip`, `user_agent`                       | Xóa khi session hết hạn/bị revoke quá 90 ngày                 | Chỉ dùng để hiển thị "thiết bị đăng nhập" và điều tra lạm dụng |
| Log request (`request_id`, path, status, latency) | 30–90 ngày tùy hạ tầng log                                    | Không log body chứa password/token                             |
| Ảnh đã xóa (`post_images` của bài đã xóa)         | Xóa khỏi storage sau 30 ngày cùng lịch xóa cứng tài khoản/bài | Tránh giữ ảnh nhạy cảm vô thời hạn                             |
| Dữ liệu vị trí trong ảnh (EXIF GPS)               | Không lưu — xem mục Upload ảnh                                | —                                                              |

## 6. Văn bản pháp lý cần có trước khi mở công khai

- **Privacy Policy**: nêu rõ thu thập gì (email/phone, IP, user agent, ảnh, vị trí bài viết), dùng để làm gì, giữ bao lâu, chia sẻ với ai (nếu dùng Google Places/CDN bên thứ ba phải nêu tên).
- **Terms of Use**: quyền/nghĩa vụ user, điều kiện khóa/xóa tài khoản.
- **Community Guidelines**: định nghĩa nội dung vi phạm (dùng chung cho report reasons ở Phần 02).
- **Quy trình khiếu nại moderation**: đã có ở mục 4 Phần 00a (nút khiếu nại + report reason `APPEAL`).
- Vì hệ thống thu thập IP, user agent, ảnh và vị trí — đây là dữ liệu cá nhân theo hầu hết khung pháp lý hiện nay. **Trước khi mở công khai nên có người am hiểu pháp lý rà lại**, đặc biệt nếu có người dùng ở nhiều tỉnh/quốc gia. Tài liệu này không thay thế tư vấn pháp lý.
- **Không tự thu thập giấy tờ tùy thân ở MVP** (không cần xác minh danh tính để đăng ký) — giảm rủi ro pháp lý và vận hành khi còn nhỏ.

---

_Tiếp theo: phần Kiến trúc & Quy tắc code bên dưới._

---

# 01 — Kiến trúc & Quy tắc code

Đây là file quyết định "code trông thế nào". Junior viết code chạy được; mid-level viết code **có ranh giới rõ ràng, test được, và người khác đọc hiểu ngay**. Toàn bộ các quyết định dưới đây phục vụ mục tiêu đó.

---

## 1. Kiến trúc phân tầng (Layered / Clean-lite)

Ba tầng, phụ thuộc chỉ đi **một chiều** từ ngoài vào trong:

```
HTTP Handler   →   Service        →   Repository      →   PostgreSQL
(biết HTTP)        (biết nghiệp vụ)    (biết SQL)
```

- **Handler**: parse request, validate cú pháp, gọi service, map kết quả/lỗi ra HTTP + JSON. **Không** chứa nghiệp vụ, **không** viết SQL.
- **Service**: chứa toàn bộ business logic và **quyết định transaction**. Không biết `http.Request`, không biết JSON. Nhận và trả struct thuần Go.
- **Repository**: chỉ đọc/ghi DB bằng SQL viết tay. Không chứa nghiệp vụ (không quyết định "có được vote không").

Quy tắc vàng: **tầng trong không được import tầng ngoài.** Repository không biết service tồn tại. Service không biết handler tồn tại.

### 1.1. Vì sao tách như vậy (không phải cho đẹp)

- **Test không cần DB**: service phụ thuộc vào _interface_ repository, nên test service bằng mock in-memory. Chạy trong mili-giây.
- **Đổi được từng phần**: đổi Postgres sang cái khác chỉ đụng repository. Đổi shape JSON chỉ đụng handler.
- **Nghiệp vụ ở một chỗ**: khi bug "vote sai trọng số", bạn biết chính xác nhìn vào đâu — service, không phải lục 3 nơi.

---

## 2. Dependency Injection thủ công (không dùng thư viện DI)

Wire mọi thứ ở `main.go`. Từ trên xuống: mở DB → tạo repository → tiêm vào service → tiêm vào handler → gắn route. Đây là "manual DI" — rõ ràng, không magic.

```go
// cmd/api/main.go
func main() {
    cfg := config.Load()

    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

    db, err := database.Connect(cfg.DatabaseURL)
    if err != nil {
        logger.Error("db connect failed", "err", err)
        os.Exit(1)
    }
    defer db.Close()

    // --- Repositories (trả về interface) ---
    userRepo := user.NewRepository(db)
    postRepo := post.NewRepository(db)
    voteRepo := vote.NewRepository(db)
    // ...

    // --- Services (nhận repo + db để mở transaction) ---
    authSvc := auth.NewService(userRepo, sessionRepo, cfg)
    postSvc := post.NewService(db, postRepo, notifRepo, logger)
    voteSvc := vote.NewService(db, voteRepo, postRepo, logger)

    // --- Handlers ---
    authH := auth.NewHandler(authSvc)
    postH := post.NewHandler(postSvc)
    voteH := vote.NewHandler(voteSvc)

    // --- Router + middleware chain ---
    mux := router.New(authH, postH, voteH /* ... */, authSvc)
    handler := middleware.Chain(mux,
        middleware.Recovery(logger),
        middleware.RequestID(),
        middleware.Logging(logger),
        middleware.RateLimit(cfg.RateLimit),
    )

    srv := &http.Server{
        Addr:              cfg.Addr,
        Handler:           handler,
        ReadHeaderTimeout: 5 * time.Second,
        ReadTimeout:       15 * time.Second,
        WriteTimeout:      15 * time.Second,
        IdleTimeout:       60 * time.Second,
        MaxHeaderBytes:    1 << 20, // 1 MB
    }

    runWithGracefulShutdown(srv, logger) // xem mục 9
}
```

> **"OOP trong Go"** không phải class/inheritance. Go làm OOP qua **struct + method + interface + composition**. "Kế thừa" thay bằng **embedding**. Encapsulation thay bằng **package boundary + chữ thường/hoa** (identifier viết hoa = public, thường = private). Polymorphism = **interface**. Toàn bộ tài liệu này theo triết lý đó — đừng cố bê Java/NestJS class hierarchy vào Go.

---

## 3. Cấu trúc thư mục

Mở rộng từ đề gốc, thêm tầng chuẩn hóa:

```text
food-social/
├── cmd/
│   └── api/
│       └── main.go                 # entrypoint, wire DI, graceful shutdown
├── internal/                       # code riêng, không cho project khác import
│   ├── config/
│   │   └── config.go               # load env → struct, validate
│   ├── database/
│   │   ├── postgres.go             # Connect(), pool config
│   │   └── tx.go                   # helper WithTx (mục 7)
│   ├── router/
│   │   └── router.go               # ServeMux, gắn tất cả route
│   ├── middleware/
│   │   ├── chain.go                # Chain(h, mws...)
│   │   ├── auth.go                 # xác thực session → nhét user vào context
│   │   ├── recovery.go             # recover panic → 500
│   │   ├── logging.go              # log request + latency
│   │   ├── requestid.go
│   │   └── ratelimit.go            # token bucket
│   ├── httpx/                      # tiện ích HTTP dùng chung
│   │   ├── response.go             # WriteJSON, envelope
│   │   ├── errors.go               # map AppError → status code
│   │   ├── decode.go               # decode + limit body + validate
│   │   └── cursor.go               # encode/decode cursor pagination
│   ├── apperr/
│   │   └── apperr.go               # AppError, sentinel errors
│   ├── auth/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go           # sessions
│   │   ├── password.go             # bcrypt hash/verify
│   │   └── service_test.go
│   ├── user/
│   │   ├── handler.go  service.go  repository.go  model.go  dto.go  *_test.go
│   ├── post/
│   ├── comment/
│   ├── vote/
│   ├── like/
│   ├── save/
│   ├── follow/
│   ├── place/
│   ├── suggestion/
│   ├── feed/
│   ├── report/
│   ├── notification/
│   ├── admin/
│   └── platform/                   # thứ hạ tầng: rate limiter, id gen, clock
│       ├── ratelimiter.go
│       └── clock.go                # interface Clock để test time-based logic
├── migrations/
│   ├── 000001_create_users.up.sql / .down.sql
│   ├── 000002_create_sessions.up.sql / .down.sql
│   ├── ...                          # đánh số 6 chữ số cho golang-migrate
├── test/
│   └── integration/                # test chạm DB thật (build tag `integration`)
├── scripts/
│   └── seed.sql                    # seed provinces/districts Hải Phòng
├── deployments/
│   ├── docker-compose.yml
│   └── Dockerfile
├── Makefile
├── .env.example
├── go.mod
└── README.md
```

**Vì sao mỗi feature là một package** (`post/`, `vote/`...): mỗi package tự chứa handler+service+repo của nó (vertical slice). Tìm code theo tính năng, không phải theo tầng. Khi module lớn lên, không bị 200 file dồn trong `handlers/`.

**Vì sao `internal/`**: Go cấm package ngoài import bất cứ thứ gì dưới `internal/`. Đây là encapsulation ở mức module — API nội bộ không rò ra ngoài.

---

## 4. Repository pattern với interface

Interface **được khai báo ở nơi tiêu thụ** (package service), không ở nơi implement — đây là idiom Go ("accept interfaces, return structs"). Nhưng để gọn cho dự án này, đặt interface ngay trong package feature, cạnh service.

```go
// internal/post/repository.go
package post

type Repository interface {
    Create(ctx context.Context, tx database.Querier, p *Post) error
    GetByID(ctx context.Context, q database.Querier, id int64) (*Post, error)
    ListFeed(ctx context.Context, q database.Querier, cur Cursor, limit int) ([]Post, error)
    UpdateContent(ctx context.Context, q database.Querier, p *Post) error // dùng version optimistic lock
    SoftDelete(ctx context.Context, q database.Querier, id, authorID int64) error
    IncrementCounter(ctx context.Context, q database.Querier, id int64, col string, delta int) error
}

// implement
type repository struct{} // stateless, nhận Querier từ ngoài

func NewRepository(_ *sql.DB) Repository { return &repository{} }
```

> **Điểm mid-level quan trọng — `database.Querier`:** repository method **không** giữ `*sql.DB` bên trong. Thay vào đó nhận một `Querier` (interface có `QueryContext/ExecContext/QueryRowContext`) từ ngoài. Cả `*sql.DB` **và** `*sql.Tx` đều thỏa interface này. Nhờ đó **cùng một hàm repo chạy được cả trong và ngoài transaction** — service quyết định. Đây là cách giải bài toán "transaction xuyên nhiều repository" mà không rối.

```go
// internal/database/tx.go
package database

type Querier interface {
    ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
    QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
}

// *sql.DB và *sql.Tx đều đã thỏa Querier sẵn.

// WithTx: mở transaction, tự rollback nếu panic/err, commit nếu ok.
func WithTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
    tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback()
            panic(p) // ném lại để middleware recovery bắt
        }
    }()
    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("rollback: %v (original: %w)", rbErr, err)
        }
        return err
    }
    return tx.Commit()
}
```

---

## 5. Service layer — nơi ở của nghiệp vụ

Service cầm `*sql.DB` (để mở transaction) và các repository interface.

```go
// internal/vote/service.go
package vote

type Service struct {
    db       *sql.DB
    votes    Repository
    posts    post.ReaderWriter // interface con: chỉ những method vote cần
    clock    platform.Clock
    log      *slog.Logger
}

func NewService(db *sql.DB, votes Repository, posts post.ReaderWriter, log *slog.Logger) *Service {
    return &Service{db: db, votes: votes, posts: posts, clock: platform.SystemClock{}, log: log}
}

// CastVote: toàn bộ luồng vote nằm gọn trong 1 transaction.
func (s *Service) CastVote(ctx context.Context, actorID, postID int64, voteType VoteType) (*Summary, error) {
    if !voteType.Valid() {
        return nil, apperr.BadRequest("vote_type không hợp lệ")
    }

    var summary *Summary
    err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
        // 1. Khóa post row để chống race khi tính lại aggregate
        p, err := s.posts.GetForUpdate(ctx, tx, postID)
        if err != nil {
            return err // repo trả apperr.ErrNotFound nếu không thấy
        }
        // 2. Nghiệp vụ: bài phải VISIBLE, không tự vote bài mình
        if p.Status != post.StatusVisible {
            return apperr.Conflict("bài không ở trạng thái cho phép vote")
        }
        if p.AuthorID == actorID {
            return apperr.Forbidden("không thể tự vote bài của mình")
        }
        // 3. Tính trọng số theo tuổi tài khoản (hàm thuần, test riêng)
        actor, err := s.posts.GetUserAge(ctx, tx, actorID)
        if err != nil {
            return err
        }
        weight := CalculateVoteWeight(actor.CreatedAt, s.clock.Now())

        // 4. Upsert vote (đổi vote thì update weight_at_vote mới)
        if err := s.votes.Upsert(ctx, tx, postID, actorID, voteType, weight); err != nil {
            return err
        }
        // 5. Tính lại aggregate từ bảng votes (nguồn sự thật)
        agg, err := s.votes.Aggregate(ctx, tx, postID)
        if err != nil {
            return err
        }
        // 6. Ghi aggregate vào posts (denormalized)
        if err := s.posts.UpdateVoteStats(ctx, tx, postID, agg); err != nil {
            return err
        }
        // 7. Đủ điều kiện thì ẩn bài (hàm thuần, test riêng)
        if ShouldHidePost(agg.TotalVoters, agg.TrustedWeight, agg.UntrustedWeight) {
            if err := s.posts.HideByCommunity(ctx, tx, postID, s.clock.Now()); err != nil {
                return err
            }
        }
        summary = agg.ToSummary()
        return nil
    })
    return summary, err
}
```

**Hàm thuần tách riêng** — không đụng DB, không đụng context → dễ test nhất:

```go
// internal/vote/rules.go
func CalculateVoteWeight(accountCreatedAt, now time.Time) float64 {
    age := now.Sub(accountCreatedAt)
    switch {
    case age < 7*24*time.Hour:
        return 0.25
    case age < 30*24*time.Hour:
        return 0.5
    default:
        return 1.0
    }
}

func ShouldHidePost(totalVoters int, trustedW, untrustedW float64) bool {
    if totalVoters < 200 {
        return false
    }
    total := trustedW + untrustedW
    if total == 0 {
        return false
    }
    return untrustedW/total > 0.70
}
```

---

## 6. Handler layer — chỉ HTTP

```go
// internal/vote/handler.go
package vote

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// PUT /api/v1/posts/{id}/vote
func (h *Handler) Cast(w http.ResponseWriter, r *http.Request) {
    actorID := middleware.UserID(r.Context()) // lấy từ context, KHÔNG từ body

    postID, err := httpx.PathInt64(r, "id")
    if err != nil {
        httpx.Error(w, apperr.BadRequest("id không hợp lệ"))
        return
    }

    var req struct {
        VoteType string `json:"vote_type"`
    }
    if err := httpx.DecodeJSON(w, r, &req); err != nil { // đã giới hạn body size
        httpx.Error(w, err)
        return
    }

    summary, err := h.svc.CastVote(r.Context(), actorID, postID, VoteType(req.VoteType))
    if err != nil {
        httpx.Error(w, err) // map apperr → status code tự động
        return
    }
    httpx.OK(w, summary)
}
```

Handler ngắn, nhàm chán, giống nhau — **đó là dấu hiệu tốt**. Mọi thứ thú vị nằm ở service.

---

## 7. Xử lý lỗi — chuẩn hóa toàn hệ thống

Một kiểu lỗi ứng dụng duy nhất, mang sẵn "muốn trả HTTP status nào":

```go
// internal/apperr/apperr.go
package apperr

type Kind int

const (
    KindInternal Kind = iota
    KindBadRequest
    KindUnauthorized
    KindForbidden
    KindNotFound
    KindConflict
    KindTooMany
)

type AppError struct {
    Kind    Kind
    Message string // an toàn để hiện cho user
    Err     error  // lỗi gốc, chỉ để log, KHÔNG trả về client
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

// Constructor tiện dụng
func NotFound(msg string) *AppError   { return &AppError{Kind: KindNotFound, Message: msg} }
func Forbidden(msg string) *AppError  { return &AppError{Kind: KindForbidden, Message: msg} }
func Conflict(msg string) *AppError   { return &AppError{Kind: KindConflict, Message: msg} }
func BadRequest(msg string) *AppError { return &AppError{Kind: KindBadRequest, Message: msg} }

// Wrap lỗi hạ tầng thành internal (giấu chi tiết)
func Internal(err error) *AppError {
    return &AppError{Kind: KindInternal, Message: "lỗi hệ thống", Err: err}
}

// Sentinel dùng ở repository
var ErrNotFound = NotFound("không tìm thấy")
```

Map ra HTTP tập trung — chỉ một nơi:

```go
// internal/httpx/errors.go
func Error(w http.ResponseWriter, err error) {
    var appErr *apperr.AppError
    if !errors.As(err, &appErr) {
        appErr = apperr.Internal(err) // lỗi lạ → 500, không leak
    }
    status := map[apperr.Kind]int{
        apperr.KindBadRequest:   http.StatusBadRequest,
        apperr.KindUnauthorized: http.StatusUnauthorized,
        apperr.KindForbidden:    http.StatusForbidden,
        apperr.KindNotFound:     http.StatusNotFound,
        apperr.KindConflict:     http.StatusConflict,
        apperr.KindTooMany:      http.StatusTooManyRequests,
        apperr.KindInternal:     http.StatusInternalServerError,
    }[appErr.Kind]

    writeJSON(w, status, envelope{Error: &apiError{
        Code:    appErr.Kind.String(),
        Message: appErr.Message,
    }})
}
```

**Nguyên tắc**: repository trả sentinel/wrap lỗi DB. Service diễn giải thành `apperr` nghiệp vụ. Handler chỉ gọi `httpx.Error`. Lỗi `%w`-wrap suốt chặng để `errors.As` bóc được, nhưng **client không bao giờ thấy chi tiết SQL/internal**.

---

## 8. Response envelope & pagination (mobile-first)

Một shape JSON duy nhất cho toàn API:

```go
// internal/httpx/response.go
type envelope struct {
    Data  any       `json:"data,omitempty"`
    Meta  *meta     `json:"meta,omitempty"`
    Error *apiError `json:"error,omitempty"`
}

type meta struct {
    NextCursor string `json:"next_cursor,omitempty"` // rỗng = hết trang
    Count      int    `json:"count"`
}
```

**Cursor pagination** — mã hóa `(created_at, id)` của bản ghi cuối thành chuỗi base64 opaque:

```go
// internal/httpx/cursor.go
type Cursor struct {
    CreatedAt time.Time
    ID        int64
}

func EncodeCursor(c Cursor) string {
    raw := fmt.Sprintf("%d|%d", c.CreatedAt.UnixNano(), c.ID)
    return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(s string) (Cursor, error) { /* parse ngược, lỗi → BadRequest */ }
```

Query keyset trong repository (nhanh cả ở trang sâu, không nhảy dòng):

```sql
SELECT id, user_id, content, created_at /* ... */
FROM posts
WHERE status = 'VISIBLE'
  AND (created_at, id) < ($1, $2)   -- $1,$2 = cursor; lần đầu dùng 'infinity'
ORDER BY created_at DESC, id DESC
LIMIT $3;
```

---

## 9. Middleware, context, graceful shutdown

**User vào context** (không truyền qua tham số lằng nhằng, không lấy từ body):

```go
// internal/middleware/auth.go
type ctxKey int
const userIDKey ctxKey = iota

func Authenticate(authSvc *auth.Service) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearer(r)
            uid, err := authSvc.ResolveSession(r.Context(), token)
            if err != nil {
                httpx.Error(w, apperr.Unauthorized("phiên không hợp lệ"))
                return
            }
            ctx := context.WithValue(r.Context(), userIDKey, uid)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func UserID(ctx context.Context) int64 {
    uid, _ := ctx.Value(userIDKey).(int64)
    return uid
}
```

Route công khai (guest xem được) không gắn `Authenticate`; route cần đăng nhập thì có. Route admin thêm một middleware `RequireRole("ADMIN")`.

**Graceful shutdown** — không cắt request đang chạy:

```go
func runWithGracefulShutdown(srv *http.Server, log *slog.Logger) {
    go func() {
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Error("server error", "err", err)
            os.Exit(1)
        }
    }()
    log.Info("server started", "addr", srv.Addr)

    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Error("graceful shutdown failed", "err", err)
    }
    log.Info("server stopped")
}
```

---

## 10. Quy tắc code Go — checklist review

Áp cho mọi PR. Đây là ranh giới junior/mid.

**Đặt tên & package**

- Package tên ngắn, số ít, không `utils`/`common` bừa bãi (`httpx`, `apperr` ok vì có nghĩa).
- Public (viết Hoa) chỉ khi thật sự cần dùng ngoài package. Mặc định để thường.
- Không stutter: trong package `post`, đặt `post.Service` không phải `post.PostService`.

**Error**

- Luôn xử lý `err`, không `_ = err` trừ khi có lý do ghi rõ.
- Wrap khi thêm ngữ cảnh: `fmt.Errorf("create post: %w", err)`.
- So sánh lỗi bằng `errors.Is` / bóc bằng `errors.As`, không so sánh chuỗi.
- Không panic trong luồng bình thường; panic chỉ cho lỗi lập trình, và có `Recovery` middleware chặn.

**Context**

- Mọi hàm chạm I/O (DB, HTTP client) nhận `ctx context.Context` là **tham số đầu tiên**.
- Truyền `r.Context()` xuống tới tận query. Không tạo `context.Background()` giữa chừng (trừ shutdown/worker).

**Concurrency**

- Không share mutable state không khóa. Rate limiter dùng `sync.Mutex`/`sync.Map`.
- Transaction: một `*sql.Tx` **không** dùng song song ở nhiều goroutine.

**SQL**

- 100% parameterized (`$1,$2`), không `fmt.Sprintf` giá trị vào query.
- `rows.Close()` bằng `defer` ngay sau khi query. Kiểm `rows.Err()` sau vòng lặp.
- `QueryRow(...).Scan(...)` phân biệt `sql.ErrNoRows` → trả `apperr.ErrNotFound`.

**Struct & interface**

- Nhận interface, trả struct.
- Interface nhỏ (1–3 method). Định nghĩa ở phía dùng.
- Không dùng embedding để "kế thừa" cho vui; chỉ khi thật sự compose hành vi.

**Test**

- Table-driven cho hàm thuần (`CalculateVoteWeight`, `ShouldHidePost`, cursor encode/decode).
- Service test dùng mock repository (implement interface).
- `platform.Clock` interface để test logic phụ thuộc thời gian mà không sleep.

```go
// Ví dụ table-driven test
func TestCalculateVoteWeight(t *testing.T) {
    now := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
    tests := []struct {
        name    string
        created time.Time
        want    float64
    }{
        {"tài khoản 5 ngày", now.Add(-5 * 24 * time.Hour), 0.25},
        {"tài khoản 7 ngày chẵn", now.Add(-7 * 24 * time.Hour), 0.5},
        {"tài khoản 29 ngày", now.Add(-29 * 24 * time.Hour), 0.5},
        {"tài khoản 30 ngày", now.Add(-30 * 24 * time.Hour), 1.0},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := CalculateVoteWeight(tt.created, now); got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

## 11. Config & clock (những thứ nhỏ nhưng lộ trình độ)

```go
// internal/config/config.go
type Config struct {
    Addr        string
    DatabaseURL string
    LogLevel    slog.Level
    RateLimit   RateLimitConfig
    SessionTTL  time.Duration
}

func Load() Config {
    return Config{
        Addr:        env("ADDR", ":8080"),
        DatabaseURL: mustEnv("DATABASE_URL"), // fail-fast nếu thiếu
        SessionTTL:  envDuration("SESSION_TTL", 30*24*time.Hour),
        // ...
    }
}
```

```go
// internal/platform/clock.go — để test time-based logic
type Clock interface{ Now() time.Time }
type SystemClock struct{}
func (SystemClock) Now() time.Time { return time.Now().UTC() }
```

Lưu ý: **luôn `.UTC()`** khi lấy thời gian ghi DB, kết hợp `TIMESTAMPTZ`. Đây là cách triệt tiêu cả lớp bug timezone.

---

_Tiếp theo: phần Cơ sở dữ liệu bên dưới._

---

# 02 — Thiết kế cơ sở dữ liệu (đã nâng cấp)

Schema dưới đây là bản chuẩn hóa từ đề gốc. Mỗi thay đổi quan trọng có ghi **[NÂNG CẤP]** kèm lý do. Migration đánh số theo chuẩn `golang-migrate`: mỗi bước có `.up.sql` và `.down.sql`.

**Quy ước chung áp cho mọi bảng:**

- `TIMESTAMPTZ` cho mọi cột thời gian (không dùng `TIMESTAMP`). Ứng dụng luôn ghi UTC.
- Soft delete: cột `deleted_at TIMESTAMPTZ NULL`, không xóa vật lý.
- Khóa ngoại luôn có index (Postgres **không** tự tạo index cho FK).
- Enum lưu dạng `VARCHAR` + `CHECK` constraint (đơn giản, dễ migrate hơn native enum).

---

## Extension cần bật (migration đầu tiên)

```sql
-- 000001_extensions.up.sql
CREATE EXTENSION IF NOT EXISTS unaccent;  -- bỏ dấu tiếng Việt cho full-text search
CREATE EXTENSION IF NOT EXISTS pg_trgm;   -- similarity cho gợi ý place trùng
```

---

## 1. users

```sql
CREATE TABLE users (
    id             BIGSERIAL PRIMARY KEY,
    username       VARCHAR(50)  NOT NULL,
    email          VARCHAR(255) NOT NULL,
    phone          VARCHAR(20),                          -- [NÂNG CẤP] đề gốc cho đăng ký bằng SĐT
    password_hash  VARCHAR(255) NOT NULL,
    display_name   VARCHAR(100) NOT NULL,
    avatar_url     TEXT,
    bio            TEXT,
    role           VARCHAR(20)  NOT NULL DEFAULT 'USER',
    status         VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    email_verified_at TIMESTAMPTZ,                          -- [NÂNG CẤP] xem Phần 00b — Bảo mật
    follower_count  INTEGER NOT NULL DEFAULT 0,           -- [NÂNG CẤP] denormalized
    following_count INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    CHECK (role   IN ('USER','ADMIN')),
    CHECK (status IN ('ACTIVE','SUSPENDED','BANNED','DELETED'))
);

-- [NÂNG CẤP] unique case-insensitive: 'Duy' và 'duy' là một
CREATE UNIQUE INDEX ux_users_username_lower ON users (lower(username)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX ux_users_email_lower    ON users (lower(email))    WHERE deleted_at IS NULL;
```

> **Vì sao `follower_count` denormalized:** trang profile luôn hiện số follower. `COUNT(*)` bảng `follows` mỗi lần load profile là lãng phí. Đổi lại phải cập nhật counter trong transaction khi follow/unfollow (xem mục 12). Đây là đánh đổi "đọc nhiều hơn ghi" — đúng cho mạng xã hội.

---

## 2. sessions

```sql
CREATE TABLE sessions (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    token_hash  VARCHAR(255) NOT NULL UNIQUE,   -- lưu sha256(token), KHÔNG lưu token gốc
    user_agent  TEXT,                            -- [NÂNG CẤP] để user xem "thiết bị đang đăng nhập"
    ip          INET,                            -- [NÂNG CẤP]
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at  TIMESTAMPTZ
);
CREATE INDEX ix_sessions_user ON sessions(user_id);
CREATE INDEX ix_sessions_expires ON sessions(expires_at) WHERE revoked_at IS NULL;
```

> **Token gốc chỉ tồn tại một lần** — sinh bằng `crypto/rand` (32 byte → base64url), trả cho client, DB chỉ lưu `sha256`. Kể cả lộ DB, không tái tạo được token. Validate = hash token client gửi rồi tra `token_hash`.

## 2b. password_reset_tokens [NÂNG CẤP — xem Phần 00b]

```sql
CREATE TABLE password_reset_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_password_reset_user ON password_reset_tokens(user_id);
```

Cùng cơ chế "lưu hash, trả token gốc một lần" như `sessions`. Sau khi `reset-password` thành công: set `used_at`, và **revoke toàn bộ session hiện có của user** (`UPDATE sessions SET revoked_at = now() WHERE user_id = ...`).

---

## 3. Địa lý: provinces → districts → wards

```sql
CREATE TABLE provinces (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE districts (
    id BIGSERIAL PRIMARY KEY,
    province_id BIGINT NOT NULL REFERENCES provinces(id),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(province_id, slug)
);
CREATE INDEX ix_districts_province ON districts(province_id);

-- [NÂNG CẤP] đề gốc nhắc "phường/xã" trong place nhưng thiếu bảng
CREATE TABLE wards (
    id BIGSERIAL PRIMARY KEY,
    district_id BIGINT NOT NULL REFERENCES districts(id),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(district_id, slug)
);
CREATE INDEX ix_wards_district ON wards(district_id);
```

---

## 4. places

```sql
CREATE TABLE places (
    id                 BIGSERIAL PRIMARY KEY,
    canonical_place_id BIGINT REFERENCES places(id),   -- NULL = tự nó là canonical
    google_place_id    VARCHAR(255),
    name               VARCHAR(255) NOT NULL,
    address            TEXT,
    province_id        BIGINT REFERENCES provinces(id),
    district_id        BIGINT REFERENCES districts(id),
    ward_id            BIGINT REFERENCES wards(id),      -- [NÂNG CẤP]
    latitude           DECIMAL(10,7),
    longitude          DECIMAL(10,7),
    post_count         INTEGER NOT NULL DEFAULT 0,       -- [NÂNG CẤP] denormalized cho trang place
    status             VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (status IN ('ACTIVE','MERGED','HIDDEN'))
);

-- google_place_id unique nhưng cho phép NULL nhiều (place tự nhập tay chưa có Google ID)
CREATE UNIQUE INDEX ux_places_google ON places(google_place_id) WHERE google_place_id IS NOT NULL;
CREATE INDEX ix_places_canonical ON places(canonical_place_id);
CREATE INDEX ix_places_province ON places(province_id);

-- [NÂNG CẤP] full-text search tên + địa chỉ, bỏ dấu tiếng Việt
ALTER TABLE places ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (
    to_tsvector('simple', unaccent(coalesce(name,'') || ' ' || coalesce(address,'')))
  ) STORED;
CREATE INDEX ix_places_search ON places USING GIN(search_vector);

-- [NÂNG CẤP] trgm để gợi ý place trùng khi admin gộp
CREATE INDEX ix_places_name_trgm ON places USING GIN(name gin_trgm_ops);
```

> **Cơ chế canonical:** khi gộp place B vào place A, đặt `B.canonical_place_id = A.id` và `B.status = 'MERGED'`. Truy vấn trang place luôn **resolve về canonical**: `COALESCE(canonical_place_id, id)`. Không phải đổi `place_id` trên hàng nghìn bài — chỉ đổi một con trỏ.
>
> **Luật bắt buộc [NÂNG CẤP]: chỉ merge một cấp, không merge dây chuyền.** `COALESCE(canonical_place_id, id)` chỉ đúng nếu `canonical_place_id` luôn trỏ thẳng tới place gốc thật sự — nếu cho phép A→B rồi B→C thì query một cấp sẽ resolve sai. Khi merge B vào A ở tầng service, **luôn resolve A về canonical thật của chính nó trước** (nếu A đã bị merge vào đâu đó thì dùng canonical của A, không dùng A trực tiếp), rồi mới gán `B.canonical_place_id = <canonical thật của A>`. Không bao giờ tạo chuỗi merge nhiều cấp.
>
> **Google Places API — giới hạn cần tuân thủ [NÂNG CẤP]:** `google_place_id` là trường được phép lưu lâu dài, nhưng Google vẫn khuyến nghị làm mới (refresh) định kỳ vì ID cũ có thể hết hiệu lực. Ngược lại, **không mặc định lưu/hiển thị tự do mọi tên, địa chỉ, ảnh** lấy về từ Google — chính sách Places có điều kiện ràng buộc về hiển thị và attribution. Trước khi tích hợp thật, đọc kỹ Google Places API Policies và hướng dẫn Place ID hiện hành, vì các điều khoản này có thể thay đổi theo thời gian.

### 4b. place_merge_history [NÂNG CẤP]

```sql
CREATE TABLE place_merge_history (
    id            BIGSERIAL PRIMARY KEY,
    merged_place_id    BIGINT NOT NULL REFERENCES places(id),  -- place bị gộp (B)
    canonical_place_id BIGINT NOT NULL REFERENCES places(id),  -- place giữ lại (A)
    merged_by     BIGINT NOT NULL REFERENCES users(id),        -- admin nào
    reason        TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## 5. posts

```sql
CREATE TABLE posts (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    place_id        BIGINT REFERENCES places(id),
    province_id     BIGINT REFERENCES provinces(id),
    content         TEXT NOT NULL,
    status          VARCHAR(30) NOT NULL DEFAULT 'VISIBLE',
    location_status VARCHAR(20) NOT NULL DEFAULT 'UNKNOWN',
    version         INTEGER NOT NULL DEFAULT 1,             -- [NÂNG CẤP] optimistic lock
    is_sponsored    BOOLEAN NOT NULL DEFAULT FALSE,          -- [NÂNG CẤP] tự khai báo nội dung tài trợ/quảng cáo

    -- Denormalized aggregate (nguồn sự thật là các bảng con; đây là cache có transaction bảo vệ)
    trusted_weight    DECIMAL(12,2) NOT NULL DEFAULT 0,
    untrusted_weight  DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_vote_count  INTEGER NOT NULL DEFAULT 0,
    untrusted_ratio   DECIMAL(6,4) NOT NULL DEFAULT 0,
    like_count        INTEGER NOT NULL DEFAULT 0,
    comment_count     INTEGER NOT NULL DEFAULT 0,
    save_count        INTEGER NOT NULL DEFAULT 0,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    hidden_at   TIMESTAMPTZ,
    deleted_at  TIMESTAMPTZ,
    CHECK (status IN ('VISIBLE','HIDDEN_BY_COMMUNITY','HIDDEN_BY_ADMIN','DELETED_BY_AUTHOR')),
    CHECK (location_status IN ('UNKNOWN','SUGGESTED','CONFIRMED'))
);

-- Index phục vụ feed: partial index chỉ bài đang hiện, sắp theo keyset
CREATE INDEX ix_posts_feed ON posts(created_at DESC, id DESC) WHERE status = 'VISIBLE';
CREATE INDEX ix_posts_user ON posts(user_id) WHERE deleted_at IS NULL;
CREATE INDEX ix_posts_place ON posts(place_id) WHERE status = 'VISIBLE';
CREATE INDEX ix_posts_province ON posts(province_id) WHERE status = 'VISIBLE';

-- [NÂNG CẤP] full-text search nội dung bài
ALTER TABLE posts ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (to_tsvector('simple', unaccent(coalesce(content,'')))) STORED;
CREATE INDEX ix_posts_search ON posts USING GIN(search_vector);
```

> **Luật `province_id` vs `place_id` [NÂNG CẤP]:** hai cột này có thể mâu thuẫn nếu không quy định rõ. Chốt luật: **khi `place_id` đã có giá trị, tỉnh hiển thị luôn lấy từ `places.province_id`** (qua JOIN), cột `posts.province_id` lúc đó chỉ là giá trị lịch sử tại thời điểm tạo bài và **không dùng để hiển thị**. `posts.province_id` chỉ đóng vai trò chính khi bài **chưa xác nhận địa điểm** (`location_status = 'UNKNOWN'` hoặc `'SUGGESTED'`), dùng cho feed theo tỉnh. Khi accept suggestion, không cần đổi `posts.province_id`.

> **Optimistic lock hoạt động thế nào:** sửa bài chạy
> `UPDATE posts SET content=$1, version=version+1, updated_at=now() WHERE id=$2 AND version=$3`.
> Nếu ai đó vừa sửa (version đã đổi), `rowsAffected = 0` → service trả `409 Conflict` "bài đã bị thay đổi, tải lại". Không bao giờ mất update thầm lặng.

---

## 6. media_assets + post_images [NÂNG CẤP — luồng upload an toàn hơn]

Bản trước để client tự gửi `public_url` bất kỳ khi tạo bài — không đủ an toàn (server không biết URL đó có thật sự do chính user vừa upload hay không, và không kiểm được MIME/size trước khi gắn vào post). Sửa thành luồng có bảng `media_assets` trung gian, post chỉ tham chiếu `media_id`:

```sql
CREATE TABLE media_assets (
    id            BIGSERIAL PRIMARY KEY,
    owner_id      BIGINT NOT NULL REFERENCES users(id),
    storage_key   VARCHAR(500) NOT NULL UNIQUE,
    mime_type     VARCHAR(50),
    size_bytes    INTEGER,
    width         INTEGER,
    height        INTEGER,
    has_gps       BOOLEAN NOT NULL DEFAULT FALSE,
    status        VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at  TIMESTAMPTZ,
    CHECK (status IN ('PENDING','USABLE','REJECTED'))
);
CREATE INDEX ix_media_assets_owner ON media_assets(owner_id, status);

CREATE TABLE post_images (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id),
    media_id BIGINT NOT NULL REFERENCES media_assets(id),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_post_images_post ON post_images(post_id);
CREATE UNIQUE INDEX ux_post_images_media ON post_images(media_id);
```

> **Luồng upload (mobile-first, đã siết lại):**
>
> 1. Client gọi `POST /api/v1/uploads/presign` kèm `content_type`, `size` → server validate MIME (chỉ JPEG/PNG/WebP) + size (giới hạn ví dụ 10MB) → tạo dòng `media_assets` (`status='PENDING'`, `owner_id` = user hiện tại) → trả `upload_url` (presigned PUT) + `media_id`.
> 2. Client PUT ảnh thẳng lên storage bằng `upload_url`. Server Go không đụng byte ảnh trong bước này.
> 3. Client gọi `POST /api/v1/uploads/{media_id}/confirm` → server **tự** tải lại object vừa upload (không tin client), kiểm MIME/kích thước thật khớp khai báo, **bỏ EXIF/GPS** (v1 mặc định luôn bỏ, trừ khi hỏi rõ và người dùng đồng ý giữ vị trí), quét virus/nội dung độc hại cơ bản khi hệ thống lớn hơn, rồi set `status='USABLE'`, `confirmed_at=now()`.
> 4. Khi tạo/sửa post, client chỉ gửi `media_id` (không gửi URL). Server kiểm `media_assets.owner_id = user hiện tại` và `status='USABLE'` trước khi tạo `post_images` — chặn việc gắn ảnh của người khác hoặc ảnh chưa xác minh vào bài.
> 5. `media_assets` ở trạng thái `PENDING` quá X giờ (ví dụ 24h) không được confirm → job dọn rác xóa khỏi storage.
> 6. Báo cáo ảnh vi phạm bản quyền/nội dung xấu dùng chung cơ chế `reports`.

---

## 7. hashtags + post_hashtags [NÂNG CẤP]

Đề gốc nhắc hashtag nhưng không có bảng. Chuẩn hóa many-to-many:

```sql
CREATE TABLE hashtags (
    id BIGSERIAL PRIMARY KEY,
    tag VARCHAR(100) NOT NULL UNIQUE,       -- lưu lowercase, không dấu #
    post_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE post_hashtags (
    post_id    BIGINT NOT NULL REFERENCES posts(id),
    hashtag_id BIGINT NOT NULL REFERENCES hashtags(id),
    PRIMARY KEY(post_id, hashtag_id)
);
CREATE INDEX ix_post_hashtags_hashtag ON post_hashtags(hashtag_id);
```

---

## 8. comments (tối đa 2 cấp)

```sql
CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    post_id   BIGINT NOT NULL REFERENCES posts(id),
    user_id   BIGINT NOT NULL REFERENCES users(id),
    parent_id BIGINT REFERENCES comments(id),
    content   TEXT NOT NULL,
    reply_count INTEGER NOT NULL DEFAULT 0,     -- [NÂNG CẤP] đếm reply cho UI
    status    VARCHAR(20) NOT NULL DEFAULT 'VISIBLE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CHECK (status IN ('VISIBLE','DELETED_BY_USER','HIDDEN_BY_ADMIN'))
);
CREATE INDEX ix_comments_post ON comments(post_id) WHERE deleted_at IS NULL;
CREATE INDEX ix_comments_parent ON comments(parent_id);
```

> **Ép 2 cấp ở tầng service**: khi tạo reply, kiểm `parent.parent_id IS NULL`. Nếu người ta reply vào một reply, ta gắn nó vào comment gốc của reply đó (flatten). Không tin client tự giữ luật.

---

## 9. Tương tác: post_likes, saved_posts

```sql
CREATE TABLE post_likes (
    post_id BIGINT NOT NULL REFERENCES posts(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(post_id, user_id)
);
CREATE INDEX ix_post_likes_user ON post_likes(user_id);

CREATE TABLE saved_posts (
    post_id BIGINT NOT NULL REFERENCES posts(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(post_id, user_id)
);
CREATE INDEX ix_saved_posts_user ON saved_posts(user_id, created_at DESC);
```

> **Idempotent like:** `INSERT INTO post_likes ... ON CONFLICT DO NOTHING` trả về số dòng chèn. Nếu = 1 (mới thật) thì `like_count++` trong cùng transaction. Bấm 2 lần không cộng đôi. Unlike làm ngược lại với `DELETE ... RETURNING`.

---

## 10. review_votes

```sql
CREATE TABLE review_votes (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    vote_type VARCHAR(20) NOT NULL,
    weight_at_vote DECIMAL(4,2) NOT NULL,       -- snapshot, KHÔNG tính lại về sau
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(post_id, user_id),
    CHECK (vote_type IN ('TRUSTED','UNTRUSTED'))
);
CREATE INDEX ix_review_votes_post ON review_votes(post_id);
```

Upsert khi đổi vote:

```sql
INSERT INTO review_votes (post_id, user_id, vote_type, weight_at_vote)
VALUES ($1, $2, $3, $4)
ON CONFLICT (post_id, user_id)
DO UPDATE SET vote_type = EXCLUDED.vote_type,
              weight_at_vote = EXCLUDED.weight_at_vote,  -- weight tính lại tại thời điểm đổi
              updated_at = now();
```

Aggregate lại từ nguồn sự thật (chạy trong transaction, sau khi đã `SELECT ... FOR UPDATE` post):

```sql
SELECT
    COUNT(*) AS total_voters,
    COALESCE(SUM(CASE WHEN vote_type='TRUSTED'   THEN weight_at_vote END), 0) AS trusted_weight,
    COALESCE(SUM(CASE WHEN vote_type='UNTRUSTED' THEN weight_at_vote END), 0) AS untrusted_weight
FROM review_votes WHERE post_id = $1;
```

Rồi ghi vào `posts` + tính `untrusted_ratio`, và ẩn nếu `total_voters >= 200 AND ratio > 0.70`.

---

## 11. location_suggestions

```sql
CREATE TABLE location_suggestions (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id),
    suggested_by BIGINT NOT NULL REFERENCES users(id),
    place_id BIGINT REFERENCES places(id),
    google_place_id VARCHAR(255),
    place_name VARCHAR(255) NOT NULL,
    address TEXT,
    map_url TEXT,   -- [NÂNG CẤP] xem ghi chú bên dưới: không nhận URL tự do từ client
    latitude DECIMAL(10,7),
    longitude DECIMAL(10,7),
    note TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    CHECK (status IN ('PENDING','ACCEPTED','REJECTED','CANCELLED'))
);
-- chống một người đề xuất trùng cùng google_place_id cho cùng bài
CREATE UNIQUE INDEX ux_suggestion_dedup ON location_suggestions(post_id, suggested_by, google_place_id)
    WHERE google_place_id IS NOT NULL;
CREATE INDEX ix_suggestions_post ON location_suggestions(post_id) WHERE status = 'PENDING';
```

Luồng accept (transaction đầy đủ ở Phần 03 và Phần 04): khóa post `FOR UPDATE` → tạo/tìm place → gắn `place_id`, `location_status='CONFIRMED'` → suggestion chọn `ACCEPTED`, các cái khác `REJECTED` → tăng `places.post_count` → tạo notification. Tất cả trong một transaction.

> **`map_url` [NÂNG CẤP]:** không nhận URL tùy ý do client gửi lên — có thể trỏ tới bất cứ đâu và không kiểm chứng được. Server **tự sinh** `map_url` ở tầng đọc từ `google_place_id` (nếu có) hoặc từ toạ độ `latitude/longitude` đã validate, thay vì lưu nguyên input thô của client.

---

## 11b. idempotency_keys [NÂNG CẤP — bảng thật, không chỉ mô tả API]

Header `Idempotency-Key` chỉ có ý nghĩa nếu có nơi lưu và so khớp. Thêm bảng:

```sql
CREATE TABLE idempotency_keys (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    idempotency_key VARCHAR(100) NOT NULL,
    endpoint        VARCHAR(200) NOT NULL,
    request_hash    VARCHAR(64) NOT NULL,
    response_status INTEGER,
    response_body   JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,
    UNIQUE(user_id, idempotency_key, endpoint)
);
CREATE INDEX ix_idempotency_expires ON idempotency_keys(expires_at);
```

> **Luồng dùng:** middleware kiểm `(user_id, idempotency_key, endpoint)` đã tồn tại chưa. Nếu có và `request_hash` khớp → trả lại response đã lưu, **không chạy lại handler**. Nếu có nhưng `request_hash` khác → `409 CONFLICT`. Nếu chưa có → chạy handler bình thường, ghi kết quả vào bảng trong cùng transaction với thao tác nghiệp vụ. Job dọn rác xóa dòng đã hết `expires_at`.

---

## 12. follows

```sql
CREATE TABLE follows (
    follower_id  BIGINT NOT NULL REFERENCES users(id),
    following_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(follower_id, following_id),
    CHECK (follower_id <> following_id)   -- DB cũng chặn tự follow
);
CREATE INDEX ix_follows_following ON follows(following_id);
```

Follow = transaction: `INSERT ... ON CONFLICT DO NOTHING`; nếu chèn mới thì `follower.following_count++` và `target.follower_count++` + tạo notification. Unfollow ngược lại.

---

## 12b. blocks + mutes [NÂNG CẤP — cần làm trước feed ranking nâng cao]

```sql
CREATE TABLE blocks (
    blocker_id BIGINT NOT NULL REFERENCES users(id),
    blocked_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);
CREATE INDEX ix_blocks_blocked ON blocks(blocked_id);

CREATE TABLE mutes (
    muter_id BIGINT NOT NULL REFERENCES users(id),
    muted_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(muter_id, muted_id),
    CHECK (muter_id <> muted_id)
);
```

> **Block là hai chiều về mặt hiển thị:** nếu A block B, tầng service khi load feed/post/comment cho A phải lọc bỏ nội dung của B **và ngược lại** — kiểm cả hai chiều `(blocker_id, blocked_id)` trong điều kiện lọc. Block còn ngăn B follow/comment/vote lên nội dung của A.
> **Mute là một chiều:** chỉ lọc nội dung người bị mute ra khỏi feed của người mute; không ảnh hưởng follow/comment/vote, và người bị mute không biết mình bị mute.

---

## 13. reports

```sql
CREATE TABLE reports (
    id BIGSERIAL PRIMARY KEY,
    reporter_id BIGINT NOT NULL REFERENCES users(id),
    target_type VARCHAR(20) NOT NULL,     -- POST | COMMENT | USER | PLACE
    target_id BIGINT NOT NULL,
    reason VARCHAR(100) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    handled_by BIGINT REFERENCES users(id),   -- [NÂNG CẤP] admin nào xử lý
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    CHECK (target_type IN ('POST','COMMENT','USER','PLACE')),
    CHECK (status IN ('PENDING','REVIEWING','RESOLVED','REJECTED'))
);
CREATE INDEX ix_reports_status ON reports(status, created_at);
-- [NÂNG CẤP] chống spam report: một người báo cáo một target một lần trong 24h (xử lý ở service)
CREATE INDEX ix_reports_dedup ON reports(reporter_id, target_type, target_id);
```

---

## 14. notifications + outbox [NÂNG CẤP]

```sql
CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),   -- người NHẬN
    actor_id BIGINT REFERENCES users(id),           -- người GÂY ra
    type VARCHAR(50) NOT NULL,                       -- COMMENT | REPLY | LIKE | FOLLOW | SUGGESTION_* | POST_HIDDEN ...
    reference_type VARCHAR(20),                      -- [NÂNG CẤP] POST|COMMENT|SUGGESTION để client điều hướng
    reference_id BIGINT,
    content TEXT,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_notifications_user ON notifications(user_id, created_at DESC);
CREATE INDEX ix_notifications_unread ON notifications(user_id) WHERE is_read = FALSE;
```

> **Outbox nhẹ:** notification được `INSERT` **trong cùng transaction** với hành động sinh ra nó (vd: cùng transaction accept suggestion). Nhờ đó nếu transaction rollback thì notification cũng biến mất — không có "đã báo chấp nhận nhưng thực ra chưa gắn place". Khi lên app thật, thêm cột `pushed_at` và một worker đọc các dòng `pushed_at IS NULL` để đẩy FCM/APNs. MVP thì client chỉ cần `GET /notifications`.

---

## 15. admin_actions (audit log) [NÂNG CẤP]

```sql
CREATE TABLE admin_actions (
    id BIGSERIAL PRIMARY KEY,
    admin_id BIGINT NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL,           -- HIDE_POST | RESTORE_POST | BAN_USER | MERGE_PLACE | DELETE_COMMENT ...
    target_type VARCHAR(20) NOT NULL,
    target_id BIGINT NOT NULL,
    detail JSONB,                           -- ghi thêm ngữ cảnh (lý do, trạng thái trước)
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_admin_actions_admin ON admin_actions(admin_id, created_at DESC);
```

Mọi hành động admin ghi một dòng ở đây (trong cùng transaction với hành động). Trả lời được câu hỏi "ai ẩn bài này, lúc nào, vì sao".

---

## 16. Chọn driver: `pgx` thay `lib/pq`

Đề gốc gợi `lib/pq`. Khuyến nghị đổi sang **`github.com/jackc/pgx/v5/stdlib`** dùng qua `database/sql` (vẫn giữ ràng buộc "dùng `database/sql`, không ORM"):

- `lib/pq` đã ở chế độ maintenance, `pgx` là de-facto standard hiện tại.
- Hỗ trợ `TIMESTAMPTZ`, `JSONB`, `INET` mượt hơn.
- Vẫn viết SQL tay 100%, vẫn `db.QueryContext(...)` như thường — chỉ khác dòng import driver.

```go
import _ "github.com/jackc/pgx/v5/stdlib"
db, _ := sql.Open("pgx", cfg.DatabaseURL)
```

Nếu bạn muốn giữ đúng đề gốc thì `lib/pq` vẫn chạy tốt cho MVP — không sao.

---

## 17. Pool config (đừng để mặc định)

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(1 * time.Minute)
```

`MaxOpenConns` mặc định là **vô hạn** — dễ làm sập Postgres khi tải cao. Đặt trần rõ ràng.

---

## 18. Thứ tự migration (golang-migrate)

```text
000001_extensions
000002_create_users                      (+ email_verified_at)
000003_create_sessions
000004_create_password_reset_tokens      [NÂNG CẤP]
000005_create_geo                        (provinces, districts, wards)
000006_create_places                     (+ merge_history, search_vector)
000007_create_media_assets               [NÂNG CẤP]
000008_create_posts                      (+ images qua media_id, hashtags, search_vector, is_sponsored)
000009_create_comments
000010_create_interactions               (likes, saves, votes)
000011_create_suggestions
000012_create_follows
000013_create_blocks_mutes               [NÂNG CẤP]
000014_create_reports
000015_create_notifications
000016_create_admin_actions
000017_create_idempotency_keys           [NÂNG CẤP]
000018_seed_haiphong_geo                 (seed data, .up chèn / .down xóa)
```

**Quy tắc migration:** đã apply lên môi trường chung thì **không sửa file cũ** — luôn viết migration mới để thay đổi. Mỗi `.up.sql` phải có `.down.sql` rollback được.

---

_Tiếp theo: phần Đặc tả API bên dưới._

---

# 03 — Đặc tả API v1

Base URL: `/api/v1`. Tất cả body JSON. Auth qua `Authorization: Bearer <session_token>`.

## Quy ước chung

**Response envelope** (mọi endpoint):

```jsonc
// Thành công (single)
{ "data": { ... } }

// Thành công (list, có phân trang)
{ "data": [ ... ], "meta": { "next_cursor": "eyJ...", "count": 20 } }

// Lỗi
{ "error": { "code": "NOT_FOUND", "message": "không tìm thấy bài viết" } }
```

**Mã lỗi → HTTP status:**

| code                | status | khi nào                                        |
| ------------------- | ------ | ---------------------------------------------- |
| `BAD_REQUEST`       | 400    | input sai cú pháp / validate fail              |
| `UNAUTHORIZED`      | 401    | thiếu/sai/hết hạn token                        |
| `FORBIDDEN`         | 403    | đăng nhập rồi nhưng không có quyền             |
| `NOT_FOUND`         | 404    | không tồn tại                                  |
| `CONFLICT`          | 409    | trùng, version lệch, trạng thái không cho phép |
| `TOO_MANY_REQUESTS` | 429    | vượt rate limit                                |
| `INTERNAL`          | 500    | lỗi hệ thống (không leak chi tiết)             |

**Pagination**: list endpoint nhận `?limit=20&cursor=<opaque>`. Lần đầu bỏ `cursor`. Server trả `meta.next_cursor`; hết trang thì rỗng. `limit` mặc định 20, tối đa 50.

**Idempotency**: POST tạo tài nguyên nhận header `Idempotency-Key: <uuid>` (client sinh). Retry cùng key không tạo trùng.

**Phân quyền tóm tắt**: 🟢 guest xem được · 🔵 cần đăng nhập · 🔴 admin.

---

## Authentication

```text
POST   /api/v1/auth/register           🟢
POST   /api/v1/auth/login              🟢
POST   /api/v1/auth/logout             🔵
GET    /api/v1/auth/me                 🔵
POST   /api/v1/auth/verify             🔵   [NÂNG CẤP] xác thực email/SĐT bằng OTP/code
POST   /api/v1/auth/forgot-password    🟢   [NÂNG CẤP] luôn trả 200 chung chung
POST   /api/v1/auth/reset-password     🟢   [NÂNG CẤP] token 1 lần dùng, revoke toàn bộ session sau khi đổi
GET    /api/v1/auth/sessions           🔵   xem thiết bị đang đăng nhập
DELETE /api/v1/auth/sessions/{id}      🔵   đăng xuất một thiết bị
DELETE /api/v1/auth/sessions           🔵   [NÂNG CẤP] đăng xuất mọi thiết bị trừ hiện tại
DELETE /api/v1/users/me                🔵   [NÂNG CẤP] xóa mềm tài khoản (xem Phần 00b, mục 4)
```

**Register** `POST /auth/register`

```jsonc
// request
{ "username": "duy", "email": "duy@x.com", "password": "minimum8ch", "display_name": "Duy" }
// 201
{ "data": { "user": { "id": 1, "username": "duy", "display_name": "Duy" },
            "token": "raw-session-token", "expires_at": "2026-08-09T..." } }
```

Validate: username 3–50 `[a-z0-9_]`, email hợp lệ hoặc phone, password ≥ 8. Trùng username/email → `409 CONFLICT`. Password hash bcrypt cost 10–12. **Không bao giờ** trả `password_hash`.

**Login** `POST /auth/login`

```jsonc
{ "identifier": "duy@x.com", "password": "..." } // identifier = email HOẶC username
```

Sai → `401 UNAUTHORIZED` (thông báo mơ hồ "sai thông tin đăng nhập", không nói sai user hay sai pass). Tài khoản `SUSPENDED/BANNED` → `403`.

---

## Users

```text
GET    /api/v1/users/{id}              🟢  profile công khai + counters
PATCH  /api/v1/users/{id}              🔵  chỉ sửa được chính mình (kiểm ở service)
GET    /api/v1/users/{id}/posts        🟢  bài của user (cursor)
POST   /api/v1/users/{id}/follow       🔵
DELETE /api/v1/users/{id}/follow       🔵
GET    /api/v1/users/{id}/followers    🟢  (cursor)
GET    /api/v1/users/{id}/following    🟢  (cursor)
POST   /api/v1/users/{id}/block        🔵  [NÂNG CẤP] hai chiều — xem Phần 02 mục 12b
DELETE /api/v1/users/{id}/block        🔵  [NÂNG CẤP]
POST   /api/v1/users/{id}/mute         🔵  [NÂNG CẤP] một chiều
DELETE /api/v1/users/{id}/mute         🔵  [NÂNG CẤP]
GET    /api/v1/me/blocked-users        🔵  [NÂNG CẤP]
```

`PATCH /users/{id}`: nếu `{id}` ≠ user trong session → `403`. Chỉ cho sửa `display_name, bio, avatar_url`. Không cho đổi `role, status` (đó là việc của admin).

---

## Posts

```text
POST   /api/v1/posts             🔵  tạo bài (kèm images, hashtags, place tùy chọn)
GET    /api/v1/posts/{id}        🟢  chi tiết + trạng thái like/save/vote của người xem
PATCH  /api/v1/posts/{id}        🔵  sửa content/images/hashtags (optimistic lock)
DELETE /api/v1/posts/{id}        🔵  soft delete, chỉ tác giả
```

**Create** `POST /posts`

```jsonc
{
  "content": "Nay ăn ốc gần Lê Lợi ngon quá nhưng quên địa chỉ...",
  "province_id": 1,
  "place_id": null, // null = chưa biết địa điểm → location_status = UNKNOWN
  "images": [{ "media_id": 501 }], // [NÂNG CẤP] chỉ nhận media_id đã USABLE, không nhận URL tự do
  "hashtags": ["oc", "haiphong"],
  "is_sponsored": false,
}
```

Xử lý: transaction tạo post + images (kiểm `media_assets.owner_id = user hiện tại` và `status='USABLE'` cho từng `media_id`) + upsert hashtags + link `post_hashtags`. Nếu có `place_id` hợp lệ → `location_status = CONFIRMED`, tăng `places.post_count`.

**Get** `GET /posts/{id}` trả kèm ngữ cảnh người xem (mobile cần trong 1 call):

```jsonc
{ "data": {
    "id": 10, "content": "...", "status": "VISIBLE", "location_status": "UNKNOWN",
    "author": { "id": 1, "username": "duy", "display_name": "Duy", "avatar_url": "..." },
    "place": null,
    "images": [...], "hashtags": ["oc","haiphong"],
    "like_count": 12, "comment_count": 3, "save_count": 2,
    "vote_summary": { "trusted_weight": 8.0, "untrusted_weight": 1.0, "total_voters": 9, "untrusted_ratio": 0.11 },
    "viewer": { "liked": true, "saved": false, "vote_type": null },  // null nếu guest
    "created_at": "..."
} }
```

**Patch** `PATCH /posts/{id}` — gửi kèm `version` hiện có:

```jsonc
{ "content": "...", "hashtags": [...], "version": 3 }
```

Version lệch → `409 CONFLICT "bài đã bị thay đổi, tải lại"`. Không cho sửa `place_id` trực tiếp nếu bài đã `CONFIRMED` và có nhiều tương tác — dùng luồng đề xuất địa điểm để lưu lịch sử.

---

## Comments

```text
POST   /api/v1/posts/{id}/comments       🔵
GET    /api/v1/posts/{id}/comments       🟢  comment gốc (cursor) + preview reply
POST   /api/v1/comments/{id}/replies     🔵  reply (ép về 2 cấp ở service)
GET    /api/v1/comments/{id}/replies     🟢  (cursor)
DELETE /api/v1/comments/{id}             🔵  tác giả soft delete
```

Tạo comment → transaction: insert comment + `posts.comment_count++` + notification cho tác giả bài. Reply → thêm `parent.reply_count++`.

---

## Likes & Saves

```text
POST   /api/v1/posts/{id}/like     🔵   idempotent
DELETE /api/v1/posts/{id}/like     🔵
POST   /api/v1/posts/{id}/save     🔵   idempotent
DELETE /api/v1/posts/{id}/save     🔵
GET    /api/v1/me/saved-posts      🔵   (cursor) — chỉ chủ nhân thấy
```

Trả về state mới: `{ "data": { "liked": true, "like_count": 13 } }`.

---

## Votes độ tin cậy

```text
PUT    /api/v1/posts/{id}/vote           🔵   set/đổi vote (idempotent theo user)
DELETE /api/v1/posts/{id}/vote           🔵   gỡ vote
GET    /api/v1/posts/{id}/vote-summary   🟢
```

```jsonc
// PUT request
{ "vote_type": "UNTRUSTED" }   // hoặc "TRUSTED"
// response = vote_summary mới nhất
{ "data": { "trusted_weight": 40.0, "untrusted_weight": 100.0,
            "total_voters": 250, "untrusted_ratio": 0.7143, "post_hidden": true } }
```

Rule: không tự vote bài mình (`403`). Bài không `VISIBLE` → `409`. Trọng số theo tuổi tài khoản, snapshot vào `weight_at_vote`. Đủ ngưỡng (≥200 voters & ratio >70%) → tự ẩn bài.

---

## Location suggestions

```text
POST   /api/v1/posts/{id}/location-suggestions    🔵   ai đọc cũng đề xuất được
GET    /api/v1/posts/{id}/location-suggestions    🟢   gom theo place, đếm số người đề xuất
POST   /api/v1/location-suggestions/{id}/accept   🔵   CHỈ chủ bài
POST   /api/v1/location-suggestions/{id}/reject   🔵   CHỈ chủ bài
DELETE /api/v1/location-suggestions/{id}          🔵   người đề xuất tự hủy (→ CANCELLED)
```

`GET` trả gom nhóm để chủ bài dễ chọn:

```jsonc
{ "data": [
  { "google_place_id": "Gx1", "place_name": "Ốc Thủy Dương", "address": "...",
    "suggestion_count": 8, "suggestion_ids": [12,15,...] },
  { "google_place_id": "Gx2", "place_name": "Ốc Hạnh", "suggestion_count": 2, "suggestion_ids": [20,21] }
] }
```

**Accept** (transaction — luồng lõi của sản phẩm):

1. Kiểm caller là chủ bài (lấy từ session) → nếu không, `403`.
2. `SELECT ... FROM posts WHERE id=$1 FOR UPDATE` (khóa bài).
3. Kiểm suggestion còn `PENDING` → nếu không, `409`.
4. Tạo hoặc lấy `place` (ưu tiên khớp `google_place_id`; không có thì tạo mới).
5. `UPDATE posts SET place_id=$1, location_status='CONFIRMED', updated_at=now()`.
6. Suggestion được chọn → `ACCEPTED`; các suggestion `PENDING` khác của bài → `REJECTED`.
7. `places.post_count++`.
8. Notification cho người được accept (và người bị reject nếu muốn).
9. Commit.

---

## Places

```text
GET    /api/v1/places/{id}              🟢   info + counters (resolve canonical)
GET    /api/v1/places/{id}/posts        🟢   bài cùng place (cursor); ?sort=latest|trusted
GET    /api/v1/places/search?q=         🟢   tìm kiếm không dấu cơ bản
POST   /api/v1/places/{id}/edit-suggestions  🔵  [NÂNG CẤP] đề xuất sửa tên/địa chỉ/giờ mở cửa, admin duyệt (xem Phần 00a mục 6) — làm sau MVP-3
```

`{id}` là place đã bị merge → tự chuyển hướng logic sang canonical (trả cả bài của place con). `search` dùng `tsvector @@ plainto_tsquery(unaccent($1))`.

---

## Feed

```text
GET    /api/v1/feed/latest                 🟢   bài VISIBLE mới nhất (cursor)
GET    /api/v1/feed/following              🔵   bài của người mình theo dõi (cursor)
GET    /api/v1/feed/province/{id}          🟢   theo tỉnh (mặc định Hải Phòng)
```

Mỗi item feed trả đủ như `GET /posts/{id}` (author + counters + viewer state) để mobile không phải gọi thêm.

---

## Reports

```text
POST   /api/v1/reports                     🔵   báo cáo (dedup 24h/target ở service)
GET    /api/v1/admin/reports               🔴   ?status=PENDING (cursor)
PATCH  /api/v1/admin/reports/{id}          🔴   đổi status, ghi handled_by + admin_action
```

## Notifications

```text
GET    /api/v1/notifications               🔵   (cursor) + ?unread=true
GET    /api/v1/notifications/unread-count   🔵   [NÂNG CẤP] badge số
PATCH  /api/v1/notifications/{id}/read      🔵
PATCH  /api/v1/notifications/read-all       🔵
```

## Admin

```text
PATCH  /api/v1/admin/users/{id}/status      🔴   SUSPEND/BAN/ACTIVE (ghi admin_action)
PATCH  /api/v1/admin/posts/{id}/hide        🔴   HIDDEN_BY_ADMIN
PATCH  /api/v1/admin/posts/{id}/restore     🔴   về VISIBLE (khôi phục cả bài bị cộng đồng ẩn nhầm)
DELETE /api/v1/admin/comments/{id}          🔴   HIDDEN_BY_ADMIN
POST   /api/v1/admin/places/merge           🔴   gộp 2 place → ghi merge_history
GET    /api/v1/admin/posts?status=HIDDEN_BY_COMMUNITY  🔴   xem bài bị cộng đồng ẩn
```

## Uploads [NÂNG CẤP — luồng an toàn hơn, xem Phần 02 mục 6]

```text
POST   /api/v1/uploads/presign             🔵   xin upload key tạm thời + tạo media_assets(PENDING)
POST   /api/v1/uploads/{media_id}/confirm  🔵   server xác minh & chuyển media sang USABLE
```

```jsonc
// POST /uploads/presign — request
{ "content_type": "image/jpeg", "size": 823400 }
// response
{ "data": { "media_id": 501, "upload_url": "https://r2.../signed?...", "expires_in": 300 } }

// Client PUT ảnh thẳng lên upload_url, sau đó:
// POST /uploads/501/confirm — request rỗng
// response
{ "data": { "media_id": 501, "status": "USABLE", "width": 1080, "height": 1350 } }
```

Khi tạo bài, client chỉ gửi `media_id` đã `USABLE` (không gửi URL) — xem lại ví dụ `POST /posts`, trường `images` giờ nhận `media_id` thay vì `url`.

## Health

```text
GET    /healthz     🟢   liveness (luôn 200 nếu process sống)
GET    /readyz      🟢   readiness (ping DB; 503 nếu DB chết)
```

---

## Ma trận rate limit (gợi ý)

| Endpoint                    | Giới hạn                                 |
| --------------------------- | ---------------------------------------- |
| `POST /auth/register`       | 5 / giờ / IP                             |
| `POST /auth/login`          | 10 / 15 phút / IP                        |
| `POST /posts`               | 20 / giờ / user                          |
| `POST /posts/{id}/comments` | 60 / giờ / user                          |
| `PUT /posts/{id}/vote`      | 100 / giờ / user                         |
| `POST /reports`             | 20 / ngày / user                         |
| còn lại                     | 300 / phút / user (chống lạm dụng chung) |

---

_Tiếp theo: phần Lộ trình & Quy trình bên dưới._

---

# 04 — Lộ trình triển khai & Quy trình

Phần này trả lời: "làm theo thứ tự nào, mỗi bước xong khi nào tính là xong, và sau khi code xong thì làm gì tiếp".

Chiến lược tổng: **backend API trước → mobile app → web**. Trong backend, đi từ móng (auth) lên dần theo phụ thuộc nghiệp vụ. Mỗi giai đoạn kết thúc bằng một API **chạy được, test được, gọi được từ Postman**, chưa cần client.

---

## Phần A — Lộ trình backend (7 giai đoạn)

Mỗi giai đoạn có **Definition of Done (DoD)** — không đạt đủ thì chưa qua giai đoạn sau.

### Giai đoạn 0 — Móng dự án (0.5–1 ngày)

Làm:

- `go mod init`, dựng cấu trúc thư mục theo Phần 01.
- `config` load env + fail-fast nếu thiếu.
- `database.Connect` + pool config + `WithTx` helper.
- Middleware: `Recovery`, `RequestID`, `Logging` (slog).
- `httpx`: envelope, `WriteJSON`, `Error` mapping, `DecodeJSON` (giới hạn body 1MB).
- `apperr` đầy đủ.
- `/healthz`, `/readyz`.
- `Makefile`, `docker-compose.yml` (Postgres), `.env.example`.
- Setup `golang-migrate`, viết migration `000001` + `000002`.

**DoD:** `make run` lên server, `curl /healthz` trả 200, `curl /readyz` trả 200 khi DB sống / 503 khi tắt DB. `make migrate-up` chạy sạch.

### Giai đoạn 1 — Auth & session

Làm: register, login, logout, `me`, middleware `Authenticate`, `RequireRole`. Bcrypt. Session token (sha256 lưu DB). Rate limit cho register/login.

**DoD:**

- Đăng ký → nhận token; đăng ký trùng username/email → 409.
- Login sai → 401; login tài khoản BANNED → 403.
- `GET /me` không token → 401; có token → 200 trả user (không có `password_hash`).
- Logout → token cũ dùng lại bị 401.
- Có **test**: `password_test.go` (hash/verify), `service_test.go` cho login (mock repo).

### Giai đoạn 2 — Posts & feed cơ bản

Làm: CRUD post (soft delete, optimistic lock), post_images, hashtags, feed latest + by province (cursor pagination), user posts, profile.

**DoD:**

- Tạo bài không địa điểm → `location_status = UNKNOWN`, đăng được.
- Sửa bài với `version` cũ → 409.
- Xóa bài → biến mất khỏi feed nhưng còn trong DB (`status=DELETED_BY_AUTHOR`).
- Feed phân trang bằng cursor, không trùng/mất bài khi chèn bài mới giữa 2 lần gọi.
- Không phải tác giả mà sửa/xóa bài người khác → 403.
- Test cursor encode/decode; test service tạo bài (mock).

### Giai đoạn 3 — Tương tác xã hội

Làm: comment + reply (ép 2 cấp), like/unlike (idempotent), save/unsave, follow/unfollow, feed following. Counter cập nhật trong transaction. Notification cho comment/like/follow.

**DoD:**

- Like 2 lần liên tiếp → `like_count` chỉ +1.
- Reply vào reply → tự flatten về comment gốc.
- Follow chính mình → 400/DB chặn.
- Counter (`like_count`, `comment_count`, `follower_count`) luôn khớp `COUNT(*)` thực (viết một integration test kiểm tra bất biến này).
- Feed following chỉ ra bài người đang theo dõi.

### Giai đoạn 4 — Địa điểm & đề xuất (luồng lõi sản phẩm)

Làm: places CRUD nội bộ, tạo/gom suggestion, accept/reject (transaction + FOR UPDATE), trang place + gom bài cùng place, place search full-text, canonical resolve.

**DoD:**

- Bài UNKNOWN → người khác đề xuất → `location_status=SUGGESTED`.
- Chủ bài accept → bài `CONFIRMED`, place gắn, các suggestion khác `REJECTED`, người đề xuất nhận notification. Tất cả **atomic** (test rollback: giả lập lỗi giữa chừng → không có thay đổi nào lọt).
- Người **không phải** chủ bài gọi accept → 403.
- Accept một suggestion đã resolved → 409.
- Trang place gom đúng bài, kể cả bài của place đã bị merge (canonical).

### Giai đoạn 5 — Vote độ tin cậy & auto-hide

Làm: cast/change/remove vote, tính trọng số theo tuổi tài khoản (snapshot), aggregate trong transaction có `FOR UPDATE`, auto-hide theo ngưỡng, ẩn bài khỏi feed/search/place.

**DoD:**

- `CalculateVoteWeight` và `ShouldHidePost` có test table-driven phủ biên (7 ngày, 30 ngày, 200 voters, 70%).
- Vote khi tài khoản 5 ngày → `weight_at_vote=0.25`; sau 30 ngày phiếu cũ vẫn 0.25.
- Đổi vote cập nhật aggregate đúng; hai người vote đồng thời không làm hỏng số liệu (integration test chạy 2 goroutine).
- Bài đạt ≥200 voters & ratio >70% → `HIDDEN_BY_COMMUNITY`, biến mất khỏi feed/search/place, tác giả và admin vẫn xem được.

### Giai đoạn 6 — Notification, report, admin, search, phân trang hoàn chỉnh

Làm: list/read notification + badge, report + dedup, admin (hide/restore/ban/merge place/handle report) + `admin_actions`, search bài/user/place, hoàn thiện pagination toàn hệ thống.

**DoD:**

- Mọi hành động admin ghi `admin_actions`.
- Merge place: bài của cả hai hiện chung ở place canonical.
- Report cùng target trong 24h bị chặn trùng.
- Search "pho" ra "phở" (unaccent hoạt động).
- Admin khôi phục được bài bị cộng đồng ẩn nhầm.

---

## Phần B — Testing (bắt buộc, không phải tùy chọn)

Đây là ranh giới rõ nhất giữa junior và mid. Ba tầng test:

**1. Unit test — hàm thuần & service (nhanh, chạy mỗi lần save)**

- Hàm thuần: `CalculateVoteWeight`, `ShouldHidePost`, cursor encode/decode, validate input. Table-driven.
- Service: mock repository (implement interface bằng struct in-memory), `platform.Clock` giả để test logic thời gian. Test được cả nhánh lỗi (bài không tồn tại → NotFound, tự vote → Forbidden).

```go
// mock repo ví dụ
type mockPostRepo struct {
    posts map[int64]*post.Post
    forUpdateErr error
}
func (m *mockPostRepo) GetForUpdate(_ context.Context, _ database.Querier, id int64) (*post.Post, error) {
    if m.forUpdateErr != nil { return nil, m.forUpdateErr }
    p, ok := m.posts[id]
    if !ok { return nil, apperr.ErrNotFound }
    return p, nil
}
```

**2. Integration test — chạm Postgres thật (build tag `//go:build integration`)**

- Dùng `docker-compose` DB test hoặc `testcontainers-go`.
- Kiểm bất biến: sau N like/unlike ngẫu nhiên, `like_count == COUNT(*)`.
- Kiểm concurrency: 50 goroutine vote đồng thời một bài → aggregate cuối đúng, không panic, không deadlock.
- Kiểm transaction rollback: inject lỗi giữa luồng accept suggestion → DB không đổi.

**3. API/e2e test — gọi HTTP thật**

- `httptest.NewServer` + gọi bằng `http.Client`, hoặc collection Postman/newman trong CI.
- Phủ các luồng chính ở "Kết quả cuối cùng" của đề gốc: đăng bài không địa chỉ → đề xuất → accept → gom place → vote → auto-hide.

Mục tiêu coverage thực dụng: **service ≥ 70%**, hàm thuần **100%**. Không chạy theo con số coverage handler (handler chỉ nên nhàm chán).

---

## Phần C — Công cụ & quy trình dev

### Makefile (khung)

```makefile
run:          ; go run ./cmd/api
build:        ; go build -o bin/api ./cmd/api
test:         ; go test ./...
test-int:     ; go test -tags=integration ./test/...
lint:         ; golangci-lint run
migrate-up:   ; migrate -path migrations -database "$$DATABASE_URL" up
migrate-down: ; migrate -path migrations -database "$$DATABASE_URL" down 1
migrate-new:  ; migrate create -ext sql -dir migrations -seq $(name)
seed:         ; psql "$$DATABASE_URL" -f scripts/seed.sql
```

### docker-compose (dev)

```yaml
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_USER: food
      POSTGRES_PASSWORD: food
      POSTGRES_DB: foodsocial
    ports: ["5432:5432"]
    volumes: ["dbdata:/var/lib/postgresql/data"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U food"]
      interval: 5s
volumes: { dbdata: {} }
```

### Git workflow

- **Trunk-based nhẹ**: nhánh `main` luôn deploy được. Mỗi tính năng một nhánh `feat/vote-weight`, PR nhỏ, merge nhanh.
- **Commit** dạng conventional: `feat(vote): auto-hide by community threshold`, `fix(feed): keyset cursor off-by-one`.
- **PR checklist**: có test, `make lint` sạch, migration có `.down`, không log lộ secret, không `TODO` cho phần bảo mật.
- Mỗi giai đoạn = một milestone; không nhảy giai đoạn khi DoD chưa đạt.

### CI (GitHub Actions — khung)

```yaml
name: ci
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          {
            POSTGRES_USER: food,
            POSTGRES_PASSWORD: food,
            POSTGRES_DB: foodsocial,
          }
        ports: ["5432:5432"]
        options: >-
          --health-cmd "pg_isready -U food" --health-interval 5s --health-retries 10
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - run: go vet ./...
      - run: go test ./...
      - run: |
          go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
          migrate -path migrations -database "$DATABASE_URL" up
        env:
          {
            DATABASE_URL: "postgres://food:food@localhost:5432/foodsocial?sslmode=disable",
          }
      - run: go test -tags=integration ./test/...
        env:
          {
            DATABASE_URL: "postgres://food:food@localhost:5432/foodsocial?sslmode=disable",
          }
```

---

## Phần D — Deploy (sau khi backend xong)

Đơn giản, đủ dùng, rẻ — hợp một dự án solo:

1. **Build**: `Dockerfile` multi-stage (build Go tĩnh → image `scratch`/`distroless` ~15MB).
2. **DB**: managed Postgres (Neon / Supabase / Railway free tier) — không tự vận hành Postgres lúc đầu.
3. **App**: chạy container trên Fly.io / Railway / một VPS nhỏ. Env qua secret, không commit `.env`.
4. **Migration on deploy**: chạy `migrate up` trong bước release (một job riêng, trước khi start app mới).
5. **Storage ảnh**: Cloudflare R2 (rẻ, không phí egress) — cấu hình CORS cho presigned PUT.
6. **Quan sát**: log JSON (slog) gom về nơi xem được; thêm `/metrics` (prometheus) khi cần. `request_id` trong mọi log để trace.
7. **Backup**: bật auto-backup DB của managed provider.

```dockerfile
# Dockerfile multi-stage
FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app ./cmd/api

FROM gcr.io/distroless/static
COPY --from=build /app /app
EXPOSE 8080
ENTRYPOINT ["/app"]
```

---

## Phần E — Sau backend: Mobile trước, Web sau

Vì đã chọn mobile-first, khi API v1 ổn định:

**Bước 1 — Chốt hợp đồng API.** Xuất một file OpenAPI/`api.http` mô tả đúng Phần 03. Đây là "nguồn sự thật" giữa backend và client. Không đổi shape bừa sau khi mobile bắt đầu tiêu thụ.

**Bước 2 — Mobile app.** Gợi ý stack (không bắt buộc):

- **Flutter** (bạn đã có kinh nghiệm Flutter từ Drama Farm) — một codebase Android/iOS.
- Kiến trúc client: repository layer gọi API + cache (Isar/Drift), state (Riverpod/Bloc), infinite scroll dùng `next_cursor`.
- Ưu tiên màn: feed → chi tiết bài → đăng bài → đề xuất/accept địa điểm → vote. Đúng thứ tự tạo "aha moment" của sản phẩm.
- Xử lý offline nhẹ: cache feed gần nhất, gửi POST kèm `Idempotency-Key` để retry an toàn.

**Bước 3 — Web sau.** Dùng lại **y hệt API v1**. Chỉ là client khác (Next.js/SvelteKit/React). SEO cho trang place công khai là điểm cộng — server-render trang place để Google index "Ốc Thủy Dương Hải Phòng".

**Bước 4 — Push notification.** Khi có app thật: thêm cột `pushed_at` vào `notifications`, viết worker đọc dòng chưa push → gửi FCM/APNs. Data model không đổi (đã thiết kế sẵn ở Phần 02).

---

## Phần F — Thứ tự học/luyện gắn với từng giai đoạn

Bạn đang học Go, nên đây là "map" kỹ năng → giai đoạn để không học lan man:

| Giai đoạn | Kỹ năng Go/SQL luyện được                                                                  |
| --------- | ------------------------------------------------------------------------------------------ |
| 0–1       | struct, interface, DI thủ công, context, middleware, error wrapping, bcrypt, `crypto/rand` |
| 2         | `database/sql` scan, cursor pagination, soft delete, optimistic lock, partial index        |
| 3         | transaction đa bảng, counter đồng bộ, `ON CONFLICT`, many-to-many                          |
| 4         | `SELECT FOR UPDATE`, upsert, canonical resolve, full-text search, atomic multi-step flow   |
| 5         | row locking dưới tải, aggregate trong transaction, table-driven test, `Clock` interface    |
| 6         | JSONB, audit pattern, dedup, GIN index, phân trang toàn hệ thống                           |
| B–D       | mock/interface test, integration test, testcontainers, CI, Docker, deploy                  |

---

## Tóm tắt một dòng cho mỗi file

- `README` — sản phẩm là gì, đã nâng cấp gì, ràng buộc công nghệ, nguyên tắc xuyên suốt.
- `00a` — quyết định sản phẩm: công khai/riêng tư, xử lý nội dung xóa/ẩn, block/mute, place approval, nhãn tài trợ, ma trận phân quyền.
- `00b` — bảo mật & quyền riêng tư: xác thực, quên mật khẩu, xóa tài khoản, chính sách dữ liệu, văn bản pháp lý.
- `01` — code trông thế nào: 3 tầng, DI thủ công, repository qua `Querier`, error/envelope/cursor, quy tắc Go.
- `02` — DB thế nào: schema đã nâng cấp + lý do từng quyết định, index, concurrency, migration order.
- `03` — API thế nào: endpoint v1, envelope, phân quyền, luồng lõi (accept suggestion, vote).
- `04` (file này) — làm theo thứ tự nào, DoD, test, CI/CD, deploy, rồi mobile → web.

Bắt đầu từ Giai đoạn 0. Đừng nhảy cóc — mỗi giai đoạn dựng trên móng của giai đoạn trước, và DoD là cách bạn tự biết mình đang viết code chuẩn chứ không chỉ "chạy được".
