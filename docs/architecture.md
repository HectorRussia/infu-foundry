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
- คลังเนื้อหาเป็น card grid แบบเดียวกับแกลเลอรี: ปกเป็นเฟรมแรกของวิดีโอล่าสุด (จาก `GET /media`) หรือปกข้อความถ้าไม่มีวิดีโอ/R2 ใช้ไม่ได้; ทั้งการ์ดเป็นลิงก์ไป editor
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
ไฟล์ไม่ผ่าน Go/Render; ลบวิดีโอ = ลบ object ก่อนแล้ว soft delete แถว; ลบเนื้อหาไม่ลบวิดีโอ
S3 SDK ตั้ง checksum เป็น "when required" เพราะ browser ส่ง CRC header ของ SDK ใหม่ไม่ได้
Bucket dev/prod แยกกัน แต่ละตัวมี token ของตัวเอง (Object Read & Write เฉพาะ bucket) และ CORS เฉพาะ origin ของตัวเอง

Container ใช้ fixed instance name เดียว จำกัดหนึ่ง instance และพักเมื่อไม่ใช้งาน 10 นาที
Container helper รอ port พร้อม จากนั้นตรวจ health ก่อนส่ง request เดิม
Worker คืน 503 แบบไม่ cache ถ้า start/health/forward ล้มเหลว ไม่มีการ retry คำขอเขียนข้อมูลอัตโนมัติ
Static assets ใช้ 404-page handling ไม่ fallback ทุก route ไปหน้าแรก
ข้อมูลและ session ในอนาคตต้องเก็บภายนอก Container เพราะ lifecycle ไม่ใช่ durable storage ของแอป

## ข้อตกลง MVP รอบถัดไป

- ทีมเดียว คลังเนื้อหาร่วม ทุกบัญชีเห็นและแก้ไขรายการร่วมกัน
- บัญชีอีเมล/รหัสผ่านที่ทีมสร้างให้ล่วงหน้า ไม่มีสมัครสมาชิกสาธารณะ
- แต่ละรายการมีหัวข้อและ Content พร้อมข้อมูลผู้สร้าง/ผู้แก้ไขและเวลา (ตัด Prompt ออกจากขอบเขตแล้ว)
- Go ตรวจสิทธิ์ทุก endpoint ที่เกี่ยวข้องกับข้อมูล การซ่อนหน้าจอไม่ใช่ access control
- วางแนวทาง server-side sessions ผ่าน HttpOnly/Secure cookie และจัดการ CSRF เมื่อทำ authentication จริง
- Neon PostgreSQL เชื่อมจาก Go เท่านั้น ข้อมูลไม่ผ่าน browser database credentials
- ไม่มี AI generation, video storage, publish-to-social, blog หรือ public portfolio ในรอบนี้

รายละเอียด schema, password/session implementation และ CRUD contracts จะออกแบบในรอบฟีเจอร์ ไม่สร้าง placeholder API ที่ดูเหมือนใช้งานจริง
