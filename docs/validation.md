# Validation — 31 October 2026

## Confirm dialog follow-up

- The source scan also rejects bare `alert(` and `prompt(`. `confirm(` stays allowed because that is `useConfirm()`. `window.alert` / `window.confirm` / `window.prompt` stay rejected
- Brand hover is the `bg-brand-hover` token (`--color-brand-hover: #ffa383`) on the confirm button, the editor save button, and “เขียนเนื้อหา”. The hard-coded `#ffa383` class is gone
- A second confirm or notice while one is already open is dropped, so a double-click in the same turn cannot queue another dialog. `beginRequest` covers that
- Web tests 12/12 with the library order tests, including the scan and the dropped second request. `pnpm check` and `pnpm build` passed on Node.js 24.22.2
- Not verified: production Neon, a browser click-through, a physical phone, and any deploy. Copy and layout are unchanged

## In-app confirm dialog

- Every `window.confirm` in `apps/web` now goes through one promise-based `useConfirm()`. `useAlert()` is the notice equivalent. There was no `window.alert` or `window.prompt` in the app. The dialog is a native `<dialog>` opened with `showModal()` (`role="alertdialog"`, labelled, focus moves in and is trapped, Esc and backdrop cancel unless an action is in flight, focus returns to the trigger, body scroll locked). Mobile is a full-width bottom sheet; buttons are `min-h-12`. Irreversible actions use the danger style and focus “ยกเลิก” first. Moving a content to the trash stays a reversible confirm (brand button, focus on “ลบเนื้อหา”) because the existing copy says it can be restored from ถังขยะ
- Copy is unchanged: category delete still says the contents become uncategorized and are not deleted or moved to the trash; trash purge and empty-trash still warn that covers and videos are removed and cannot be restored; video delete stays permanent; cover delete stays “ลบภาพปก?”
- Web tests cover confirm defaults, danger focus, the pending label “กำลังทำ…”, dismiss blocked while pending or closing, the queue, reduced motion, and a source scan that `window.alert` / `window.confirm` / `window.prompt` are gone (11/11 with the library order tests). Gateway tests 7/7. `go vet` and Go tests passed
- `pnpm check` and `pnpm build` passed on Node.js 24.22.2 (static export, Go binaries, Worker dry-run)
- Browser, against a local mock API (not Neon): category delete on a 1280px desktop viewport and a 390px phone viewport. Esc and the backdrop dismiss and return focus to “ลบหมวดหมู่”. The danger confirm opens with focus on “ยกเลิก”. Tab wraps between ยกเลิก and ลบหมวดหมู่. While the delete request is in flight the confirm button reads “กำลังทำ…” and Esc / the backdrop do nothing. Editor move-to-trash focuses “ลบเนื้อหา” (reversible). Cover delete, video delete, trash purge, and empty-trash open with focus on “ยกเลิก” and the existing Thai copy
- Not verified: production Neon, a browser click-through against the live API, a physical phone, and any deploy

# Validation — 25 October 2026

## Library pin and manual order

- Shared by every account. `contents.pinned` plus gap-based `sort_key` (initial gap 65536). Pinned contents stay above unpinned ones. Ordering is inside each group. A category filter keeps that relative order and only hides other rows
- New content is unpinned and lands at the top of the unpinned group, under the pins. Editing a title or body does not move it. One drag or arrow move updates only the moved row. The pin group is rewritten only when no integer fits between the neighbours. Writers take transaction advisory lock `20261031` (the same id as the insert trigger), so concurrent moves cannot take the same key. The lock is transaction-scoped and works on the Neon pooler
- Migration `add_content_order` (`20261031193000`, after `add_categories`) backfills every row, including trash, in the previous last-edited order (`updated_at DESC, id`). Down drops the trigger, function, indexes, then the two columns. Existing live contents therefore keep today's library order. The previous API can still insert after the migration because the trigger fills `sort_key`. The new API must not start before `db:up`, or the library and editor fail on the new columns
- Trash keeps the pin and the sort key. Restore puts the content back in that place when the key is free in its pin group; if a live row already has it, the content goes to the top of that group. Videos, covers and the editor are unchanged
- API: PATCH `/contents/{id}/pin` `{pinned}`, PATCH `/contents/{id}/order` `{afterId}` (`null` = top of the same pin group). A neighbour in the other group is 422. List is `pinned DESC, sort_key ASC, id ASC`, max 200
- Web: pin control, drag handle (pointer events, `touch-action: none`, works with a finger) and up/down buttons on each library card. Dragging stays inside the pin group. Thai labels match the rest of the library
- Go tests cover placement (one-row moves, rebalance when the gap is 1, new keys at the top, filtered order, scratch keys that do not collide, concurrent moves under a mutex standing in for the advisory lock) and the HTTP routes (session, 422, 404, pin, unpin, move, category filter). Gateway tests 7/7, including PATCH `/contents/{id}/order`. Web order helpers 3/3
- `pnpm check` passed (ESLint, TypeScript, the web order tests, gateway tests, `go vet`, Go tests) and `pnpm build` passed (static export, Go binaries, Worker dry-run). This environment ran Node 22.23.4, so pnpm warned that the repo asks for Node >= 24; the commands still exited 0
- Browser, against a local mock API (not Neon): on a phone-width touch viewport, dragging หนึ่ง onto สอง reordered the unpinned group to สอง then หนึ่ง under the pinned card; pinning สอง moved it to the top of the pinned group; filtering เรื่อง hid สอง and kept สาม then หนึ่ง; clearing the filter restored สอง, สาม, หนึ่ง. Desktop showed the two pinned cards side by side. Arrow buttons at the ends of a group were disabled
- Not yet: `db:up` on the live Neon database (needs the owner's approval; do not start the new API or push this API to Render before that), a physical phone, and production. This PR is stacked on the categories branch; merge categories first so `add_categories` is applied before `add_content_order`

## Categories

- A content has at most one shared category, optional. The link lives in `content_categories` (`content_id` primary key) so a later many-to-many tag model does not rewrite `contents`. No nested categories
- Migration `add_categories` creates `categories` and `content_categories` only. Down drops the link table, then `categories`. Deleting a category cascades to the link, so contents stay and become uncategorized. Existing rows have no link, so list, editor, trash, videos and covers keep working. Soft-deleted contents keep the link; a permanent purge removes it with the content row
- API: GET/POST `/categories`, PATCH/DELETE `/categories/{id}`; GET `/contents?category=` (`uncategorized` or a UUID); create/update accept `categoryId` (omit leaves it, null clears it, unknown UUID is 422)
- Web: library chips for all / uncategorized / each category, plus create, rename and delete; editor select and “สร้างหมวดหมู่ใหม่”. Inputs stay 16px. Deleting a category tells the user contents are not deleted or trashed
- Go tests: session required, name validation, duplicate names (case-insensitive), rename, omit vs null vs assign, a rejected category does not change the content, filters, and deleting a category leaves the content uncategorized. Gateway tests 6/6, including `/categories` forwarded like other `/api` routes
- `pnpm check` passed (ESLint, TypeScript, gateway tests, `go vet`, Go tests) and `pnpm build` passed (static export of `/contents` and `/contents/editor`, Go server and migrate binaries, Worker dry-run). This environment ran Node 22.23.4, so pnpm warned that the repo asks for Node >= 24; the commands still exited 0
- Not yet: migration on the live Neon database (needs the owner’s approval before `pnpm db:up`), a browser click-through, and production. Do not merge until `db:up` has run, or the library and editor break on the Render deploy
- After merging `origin/main` (CI workflow and project rules) into this branch: `pnpm check` and `pnpm build` passed again on Node 22.23.4 (pnpm warned that the repo asks for Node >= 24). The merge changed docs only; no migration or application code

## Cursor project rules

- Moved the detailed agent procedures out of `CLAUDE.md` into `.cursor/rules/` (one `.mdc` file per topic). `CLAUDE.md` is now an index; `AGENTS.md` points at it and at that directory
- Rebased onto `main` at `7a5ff2c` so the CI workflow and this rules index both stay. The GitHub Actions section below is unchanged
- `pnpm check` passed on Node.js 24.22.2 after that rebase: web typegen, ESLint, and `tsc`; gateway tests 5/5; `go vet` and Go tests. `pnpm build` passed: static export, Go binaries, Worker dry-run. No application code changed
- Not verified: Cursor attaching each rule from `alwaysApply` or `globs` in a live chat. No production Neon, no browser click-through, no deploy

## GitHub Actions: job `check`

- Workflow `.github/workflows/ci.yml` (ชื่อ `CI`) ทำงานเมื่อ `pull_request` ทุก base และ `push` ขึ้น `main`
- มี job เดียวชื่อ `check` นี่คือ status check ที่ branch protection ของ `main` ต้อง require
- รัน `pnpm install --frozen-lockfile`, `pnpm check` และ `pnpm build` ด้วย Node 24, pnpm จาก `packageManager` (10.32.1) และ Go toolchain ใน `apps/api/go.mod` โดยไม่มี secret
- ไม่ได้รัน `db:*`, deploy จริง, preview ที่ต้องใช้ Docker หรือ Wrangler login, และการเชื่อม Neon/R2
- Dependabot รายสัปดาห์สำหรับ github-actions, npm และ gomod รวม version update ของแต่ละ ecosystem เป็น PR เดียว ยังไม่ได้เปิด PR จาก Dependabot
- ผลบน GitHub Actions: job `check` ผ่านเมื่อ 31 October 2026 บน [run 41229008153](https://github.com/HectorRussia/infu-foundry/actions/runs/41229008153) (commit `289107c`) ชื่อ status check คือ `check`

## Trash: restore or permanently delete contents

- Scope chosen by the owner: contents only (deleting a single video stays immediate and permanent), no automatic purge, older soft-deleted contents appear in the trash
- No schema change. API: GET `/trash`, POST `/trash/{id}/restore`, DELETE `/trash/{id}`, DELETE `/trash` (empty); purge deletes the cover and every video object in R2 first, then deletes media and content rows in one transaction, only for rows already in the trash
- Also fixed: the local dev CORS did not allow PATCH, so saving a cover position from localhost:3000 would fail (production is same-origin and was unaffected)
- Go tests: session required, list hides keys, restore needs JSON and 404s when not in the trash, live or invalid ids are never purged, an R2 failure keeps the content in the trash (500), purge removes cover and videos, emptying purges everything, routes absent without R2
- Web: "ถังขยะ" page in the menu with restore / delete permanently / empty trash; content delete now says it moves to the trash; video delete warns it is permanent
- `pnpm check` and the web static build pass
- Live end-to-end after the owner's approval (temporary API :8099, Neon + `infu-media-prod`), 17/17 passed: trash lists 12 older deleted contents, a test content with cover and video goes to the trash first with `videoCount` 1, restore brings it back, a live content cannot be purged (404) and its objects survive, purge → 204 with cover and video 404 in R2, the 12 older items untouched, content 404, second purge 404. Only the test content was purged; `DELETE /trash` was not called
- Production: Render served `/trash` within ~15 s of the check; web redeployed (version `1d13b38d`). The same 17 checks through the Worker passed (only the new test content purged, the 12 older trash items untouched); `/trash/` serves the page and its chunk contains "ล้างถังขยะ"
- Not yet: the owners using the trash page in a browser

## Cover crop position and cover download

- Migration `add_content_cover_position`: `cover_position_x/y` smallint, default 50, CHECK 0–100. Applied on Neon after review (2026-10-31); existing covers stay centred
- API: PATCH `/contents/{id}/cover` `{x, y}` saves the crop (422 out of range or missing, 404 unknown content); setting a new image recentres it; content GET/PUT and the cover endpoints return `coverDownloadUrl` (attachment named after the title, unsafe characters → `-`, fallback `cover`)
- Web: editor "จัดตำแหน่ง" mode drags the image with mouse or finger (`touch-action: none` so phones do not scroll), arrow keys/Enter/Esc, "กึ่งกลาง" reset; library cards use the saved `object-position`; download button next to the cover
- Go tests: PATCH validation and save, list carries the position, new image resets to 50/50, download URL carries the title filename; `CoverFilename` handles Thai, unsafe characters and blank titles
- `pnpm check` and the web static build pass
- Live end-to-end (temporary API :8099, Neon + `infu-media-prod`, generated PNGs), 12/12 passed: library loads with positions (6 items), new content centred, PATCH saves 50/20, out of range → 422, library card carries the position without a download URL, download URL → 200 `attachment` with the UTF-8 title filename, view URL stays inline, a new image recentres, remove cover, test content soft-deleted
- Production: Render served PATCH ~60 s after push; web redeployed (version `4c3315e9`). The same 12 checks through the Worker passed (R2 CORS for the site, download `attachment` with the Thai title, test content soft-deleted); deployed editor bundle contains "จัดตำแหน่ง"
- Not yet: dragging in a real browser and on a phone

## Cover images; library shows covers instead of videos

- Go tests: unsupported type / >10 MB → 422, unknown content 404, PUT before upload 409, a video key or another content's key → 422, replacing deletes the old image, removing clears key and object, list/GET carry `coverUrl` but never the key, routes absent without R2; key helpers reject traversal and foreign ids
- The local API was restarted with this code before the migration ran, so the library failed (`cover_key` missing); data was unaffected and production untouched. Gotcha added to CLAUDE.md
- `add_content_cover` applied on Neon after review
- Live end-to-end (temporary API :8099, Neon + `infu-media-prod`, generated 64×36 PNGs), 13/13 passed: library loads (6 items), set cover, cover URL serves `image/png`, library card and editor carry `coverUrl` without the key, replace → old object 404, remove → object 404 and no `coverUrl`, test content soft-deleted
- Browser (localhost): library shows 6 cards with no video elements; editor shows the cover box above videos accepting JPG/PNG/WebP
- Production: Render served the cover routes right after push; web redeployed (version `a28cdd80`). Through the Worker: cover upload URL, R2 CORS allows the site, PUT 200, set cover, library card image 200 `image/png`, remove → object 404, test content soft-deleted; deployed editor bundle contains the cover box
- Not yet: picking a real photo in the browser

## Video download on phones (Safari and Chrome)

- Cause: the phone players have no download item (iOS Safari never does; Chrome only on desktop), and `<a download>` is ignored for cross-origin R2 links
- Fix: API returns `downloadUrl`, a presigned R2 GET with `response-content-disposition: attachment`; download buttons in the editor video list and the gallery
- Go tests: download URL carries the filename; `AttachmentDisposition` handles ASCII, Thai (UTF-8 `filename*`) and quotes/backslashes
- Live check against `infu-media-prod` (temporary API :8099, existing videos, read-only): download URL → 206 with `attachment; filename=...`; play URL stays inline
- Production: Render served `downloadUrl` ~100 s after push; web redeployed (version `22bc8cd3`); both production videos' download URLs → 206 `attachment` with their filenames; the deployed gallery bundle contains the download button
- Verified on a real phone by the owner (2026-10-30): download works; iOS saves to Files (Downloads) — use Share → Save Video to put it in Photos

## Fix: videos uploaded locally did not play on production

- Production gallery listed `catknight_ep2.mp4` and `prod_2.mp4`, but both view URLs returned 404 `NoSuchKey` from `infu-media-prod`: they were uploaded from local into `infu-media-dev` while both environments share one Neon database
- Copied both objects dev → prod with wrangler (same keys; downloaded sizes 121,611,532 and 55,600,971 bytes match the database)
- Local `apps/api/.env` now uses `infu-media-prod` with the `infu-prod` token (values copied from `.env.render`, never printed); prod bucket CORS now also allows `http://localhost:3000`
- Production check after the fix: both videos → 206, full length matches, CORS header present, MP4 header present
- An attempt to read production keys for a per-bucket check was blocked by the permission system and not retried
- Local upload with the new bucket not yet exercised (needs `pnpm dev` restarted)

## Production: video uploads live

- Render had no `R2_*` variables at first (`GET /media` 404 while the new build was live); after the owner added them with "Save and deploy", `GET /media` → 200
- Web + Worker redeployed with `pnpm deploy:cloudflare` (version `de2bba7a`)
- Production end-to-end through the Worker with the 3.86 MB Gemini MP4: login, create test content, upload URL, prod bucket CORS allows the site and rejects localhost (403), PUT 200 in ~2.4 s, complete → ready, gallery shows the title, range playback 206 with CORS, delete → R2 object 404, test content soft-deleted
- One check failed in the test itself: it looked for the bucket in the URL path, but the SDK puts it in the hostname; the CORS result shows the upload went to `infu-media-prod`
- Browser upload on production by the owners not yet observed

## Library card grid and menu order

- Menu order now รายการทั้งหมด → คลังเนื้อหา → เขียนเนื้อหา (checked in the browser)
- Library at 1440px: 3 columns of 344px cards in a 1152px container; the video cover loaded (1080×1920 first frame); text covers for contents without video
- Library at 375px: one 343px column, no horizontal scroll
- Web lint/tsc pass; visual screenshot not captured (app window hidden) — layout verified by measurements

## Video gallery ("รายการทั้งหมด")

- Go test: `GET /media` needs login, returns the ready video with title and view URL, never exposes the object key
- First draft scanned into a struct embedding `Media` (the GORM pitfall in CLAUDE.md); switched to flat rows before testing
- Read-only query on Neon returned the real video with all fields; temporary API :8099: 401 without login, Ploy sees Horn's "Ep2 แมวน้อยผจญภัย", view URL range → 206
- Browser (localhost): new nav item and header label work; the page needs the local API restarted to show data
- Web lint/tsc/build pass (`/videos` exported); production not yet deployed

## Desktop editor layout

- Editor at ≥1024px: two columns (writing area 718px, 320px panel with status, save/copy, character count, author, last edited, delete); videos moved under the panel at the owner's request
- Measured in the browser on localhost (Horn account): 1440px → panel and videos in the right column, mobile bar hidden; 375px → panel hidden, videos below the content, bottom action bar shown, no horizontal scroll
- Web lint/tsc pass; production not yet deployed with this layout

## Video uploads with Cloudflare R2

- R2 enabled by the account owner; buckets `infu-media-dev` / `infu-media-prod` (APAC) with per-origin CORS; tokens `infu-dev` / `infu-prod` (Object Read & Write, one bucket each)
- `create_media` applied on Neon after review
- Go tests: auth required, unknown content 404, unsupported type / >1 GB / blank name 422, object key never exposed, complete → 409 before upload and on size mismatch, failed R2 delete keeps the row, delete removes the object, routes absent without R2 config
- Live end-to-end (temporary API :8099, Neon + `infu-media-dev`, the 3.86 MB Gemini MP4), 15/15 passed: R2 rejected a PUT with the wrong Content-Type (403) and the wrong size (403); real PUT 200 in ~0.5 s; complete → ready; list returns a view URL; Range request → 206 with `ftyp`; full download byte-identical; delete → object 404 and list empty; test content soft-deleted
- Web lint/tsc pass; the upload UI in the browser and production (Render R2 env, web deploy) are not verified yet

## Production: Cloudflare + Render

- Render Blueprint deployed `infu-foundry-api` (free, Singapore); `API_ORIGIN` set as a Worker var, `GATEWAY_SECRET` as a Worker secret and a Render env var
- Render direct: health 200; `/auth/me` without the secret → 404, with the secret → 401 (database connected, auth active)
- Through the Worker: health 200, contents without login 401, wrong password 401, Horn login 200 with `HttpOnly; Secure; SameSite=Lax; Path=/api` cookie, `/auth/me` and contents 200, logout 204 then 401; a client-supplied `X-Gateway-Secret` is overwritten by the Worker
- Production web build redirects to `/login/` when `/auth/me` returns 401 and no longer shows preview notices
- Logging in through the browser form on production is left to the account owners

## Real login (email/password + sessions)

- `create_sessions` applied on Neon after review; Horn and Ploy accounts created with `pnpm db:create-user` (bcrypt, password via stdin)
- Go tests: session required for contents, cookie flags (HttpOnly/SameSite=Lax/Path=/api, Secure outside development), per-email login lockout, JSON-only writes, gateway secret, credentialed CORS for localhost:3000
- Live end-to-end on Neon (temporary API :8099), 13/13 passed: 401 without login, wrong password and unknown email → 401, case-insensitive email login, HttpOnly cookie, `/auth/me`, content authored by the session user, the other account sees it, logout invalidates only that session; e2e content soft-deleted
- The first e2e run found two bugs fixed before commit: GORM expanded `[]byte` session hashes as a list, and an embedded struct was not scanned
- Browser login UI not yet exercised against the new API; production waits for Render

## Cloudflare Workers Free deploy

- Gateway tests 5/5 (incl. external-origin forwarding and gateway-secret stripping); dry-run builds passed for both `wrangler.jsonc` (ASSETS only) and `wrangler.containers.jsonc`
- Deployed with `pnpm deploy:cloudflare` to https://infu-foundry.ponkrit-wo.workers.dev (Workers Free, ASSETS binding only)
- Live: `/`, `/login/`, `/contents/`, `/contents/editor/` → 200; `/missing/` → 404; `/api/v1/*` → 503 `no-store` + `Retry-After: 5` because `API_ORIGIN` is not set yet
- Production contents page showed the load-failure message with no console errors
- Render, `API_ORIGIN`/`GATEWAY_SECRET` and Go-side secret checking remain to be done

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

## Neon + content library

- Neon project (AWS Singapore) connected locally: API startup logged `database connected` via the pooled URL
- `pnpm db:up` applied `create_users_and_contents` through the direct URL; column check matched the reviewed schema
- `apps/api/seeds/dev_user.sql` inserted one dev author after review
- `pnpm check` (Go) passed, including fake-store content route tests and a check that production does not expose content routes
- Web lint, typecheck and `next build` static export passed
- Live end-to-end on Neon (temporary API port 8099): CORS preflight, create, get, update, list ordering, empty-input 422, soft delete, 404 after delete — all passed; the e2e row remains soft-deleted
- Editor save from the browser confirmed by the user after restarting the API
- Production authentication and Containers with Neon remain unverified

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
