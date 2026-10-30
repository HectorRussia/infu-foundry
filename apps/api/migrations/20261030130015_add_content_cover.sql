-- +goose Up
-- Optional cover image of a content. The image lives in Cloudflare R2 under
-- covers/<content id>/<random>.<ext>; only the object key is stored here.
ALTER TABLE contents ADD COLUMN cover_key text;
ALTER TABLE contents ADD CONSTRAINT contents_cover_key_format
    CHECK (cover_key IS NULL OR cover_key LIKE 'covers/%');

-- +goose Down
ALTER TABLE contents DROP COLUMN cover_key;
