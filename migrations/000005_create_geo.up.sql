-- Địa lý phân cấp: provinces (tỉnh/thành) → districts (quận/huyện) → wards (phường/xã).
-- Data model hỗ trợ nhiều tỉnh ngay từ đầu, dù v1 chỉ chạy Hải Phòng.
-- posts.province_id và places.{province_id,district_id,ward_id} sẽ tham chiếu các bảng này.

CREATE TABLE provinces (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(120) NOT NULL UNIQUE,      -- định danh không dấu, dùng cho URL/feed theo tỉnh
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE districts (
    id          BIGSERIAL PRIMARY KEY,
    province_id BIGINT NOT NULL REFERENCES provinces(id),
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(120) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (province_id, slug)                    -- slug chỉ cần duy nhất trong 1 tỉnh
);
CREATE INDEX ix_districts_province ON districts(province_id);

CREATE TABLE wards (
    id          BIGSERIAL PRIMARY KEY,
    district_id BIGINT NOT NULL REFERENCES districts(id),
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(120) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (district_id, slug)
);
CREATE INDEX ix_wards_district ON wards(district_id);
