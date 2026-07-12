-- Bảng places: địa điểm ăn uống. Nhiều bài review gom về một place.
-- Place có thể do user tạo (hậu kiểm, không duyệt trước) hoặc gắn từ Google Places.

CREATE TABLE places (
    id                 BIGSERIAL PRIMARY KEY,
    canonical_place_id BIGINT REFERENCES places(id),   -- NULL = tự nó là canonical (place gốc)
    google_place_id    VARCHAR(255),
    name               VARCHAR(255) NOT NULL,
    address            TEXT,
    province_id        BIGINT REFERENCES provinces(id),
    district_id        BIGINT REFERENCES districts(id),
    ward_id            BIGINT REFERENCES wards(id),
    latitude           DECIMAL(10,7),
    longitude          DECIMAL(10,7),
    post_count         INTEGER NOT NULL DEFAULT 0,      -- denormalized cho trang place
    status             VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (status IN ('ACTIVE','MERGED','HIDDEN'))
);

-- google_place_id unique NHƯNG cho phép nhiều NULL (place nhập tay chưa có Google ID).
-- Partial unique index: chỉ ràng buộc khi google_place_id IS NOT NULL.
CREATE UNIQUE INDEX ux_places_google    ON places(google_place_id) WHERE google_place_id IS NOT NULL;
CREATE INDEX        ix_places_canonical ON places(canonical_place_id);
CREATE INDEX        ix_places_province  ON places(province_id);

-- Full-text search tên + địa chỉ, bỏ dấu tiếng Việt ("pho" khớp "phở").
-- Cột generated: Postgres tự tính, repository KHÔNG insert/update cột này.
ALTER TABLE places ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', unaccent(coalesce(name,'') || ' ' || coalesce(address,'')))
    ) STORED;
CREATE INDEX ix_places_search ON places USING GIN(search_vector);

-- Trgm để gợi ý place trùng tên khi admin gộp (canonical merge).
CREATE INDEX ix_places_name_trgm ON places USING GIN(name gin_trgm_ops);

-- Lịch sử gộp place: ai gộp B vào A, lúc nào, vì sao (audit).
-- Luật: chỉ merge MỘT cấp — canonical_place_id luôn trỏ thẳng tới place gốc thật, không tạo chuỗi A→B→C.
CREATE TABLE place_merge_history (
    id                 BIGSERIAL PRIMARY KEY,
    merged_place_id    BIGINT NOT NULL REFERENCES places(id),  -- place bị gộp (B)
    canonical_place_id BIGINT NOT NULL REFERENCES places(id),  -- place giữ lại (A)
    merged_by          BIGINT NOT NULL REFERENCES users(id),   -- admin thực hiện
    reason             TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_place_merge_merged    ON place_merge_history(merged_place_id);
CREATE INDEX ix_place_merge_canonical ON place_merge_history(canonical_place_id);
