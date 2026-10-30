-- +goose Up
-- Videos attached to contents. The file lives in Cloudflare R2; only its object
-- key is stored here, and viewing URLs are short-lived presigned links.
CREATE TABLE media (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id   uuid NOT NULL REFERENCES contents (id) ON DELETE RESTRICT,
    uploader_id  uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    object_key   text NOT NULL UNIQUE,
    filename     text NOT NULL,
    content_type text NOT NULL,
    size_bytes   bigint NOT NULL,
    status       text NOT NULL DEFAULT 'pending',
    created_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz,
    CONSTRAINT media_status_check CHECK (status IN ('pending', 'ready')),
    CONSTRAINT media_content_type_check CHECK (content_type IN ('video/mp4', 'video/quicktime', 'video/webm')),
    CONSTRAINT media_size_check CHECK (size_bytes > 0 AND size_bytes <= 1073741824),
    CONSTRAINT media_filename_not_blank CHECK (btrim(filename) <> '')
);

CREATE INDEX media_content_id_idx ON media (content_id, created_at) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE media;
