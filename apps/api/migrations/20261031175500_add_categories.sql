-- +goose Up
-- Shared folders. A content has at most one category: content_id is the primary
-- key of the link table, not a column on contents. Dropping that primary key for
-- PRIMARY KEY (content_id, category_id) is enough to allow many tags later,
-- without rewriting contents.
--
-- Additive only (new empty tables). Safe on the live database while the previous
-- API is still running. Deploy the API that joins these tables only after this
-- migration is applied.
--
-- Deleting a category cascades to content_categories only, so its contents stay
-- and become uncategorized. Deleting a content row (permanent purge) removes its
-- link. A soft delete does not, so restoring a content keeps its category.
CREATE TABLE categories (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT categories_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT categories_name_length CHECK (char_length(name) <= 80)
);

CREATE UNIQUE INDEX categories_name_key ON categories (lower(btrim(name)));

CREATE TABLE content_categories (
    content_id  uuid PRIMARY KEY REFERENCES contents (id) ON DELETE CASCADE,
    category_id uuid NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX content_categories_category_id_idx ON content_categories (category_id);

-- +goose Down
DROP TABLE content_categories;
DROP TABLE categories;
