-- +goose Up
-- Every signed-in user sees all contents; author_id only records who wrote it.
CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text NOT NULL,
    display_name  text NOT NULL,
    password_hash text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_not_blank CHECK (btrim(email) <> ''),
    CONSTRAINT users_display_name_not_blank CHECK (btrim(display_name) <> '')
);

CREATE UNIQUE INDEX users_email_key ON users (lower(email));

CREATE TABLE contents (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id  uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    title      text NOT NULL DEFAULT '',
    body       text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE INDEX contents_recent_idx ON contents (updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX contents_author_id_idx ON contents (author_id);

-- +goose Down
DROP TABLE contents;
DROP TABLE users;
