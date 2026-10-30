-- +goose Up
-- Which part of the cover image stays visible when a card crops it, as CSS
-- object-position percentages (0 = left/top, 50 = centre, 100 = right/bottom).
ALTER TABLE contents
    ADD COLUMN cover_position_x smallint NOT NULL DEFAULT 50,
    ADD COLUMN cover_position_y smallint NOT NULL DEFAULT 50,
    ADD CONSTRAINT contents_cover_position_range
        CHECK (cover_position_x BETWEEN 0 AND 100 AND cover_position_y BETWEEN 0 AND 100);

-- +goose Down
ALTER TABLE contents
    DROP COLUMN cover_position_x,
    DROP COLUMN cover_position_y;
