# Validation — 25 October 2026

## Local dev login (latest)

- Development-only account enables UI login without Go/Neon calls. Production still has disabled login and public preview workspace routes
- Browser verified invalid password rejection, valid login to library, session surviving refresh, logout, and direct editor access redirecting to login after logout
- Session marker only in sessionStorage; password is not persisted. This is not a security boundary
- Web check and static build passed; production output scanned and contains neither dev email nor dev password
- Local Workers production preview at port 8787 confirmed disabled login and no development-account hint
- Browser error log empty after the flow

## Login and workspace separation (latest UI)

- User confirmed UI-only scope; authentication remains deferred
- `/` and `/login/` match the supplied welcome/form design without sidebar; only workspace routes render the sidebar
- Desktop sidebar has contents/editor links; mobile uses a native collapsible menu that resets on route changes
- In-app browser: compared login at 506×726 and sidebar at 1217×888 against supplied images; tested desktop editor navigation and mobile menu → library navigation
- At 320px, library and login had no horizontal overflow; login had zero sidebar elements. Final browser error log was empty
- Web lint, TypeScript and static build passed. Visual comparison recorded in `design-qa.md`
- Workspace routes remain public previews until real authentication is implemented; no fake login/session was added

## Simplified MVP screens (supersedes the navigation described below)

- `/` now shows only the login card; `/login/` uses the same component. No dashboard, sidebar or bottom navigation
- Workspace contains the contents library and title/Content editor, with direct links between them
- Web check (ESLint, TypeScript) and static production build passed
- Browser checked login and editor at 390px and 320px widths with no measured horizontal overflow; login also captured at 1440×1000
- Clicked library → editor → library successfully. Editor has exactly title/content fields; login and save stay disabled
- No browser console errors in final login check. Screenshots: `output/playwright/simple-*.png`
- Login-to-library authentication flow remains planned, not implemented; workspace URLs are still public preview routes

## Dark mobile UI update

- All preview pages use Tailwind utilities with shared dark theme tokens; editor fields are title and Content only
- `pnpm --filter @infu-foundry/web check` and `pnpm --filter @infu-foundry/web build` passed
- Chrome viewport checks: editor at 320×640, 390×844, 768×1024 and desktop at 1440×1000; mobile home, contents and login at 390×844
- No horizontal overflow in measured mobile/tablet layouts; mobile editor inputs use 16px text, action buttons are 48px high, bottom navigation targets are 72px high
- Typed Thai content, copied it, and pasted it back into the title field to verify actual clipboard output; save remained disabled
- Refresh cleared both fields as expected; login remained disabled and all preview notices remained visible
- Production editor, contents and login opened through local Workers static preview; editor-to-contents client navigation succeeded; final production editor console had no errors or warnings
- Local home showed the running API as connected
- Screenshots in `output/playwright/dark-*.png`; viewport checks are desktop Chrome simulations, not physical-device or mobile-keyboard verification
- This UI change does not verify real Containers, Neon, authentication or persistence

## Backend stack update — Fiber / GORM / Goose

- Project toolchain: Go 1.26.10 (selected automatically; global Go installation unchanged)
- Fiber v3.6.1, GORM v1.31.2, PostgreSQL driver v1.6.3, Goose v3.29.0
- `pnpm check` passed after migration, including Fiber HTTP boundaries, panic sanitization, database cancellation/URL validation and migration command guards
- `pnpm build` passed, compiling both server and migration binaries and bundling the Worker in dry-run mode
- Linux amd64 cross-builds of server and migrator passed with `CGO_ENABLED=0`; this does not verify Docker image execution
- Live Fiber health returned 200 / `{"status":"ok"}` / `Cache-Control: no-store`; HEAD returned 200; unknown API returned 404
- Live CORS allowed localhost:3000 and rejected a foreign origin with 403
- `pnpm dev` starts the migrated API without DATABASE_URL; the existing editor route returned 200
- `pnpm db:status` correctly reported no SQL migrations and made no database connection
- Migration-template tests created files only in temporary test directories; no application schema or real database was modified
- GORM startup ping and Goose SQL execution against a live PostgreSQL/Neon database remain unverified; no connection credentials were configured
- The Worker can forward an optional DATABASE_URL secret to the Container. No secret was set and real Containers remain unverified on this Windows environment

The remaining sections record the initial frontend/scaffold verification.

## Environment

- Windows, Node.js 24.18.0, pnpm 10.32.1, Go 1.24.1
- Next.js 16.4.3, React 19.3.1, Wrangler 4.152.0
- Docker CLI installed; Docker engine unavailable during verification
- Wrangler local Containers also reject native Windows; WSL/Linux/macOS is required

## Completed checks

- `pnpm check`: Next route type generation, ESLint, web/gateway TypeScript, gateway routing tests, Go vet and Go HTTP tests
- `pnpm build`: Next static export, Windows RSC filename normalization, local Go binary and Worker dry-run bundle
- `pnpm dev`: starts both web/API; stopping the command released ports 3000 and 8080
- Live Go: `/api/v1/health` responds 200 with `{"status":"ok"}`
- Browser: API status shows connected while Go runs, disconnected after stopping Go, and recovers via the retry button after restarting Go
- Editor: text entry and Prompt copy work; save stays disabled; refresh clears fields; no localStorage entries are created
- Login: labeled fields and disabled submit; visible notice that authentication is not implemented
- Desktop (1440×1050) and mobile (390×844) visually inspected; no horizontal overflow on mobile home/editor
- Static assets served through the actual local Workers runtime: home, login, contents and editor respond 200; missing route responds 404
- Static client-side navigation between contents and editor, followed by direct refresh: no browser console errors after fixing the Windows export filename issue
- Gateway without Containers: API responds 503 with the expected JSON rather than an HTML fallback
- Gateway tests verify preservation of method, query, body, cookie headers, response status and Set-Cookie; missing assets retain 404; backend failures are not retried
- Go tests cover local CORS allow/deny, preflight, unknown route, unsupported method and production CORS behavior

## Discovered and fixed

Next.js exports nested RSC segment files on Windows but its client requests dot-separated names.
This produced real 404s during production client navigation. The web build now runs a targeted,
idempotent alias step without modifying framework source or deleting generated files.
Static navigation was tested again on Workers with no console errors.
Reference: https://github.com/vercel/next.js/issues/85374

## Not verified

- Docker image build and a healthy real Cloudflare Container end-to-end request
- Production deployment, Cloudflare account configuration, Neon connectivity
- Authentication and persistence (deliberately outside this scaffold)

`pnpm build` uses `--dry-run --containers-rollout=none`; passing it confirms Worker bundling,
not Container execution. `pnpm preview:assets` explicitly disables Containers and is not
an end-to-end substitute for `pnpm preview:cloudflare` on a supported Docker environment.

Browser captures are local, ignored artifacts in `output/playwright/`.
