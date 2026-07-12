-- Bảng posts: bài review đồ ăn — trung tâm của sản phẩm.
-- Địa điểm KHÔNG bắt buộc lúc đăng: place_id NULL + location_status='UNKNOWN',
-- người khác đề xuất địa điểm, chủ bài xác nhận → location_status='CONFIRMED'.
CREATE TABLE posts (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    place_id        BIGINT REFERENCES places(id),          -- NULL khi chưa biết địa điểm
    province_id     BIGINT REFERENCES provinces(id),       -- feed theo tỉnh khi chưa có place (xem luật bên dưới)
    content         TEXT NOT NULL,
    status          VARCHAR(30) NOT NULL DEFAULT 'VISIBLE',
    location_status VARCHAR(20) NOT NULL DEFAULT 'UNKNOWN',
    version         INTEGER NOT NULL DEFAULT 1,             -- optimistic lock: chống lost-update khi sửa bài
    is_sponsored    BOOLEAN NOT NULL DEFAULT FALSE,         -- tự khai báo nội dung tài trợ

    -- Aggregate denormalized. Nguồn sự thật là các bảng con (review_votes, post_likes...);
    -- các cột này là cache, LUÔN cập nhật trong cùng transaction với thao tác gốc.
    trusted_weight   DECIMAL(12,2) NOT NULL DEFAULT 0,
    untrusted_weight DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_vote_count INTEGER       NOT NULL DEFAULT 0,
    untrusted_ratio  DECIMAL(6,4)  NOT NULL DEFAULT 0,
    like_count       INTEGER       NOT NULL DEFAULT 0,
    comment_count    INTEGER       NOT NULL DEFAULT 0,
    save_count       INTEGER       NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    hidden_at  TIMESTAMPTZ,                                 -- thời điểm bị ẩn (community/admin)
    deleted_at TIMESTAMPTZ,                                 -- soft delete
    CHECK (status IN ('VISIBLE','HIDDEN_BY_COMMUNITY','HIDDEN_BY_ADMIN','DELETED_BY_AUTHOR')),
    CHECK (location_status IN ('UNKNOWN','SUGGESTED','CONFIRMED'))
);

-- Feed: partial index chỉ bài đang hiện, sắp theo keyset (created_at, id) DESC cho cursor pagination.
CREATE INDEX ix_posts_feed     ON posts(created_at DESC, id DESC) WHERE status = 'VISIBLE';
CREATE INDEX ix_posts_user     ON posts(user_id)     WHERE deleted_at IS NULL;
CREATE INDEX ix_posts_place    ON posts(place_id)    WHERE status = 'VISIBLE';
CREATE INDEX ix_posts_province ON posts(province_id) WHERE status = 'VISIBLE';

-- Full-text search nội dung bài, bỏ dấu. Cột generated: repository KHÔNG insert cột này.
ALTER TABLE posts ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('simple', unaccent(coalesce(content,'')))) STORED;
CREATE INDEX ix_posts_search ON posts USING GIN(search_vector);

-- Luật province_id vs place_id: khi place_id đã có, tỉnh hiển thị lấy từ places.province_id (JOIN);
-- posts.province_id chỉ dùng cho feed theo tỉnh khi bài CHƯA xác nhận địa điểm.

-- ------------------------------------------------------------------
-- post_images: ảnh của bài, tham chiếu media_assets đã USABLE (không lưu URL thô).
-- ------------------------------------------------------------------
CREATE TABLE post_images (
    id         BIGSERIAL PRIMARY KEY,
    post_id    BIGINT NOT NULL REFERENCES posts(id),
    media_id   BIGINT NOT NULL REFERENCES media_assets(id),
    sort_order INTEGER NOT NULL DEFAULT 0,        -- thứ tự hiển thị ảnh trong bài
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX        ix_post_images_post  ON post_images(post_id);
CREATE UNIQUE INDEX ux_post_images_media ON post_images(media_id);  -- 1 ảnh chỉ gắn 1 bài

-- ------------------------------------------------------------------
-- hashtags + post_hashtags: quan hệ nhiều-nhiều giữa bài và thẻ.
-- ------------------------------------------------------------------
CREATE TABLE hashtags (
    id         BIGSERIAL PRIMARY KEY,
    tag        VARCHAR(100) NOT NULL UNIQUE,      -- lưu lowercase, KHÔNG kèm dấu '#'
    post_count INTEGER NOT NULL DEFAULT 0,        -- denormalized: đếm bài theo tag
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE post_hashtags (
    post_id    BIGINT NOT NULL REFERENCES posts(id),
    hashtag_id BIGINT NOT NULL REFERENCES hashtags(id),
    PRIMARY KEY (post_id, hashtag_id)
);
CREATE INDEX ix_post_hashtags_hashtag ON post_hashtags(hashtag_id);
