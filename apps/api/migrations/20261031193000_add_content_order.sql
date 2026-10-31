-- +goose Up
-- Shared library order. Every account sees the same pins and the same order.
-- pinned rows are listed first. sort_key is a gap-based position (multiples of
-- 65536 at first): a move writes a key between two neighbours and updates only
-- that row. The group is rewritten only when no integer fits in the gap.
--
-- Existing rows, including the trash, keep today's last-edited order
-- (updated_at DESC, id). Live rows therefore stay in the order the library
-- showed before this migration. New inserts that omit sort_key (the previous
-- API) are placed at the top of their pin group by the trigger below, under
-- the same transaction advisory lock the new API uses (key 20261031, matching
-- libraryOrderLock in internal/contents/order.go). That lock is transaction
-- scoped, so it is safe on the Neon pooler.
--
-- Apply this after 20261031175500_add_categories. The new API reads these
-- columns, so run db:up before that API starts. Adding the columns is safe
-- while the previous API is still running.
--
-- Restoring a content keeps its pin. It returns to this sort_key when no live
-- row in that pin group already has it; otherwise it goes to the top of the group.

ALTER TABLE contents
  ADD COLUMN pinned boolean NOT NULL DEFAULT false,
  ADD COLUMN sort_key bigint;

WITH ordered AS (
  SELECT id, row_number() OVER (ORDER BY updated_at DESC, id) AS rn
  FROM contents
)
UPDATE contents AS c
SET sort_key = ordered.rn * 65536
FROM ordered
WHERE c.id = ordered.id;

ALTER TABLE contents ALTER COLUMN sort_key SET NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION contents_assign_sort_key() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
  min_key bigint;
BEGIN
  IF NEW.sort_key IS NOT NULL THEN
    RETURN NEW;
  END IF;
  IF NEW.pinned IS NULL THEN
    NEW.pinned := false;
  END IF;
  -- Same lock id as libraryOrderLock in internal/contents/order.go.
  PERFORM pg_advisory_xact_lock(20261031);
  SELECT min(sort_key) INTO min_key
  FROM contents
  WHERE deleted_at IS NULL
    AND pinned = NEW.pinned
    AND id IS DISTINCT FROM NEW.id;
  IF min_key IS NULL THEN
    NEW.sort_key := 0;
  ELSE
    NEW.sort_key := min_key - 65536;
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER contents_assign_sort_key
  BEFORE INSERT ON contents
  FOR EACH ROW
  EXECUTE FUNCTION contents_assign_sort_key();

CREATE UNIQUE INDEX contents_live_sort_key
  ON contents (pinned, sort_key)
  WHERE deleted_at IS NULL;

CREATE INDEX contents_library_order_idx
  ON contents (pinned DESC, sort_key ASC, id ASC)
  WHERE deleted_at IS NULL;

-- +goose Down
DROP TRIGGER IF EXISTS contents_assign_sort_key ON contents;
DROP FUNCTION IF EXISTS contents_assign_sort_key();
DROP INDEX IF EXISTS contents_library_order_idx;
DROP INDEX IF EXISTS contents_live_sort_key;
ALTER TABLE contents DROP COLUMN IF EXISTS sort_key;
ALTER TABLE contents DROP COLUMN IF EXISTS pinned;
