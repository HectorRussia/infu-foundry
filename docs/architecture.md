# Architecture — Infu Foundry

## ขอบเขตและเจ้าของความรับผิดชอบ

```text
Browser
  → Cloudflare Worker
      → Workers Static Assets: Next.js export
      → /api/*: Go Container
                   → Neon PostgreSQL
```

ช่วง MVP `/api/*` จะส่งไป Go บน Render แทน Container ดู [Deployment plan](deployment.md)

ใช้ pnpm workspace สำหรับ web/gateway และ Go module แยกสำหรับ backend
Next.js เป็น UI อย่างเดียว ไม่มี Next.js route handlers, Server Actions หรือ runtime SSR
Go เป็นเจ้าของ business logic, authentication และ database access ในอนาคต
Gateway ทำ routing และ Container lifecycle เท่านั้น

## Frontend

- App Router + TypeScript strict + Tailwind CSS utilities; global CSS กำหนด theme tokens และ base styles
- ธีมมืดและ mobile-first: หน้า Login แยกจาก workspace shell โดยไม่มี sidebar; ภายในมี sidebar บน desktop และเมนูพับได้บนมือถือ มีเพียงคลังเนื้อหาและ Editor หัวข้อ/Content ไม่มี dashboard หรือ bottom navigation
- Editor บน desktop (≥1024px) เป็นสองคอลัมน์: ซ้ายหัวข้อ+เนื้อหา, ขวาเป็นแผง (เลื่อนตามหน้า ไม่มี scrollbar แยก) (สถานะ, บันทึก, คัดลอก, ตัวอักษร/ผู้เขียน/แก้ไขล่าสุด, ลบ) และวิดีโอใต้แผง; มือถือคงคอลัมน์เดียว วิดีโอใต้เนื้อหา และแถบปุ่มติดล่างจอ
- `/videos/` (เมนู "รายการทั้งหมด") แสดงวิดีโอ ready ของเนื้อหาที่ยังไม่ถูกลบ ใหม่สุดก่อน (สูงสุด 200) ผ่าน `GET /api/v1/media`; 1/2/3 คอลัมน์ตามความกว้างจอ
- เมนูเรียง: รายการทั้งหมด → คลังเนื้อหา → เขียนเนื้อหา; หน้า workspace กว้าง `max-w-6xl` บน desktop
- คลังเนื้อหาเป็น card grid แบบเดียวกับแกลเลอรี: ปกเป็น**ภาพปก**ที่ผู้ใช้อัป (`coverUrl`) หรือปกข้อความถ้าไม่มี; ไม่แสดงวิดีโอในคลัง (ดูในหน้าเนื้อหา) ทั้งการ์ดเป็นลิงก์ไป editor ชื่อหมวดอยู่บนการ์ดถ้ามี
- กรองคลังด้วยชิปหมวดหมู่: ทั้งหมด, ไม่มีหมวดหมู่, และแต่ละหมวด (`?category=` เป็น `uncategorized` หรือ UUID) สร้าง เปลี่ยนชื่อ และลบหมวดได้จากหน้าเดียวกัน ลบหมวดไม่ลบเนื้อหา
- Editor เลือกหมวดได้หนึ่งหมวด (หรือไม่มี) และสร้างหมวดใหม่ได้ การจัดเข้าหมวดบันทึกพร้อมเนื้อหา
- Editor มีกล่องภาพปกเหนือส่วนวิดีโอ (คอลัมน์ขวาบน desktop, ใต้เนื้อหาบนมือถือ) พร้อมปุ่ม "จัดตำแหน่ง" (ลากด้วยเมาส์/นิ้ว
  หรือปุ่มลูกศร; กรอบ 16:9 เท่ากับการ์ดในคลัง จึงเห็นผลตรงกัน), ปุ่มดาวน์โหลด และปุ่มลบ
- `/login/` เป็น alias ของหน้า Login ที่ `/`; ส่ง email/password ไป Go แล้ว Go ตั้ง session cookie HttpOnly
- UI อ่านสถานะจาก `GET /api/v1/auth/me` เพื่อแสดงผลเท่านั้น การป้องกันจริงอยู่ที่ Go ทุก content route
- `src/app`: routes/layout; `components`: shared UI; `features`: auth/content UI; `lib`: API configuration
- Static export และ trailing slash ทำให้ทุก route มี HTML สำหรับเปิดตรงและ refresh
- หลัง build มีขั้นตอนเติม filename aliases สำหรับ RSC segment files ที่ Next.js 16 export บน Windows ผิดรูปแบบ เป็น no-op เมื่อ output ถูกต้อง และไม่แก้ไฟล์ใน node_modules; ดู [upstream issue](https://github.com/vercel/next.js/issues/85374)
- Editor เป็น `/contents/editor/` คงที่ รอบถัดไปใช้ query parameter ระบุ ID ของรายการ ไม่ใช้ dynamic route ที่ต้องรู้ ID ตอน build
- Next.js render HTML ล่วงหน้าได้ แต่ข้อมูลส่วนตัวในอนาคตต้องโหลดหลังตรวจ session ผ่าน Go API

## Backend

- `cmd/server`: composition, optional database startup และ Fiber lifecycle ผ่าน GracefulContext
- `cmd/migrate`: Goose CLI สำหรับ create/status/up/down แยกจาก API
- `internal/config`: validation ของ process configuration
- `internal/httpapi`: Fiber v3 routing, CORS, panic recovery และ slog request logging
- `internal/database`: GORM + PostgreSQL driver, connection pool และ bounded startup ping ไม่มี AutoMigrate
- `migrations`: Goose SQL files; ตอนนี้มีเฉพาะคู่มือ ยังไม่สร้าง application tables
- รอบฟีเจอร์จริงเพิ่ม `internal/auth` และ `internal/contents` โดย inject GORM connection เข้า repositories
- Dependencies pin ใน go.mod/go.sum; toolchain Go 1.26.10 และ Docker builder ใช้เวอร์ชันเดียวกัน
- `GET /api/v1/health` เป็น liveness check ไม่ตรวจ database และไม่เปิดเผย secrets
- Logging บันทึก method/path/status/duration ไม่บันทึก request body, cookies หรือ query parameters

## Local development กับ production

Local development เรียกเว็บ port 3000 และ Go port 8080 จึงอนุญาต CORS เฉพาะ `http://localhost:3000`
Production build ใช้ relative URL `/api/v1` ผ่าน Worker จึงเป็น same origin และไม่เปิด CORS ให้ localhost
ไม่มี credentials หรือ database ที่จำเป็นต่อ health check

เมื่อไม่ตั้ง DATABASE_URL จะรัน foundation API โดยไม่สร้าง connection หากตั้งแล้วต้อง ping ผ่านก่อนรับ request
runtime รองรับ Neon pooled URL; Goose ใช้ MIGRATION_DATABASE_URL แยกและต้องเป็น direct URL
Cloudflare Worker ส่ง DATABASE_URL ที่ตั้งเป็น secret เข้า Container environment เท่านั้น ไม่ส่งเข้า frontend หรือ asset bundle
Goose ใช้ session-level advisory lock เพื่อกัน migration พร้อมกัน และไม่ถูกเรียกจาก API startup
เมื่อไม่มี SQL files คำสั่ง migration จะไม่เชื่อมฐานข้อมูล; เมื่อมีแล้ว Goose อาจสร้างตารางเก็บ version แม้เป็นคำสั่ง status
Schema แรกมี `users` และ `contents` (UUID, soft delete ด้วย `deleted_at`) ผู้ใช้ทุกคนเห็นเนื้อหาร่วมกัน;
`author_id` บันทึกผู้เขียนเท่านั้น หากต้องแยกตามทีมในอนาคตให้เพิ่ม workspaces + `contents.workspace_id`
Authentication: bcrypt (cost 12) + server-side sessions; ตาราง `sessions` เก็บเฉพาะ SHA-256 ของ token
Cookie `infu_session` Path `/api` HttpOnly SameSite=Lax (Secure นอก development) อายุ 30 วัน ไม่มีการสมัคร
Login ผิด 10 ครั้งใน 15 นาทีต่ออีเมลถูกบล็อกชั่วคราว (in-memory ต่อ instance)
คำขอเขียนต้องเป็น JSON เพื่อกัน form ข้ามเว็บ; development อนุญาต CORS พร้อม credentials เฉพาะ localhost:3000
เมื่อตั้ง `GATEWAY_SECRET` Go รับเฉพาะคำขอที่มี `X-Gateway-Secret` จาก Worker (ยกเว้น health)

วิดีโอ: ไฟล์อยู่ใน Cloudflare R2, ตาราง `media` เก็บ `object_key` (`media/<uuid>.<ext>`) ไม่เก็บ URL
Browser อัปโหลดตรงเข้า R2 ด้วย presigned PUT (15 นาที, ลงนาม Content-Type และ Content-Length) แล้วเรียก
`/media/{id}/complete` ให้ Go HEAD ตรวจขนาดก่อนตั้ง `ready`; ดูด้วย presigned GET อายุ 1 ชั่วโมง (รองรับ Range)
ไฟล์ไม่ผ่าน Go/Render; ลบวิดีโอ = ลบ object ก่อนแล้ว soft delete แถว; ลบเนื้อหา = soft delete (เข้าถังขยะ) ไม่ลบวิดีโอ/ภาพปก
ถังขยะ (`/trash`): เฉพาะเนื้อหา (ตัดสินใจ 2026-10-31) — กู้คืน = `deleted_at = NULL`; ลบถาวร = ลบ object ใน R2 ทุกตัว (ภาพปก + วิดีโอทุกสถานะ) ก่อน
ถ้าลบไม่ได้แถวยังอยู่ในถังให้ลองใหม่ จากนั้น transaction `DELETE media` + `DELETE contents` (เฉพาะแถวที่ `deleted_at IS NOT NULL` เท่านั้น เนื้อหาที่ยังใช้อยู่ลบถาวรไม่ได้)
ไม่มีการล้างอัตโนมัติ (Render ฟรีไม่มี cron และผู้ใช้เลือกล้างเอง); route ถังขยะเปิดเฉพาะเมื่อมี R2
ภาพปก: `contents.cover_key` (nullable, CHECK `covers/%`), object `covers/<content id>/<random>.<ext>` ชื่อใหม่ทุกครั้งกัน cache;
flow POST (presigned PUT, JPG/PNG/WebP ≤ 10 MB) → PUT key (Go ตรวจว่า key เป็นของเนื้อหานั้นและ object เป็นภาพจริง) → ลบภาพเก่า; DELETE ลบทั้ง key และ object
ตำแหน่งครอป: `cover_position_x/y` (smallint 0–100, default 50, ใช้เป็น CSS `object-position`) ตั้งด้วย PATCH; ภาพใหม่เริ่มที่กึ่งกลางเสมอ
หมวดหมู่ใช้ร่วมกันทุกบัญชี ไม่มีหมวดย่อย หนึ่งเนื้อหาอยู่ได้มากสุดหนึ่งหมวด (ไม่บังคับ) เก็บใน `categories` + `content_categories` โดย `content_id` เป็น primary key ของตารางเชื่อม ไม่ใส่คอลัมน์บน `contents` เพื่อให้ภายหลังเปลี่ยนเป็น many-to-many (แท็ก) ได้โดยไม่ย้ายข้อมูลใน `contents` เนื้อหาที่ไม่มีแถวเชื่อมคือไม่มีหมวดหมู่ ลบหมวดแล้ว cascade เฉพาะแถวเชื่อม เนื้อหาคงอยู่และกลับเป็นไม่มีหมวด (soft delete ของเนื้อหายังเก็บลิงก์ไว้ จึงกู้จากถังขยะแล้วยังอยู่หมวดเดิม; ลบถาวรลบลิงก์ตามเนื้อหา) รายการคลังและ GET เนื้อหา LEFT JOIN จึงยังคืนเนื้อหาเดิมที่ไม่มีหมวด ต้องรัน migration นี้ก่อน deploy API ที่ join ตารางใหม่ ไม่งั้นคลังและ editor โหลดไม่ได้
ภาพปกมี `coverDownloadUrl` (attachment ตั้งชื่อตามหัวข้อ + นามสกุลเดิม) ส่งเฉพาะ GET/PUT เนื้อหาและ endpoint ภาพปก ไม่ส่งในรายการคลัง
ดาวน์โหลด: presigned GET แยกที่ใส่ `response-content-disposition: attachment` (ชื่อไฟล์เดิม แบบ RFC 6266)
เพราะ iOS Safari / Android Chrome ไม่สนใจ `<a download>` ข้าม origin; header จาก R2 คือสิ่งที่ทำให้มือถือบันทึกไฟล์
S3 SDK ตั้ง checksum เป็น "when required" เพราะ browser ส่ง CRC header ของ SDK ใหม่ไม่ได้
ใช้ bucket `infu-media-prod` ทั้ง local และ production เพราะทั้งสองใช้ Neon ก้อนเดียว (token `infu-prod`, Object Read & Write เฉพาะ bucket)
CORS ของ bucket อนุญาต production และ `http://localhost:3000`; `infu-media-dev` เลิกใช้แล้ว (เก็บไว้จนกว่าจะแยก DB สำหรับ dev)

Container ใช้ fixed instance name เดียว จำกัดหนึ่ง instance และพักเมื่อไม่ใช้งาน 10 นาที
Container helper รอ port พร้อม จากนั้นตรวจ health ก่อนส่ง request เดิม
Worker คืน 503 แบบไม่ cache ถ้า start/health/forward ล้มเหลว ไม่มีการ retry คำขอเขียนข้อมูลอัตโนมัติ
Static assets ใช้ 404-page handling ไม่ fallback ทุก route ไปหน้าแรก
ข้อมูลและ session ในอนาคตต้องเก็บภายนอก Container เพราะ lifecycle ไม่ใช่ durable storage ของแอป

## CI

Pull request (ทุก base branch) และ push ขึ้น `main` รัน workflow `CI` ใน `.github/workflows/ci.yml`
มี job เดียวชื่อ `check` — นี่คือชื่อ status check ที่ branch protection ของ `main` ต้อง require
(ในหน้าตั้งค่าค้น `check` ชื่อนี้มาจาก job ไม่ใช่ชื่อไฟล์ workflow)

Runner ติดตั้ง Node.js 24 (ตาม `engines`), pnpm 10.32.1 จากฟิลด์ `packageManager` ผ่าน `pnpm/action-setup`
และ Go ตามบรรทัด `toolchain` ใน `apps/api/go.mod` (`go1.26.10`) ผ่าน `actions/setup-go`
cache ของ pnpm store และ Go modules เปิดอยู่

ลำดับคำสั่งคือ `pnpm install --frozen-lockfile` แล้ว `pnpm check`
(ESLint, TypeScript, gateway tests, `go vet`, Go tests) แล้ว `pnpm build`
(static export, Go binaries, `wrangler deploy --dry-run`)
workflow ไม่มี secret และไม่รันคำสั่งที่ต้องใช้ Cloudflare, Neon หรือ R2
(`db:*`, `deploy:cloudflare`, preview) CI ไม่ deploy

Token ของ workflow อ่านโค้ดได้อย่างเดียว (`contents: read`) action ถูก pin ด้วย commit SHA
run ที่ถูกแทนที่บน PR หรือ branch เดียวกันถูกยกเลิก
Dependabot ตรวจ github-actions, npm และ Go module ทุกสัปดาห์
version update ของแต่ละ ecosystem ถูกรวมเป็น PR เดียว; security update ยังแยกตามแพ็กเกจ

## ข้อตกลง MVP รอบถัดไป

- ทีมเดียว คลังเนื้อหาร่วม ทุกบัญชีเห็นและแก้ไขรายการร่วมกัน
- บัญชีอีเมล/รหัสผ่านที่ทีมสร้างให้ล่วงหน้า ไม่มีสมัครสมาชิกสาธารณะ
- แต่ละรายการมีหัวข้อและ Content พร้อมข้อมูลผู้สร้าง/ผู้แก้ไขและเวลา (ตัด Prompt ออกจากขอบเขตแล้ว)
- Go ตรวจสิทธิ์ทุก endpoint ที่เกี่ยวข้องกับข้อมูล การซ่อนหน้าจอไม่ใช่ access control
- วางแนวทาง server-side sessions ผ่าน HttpOnly/Secure cookie และจัดการ CSRF เมื่อทำ authentication จริง
- Neon PostgreSQL เชื่อมจาก Go เท่านั้น ข้อมูลไม่ผ่าน browser database credentials
- ไม่มี AI generation, video storage, publish-to-social, blog หรือ public portfolio ในรอบนี้

รายละเอียด schema, password/session implementation และ CRUD contracts จะออกแบบในรอบฟีเจอร์ ไม่สร้าง placeholder API ที่ดูเหมือนใช้งานจริง
