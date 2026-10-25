-- Development-only author used by the API while APP_ENV=development.
-- Not a migration: run manually once per database/branch, never in production.
INSERT INTO users (email, display_name)
VALUES ('dev@infu.local', 'Dev')
ON CONFLICT ((lower(email))) DO NOTHING;
