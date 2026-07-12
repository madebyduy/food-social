-- Bảng media_assets: ảnh upload theo luồng an toàn (presign → PUT thẳng lên storage → confirm).
-- Post KHÔNG lưu URL tự do từ client; post chỉ tham chiếu media_id đã ở trạng thái USABLE.
--
-- Vòng đời status:
--   PENDING  : vừa presign, chờ client PUT ảnh + gọi confirm.
--   USABLE   : server đã tải lại, kiểm MIME/size thật, bỏ EXIF/GPS → cho phép gắn vào post.
--   REJECTED : không hợp lệ (sai MIME/size, nội dung xấu...).
-- media_assets PENDING quá lâu (vd 24h) sẽ bị job dọn rác xóa khỏi storage.
CREATE TABLE media_assets (
    id           BIGSERIAL PRIMARY KEY,
    owner_id     BIGINT NOT NULL REFERENCES users(id),  -- chỉ chủ sở hữu mới gắn được vào bài của mình
    storage_key  VARCHAR(500) NOT NULL UNIQUE,          -- key object trên S3/R2
    mime_type    VARCHAR(50),
    size_bytes   INTEGER,
    width        INTEGER,
    height       INTEGER,
    has_gps      BOOLEAN NOT NULL DEFAULT FALSE,         -- ảnh gốc có EXIF GPS không (đã bỏ khi confirm)
    status       VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at TIMESTAMPTZ,
    CHECK (status IN ('PENDING','USABLE','REJECTED'))
);
-- Truy vấn "ảnh usable của user này" hay dọn rác "PENDING quá hạn" đều lọc theo (owner, status).
CREATE INDEX ix_media_assets_owner ON media_assets(owner_id, status);
