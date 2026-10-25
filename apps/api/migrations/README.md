# PostgreSQL migrations

SQL migrations are managed by Goose, separately from starting the API.
Accounts are created with `pnpm db:create-user`, never in migrations.

From the repository root:

```powershell
pnpm db:create create_users
# Review and fill the generated Up/Down SQL before running it.
$env:MIGRATION_DATABASE_URL = '<direct PostgreSQL connection URL with TLS>'
pnpm db:status
pnpm db:up
# Explicit rollback of the last applied migration:
pnpm db:down
```

Use the **direct** Neon connection, not the `-pooler` endpoint: the migration
runner uses a session-level PostgreSQL advisory lock to serialize migrations.
The app's `DATABASE_URL` can use a pooled connection. No fallback between the
two variables is performed.

With no SQL files, status/up/down report that there are no migrations and do
not connect to a database. Once SQL files exist, Goose may initialize its own
version-tracking table, including when reading status. Nothing runs on API startup.

Keep migration files in this directory. The Node scripts set the working directory
to `apps/api`; a direct invocation of the Go command must run from there too.
