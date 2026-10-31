# Deployment plan — Infu Foundry

สถานะ: **ใช้งานจริงแล้ว** (2026-10-25) — เว็บ https://infu-foundry.ponkrit-wo.workers.dev,
API https://infu-foundry-api.onrender.com (เรียกผ่าน Worker เท่านั้น)
เป้าหมายช่วง MVP คือค่าใช้จ่าย $0 แล้วย้าย API กลับมา Cloudflare Containers เมื่อพร้อมจ่าย

## สรุปการตัดสินใจ

| ส่วน                     | ที่อยู่ช่วง MVP (ฟรี)                                        | อนาคต (เมื่อพร้อมจ่าย)                 |
| ------------------------ | ------------------------------------------------------------ | -------------------------------------- |
| หน้าเว็บ (static export) | Cloudflare Workers Static Assets                             | เหมือนเดิม                             |
| Gateway `/api/*`         | Cloudflare Worker (Free)                                     | เหมือนเดิม                             |
| Go API                   | **Render Free**                                              | **Cloudflare Containers** (setup เดิม) |
| PostgreSQL               | Neon Free (AWS Singapore)                                    | Neon (อัปเกรดเมื่อจำเป็น)              |
| ไฟล์ / วิดีโอ            | Cloudflare R2 — ดู [video-upload-plan](video-upload-plan.md) | เหมือนเดิม                             |

```text
ช่วง MVP
Browser → Cloudflare Worker ─┬─ Static Assets (Next.js export)
                             └─ /api/* → Go บน Render → Neon (pooled URL)

อนาคต
Browser → Cloudflare Worker ─┬─ Static Assets
                             └─ /api/* → Go Container → Neon
```

## ทำไมเลือกแบบนี้

- Cloudflare Containers ต้องใช้ Workers Paid ($5/เดือน ต่อ account ใช้ร่วมได้หลายโปรเจกต์)
  เป็นส่วนเดียวในระบบที่ไม่มี free tier
- Render มี free tier ถาวรสำหรับ web service; Railway เหลือแค่ trial credit แล้วต้องจ่าย $5/เดือน
  ถ้าจะจ่าย $5 อยู่แล้ว Cloudflare Containers คุ้มกว่าเพราะใช้ setup เดิมและอยู่ที่เดียว
- Worker เป็นตัวส่งต่อ `/api/*` ไป Render ทำให้ browser ยังเรียก same-origin `/api/v1`
  **หน้าเว็บไม่ต้องแก้ และไม่ต้องเปิด CORS ใน production**

ตัวเลขราคา/โควตาในเอกสารนี้มาจากความรู้ ณ วันที่ตัดสินใจ ยังไม่ได้ตรวจกับหน้า pricing ล่าสุด
ต้องตรวจอีกครั้งก่อน deploy จริง

## CI ก่อนรวมเข้า main

Workflow `CI` รัน job `check` บน pull request และบน push ขึ้น `main` รายละเอียดอยู่ใน [Architecture](architecture.md)
ก่อนเปิด branch protection ให้ require status check ชื่อ **`check`**
(Settings → Branches หรือ Rules → Require status checks → ค้น `check`)

CI ไม่ deploy Worker, ไม่สั่ง Render และไม่รัน migration
Render ยัง deploy อัตโนมัติเมื่อมี commit บน `main` ที่แตะ `apps/api/**` หลัง merge แล้วเท่านั้น
การเปิด required check ไม่ได้แทนการอนุมัติ migration บน Neon

## เงื่อนไขก่อน deploy

1. ~~มี authentication จริง~~ — ทำแล้ว: login + session cookie, content routes ต้อง login
2. Neon มี migration ล่าสุด (`pnpm db:up` จากเครื่องด้วย direct URL) และบัญชีผู้ใช้จาก `pnpm db:create-user`
   ถ้าโค้ดที่จะเข้า `main` อ่านตารางหรือคอลัมน์ของ migration นั้น ต้อง `db:up` ก่อน merge
   Render deploy API ทันทีที่ `apps/api/**` เข้า `main` และไม่รัน migration ให้
   ลำดับสำหรับ agent อยู่ใน [.cursor/rules/database-migrations.mdc](../.cursor/rules/database-migrations.mdc)

## ขั้นตอนช่วง MVP (Render + Cloudflare)

### 1. Go API บน Render

- ใช้ Blueprint `render.yaml` ที่ root: Docker จาก `apps/api/Dockerfile`, plan free, region Singapore,
  health check `/api/v1/health`, deploy ใหม่เฉพาะเมื่อ `apps/api/**` เปลี่ยน
- Render dashboard > New > Blueprint > เลือก repo แล้วกรอก env ที่ `sync: false`:
  - `DATABASE_URL` = Neon **pooled** URL (ห้ามใช้ direct URL สำหรับ runtime)
  - `GATEWAY_SECRET` = ค่าเดียวกับ Worker secret (เก็บไว้ใน `.env.render` ที่ gitignored)
  - `R2_ACCOUNT_ID`, `R2_BUCKET=infu-media-prod`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY` จาก token `infu-prod`
    (ค่า key อยู่ใน `.env.render`); ไม่ตั้ง = Go ปิดการอัปโหลดวิดีโอแต่ส่วนอื่นทำงานปกติ
- `APP_ENV=production` อยู่ใน blueprint; `PORT` Render กำหนดให้เองและ Go อ่านอยู่แล้ว
- Migration ไม่รันบน Render; รันจากเครื่องด้วย `MIGRATION_DATABASE_URL` เท่านั้น

### 2. Worker gateway ส่งต่อไป Render (ทำแล้ว)

- `apps/gateway/wrangler.jsonc` เป็น config ของ Workers Free: มีแค่ static assets ไม่มี Container
- ตั้ง var `API_ORIGIN` (เช่น `https://infu-foundry-api.onrender.com`) แล้ว Worker ส่ง `/api/*` ไปที่นั้น
  - ยังไม่ตั้ง → `/api/*` ตอบ 503 + `Retry-After` (หน้าเว็บยังใช้ได้)
- Config ของ Containers แยกไว้ที่ `apps/gateway/wrangler.containers.jsonc` (ใช้ตอนจ่าย $5)
- Render ที่หลับอยู่อาจใช้เวลาตื่น 30–60 วินาที request แรกจะช้าแต่ไม่ถูกตัด

### 3. กันการเรียก Render ตรง (ทำแล้ว)

- URL ของ Render เป็น public: Worker แนบ `X-Gateway-Secret` ทุก request และลบค่าที่ client ส่งมาเอง
- Go ตรวจ header นี้เมื่อตั้ง `GATEWAY_SECRET` และตอบ 404 ถ้าไม่ตรง (ยกเว้น health check ของ Render)
- Worker secret ตั้งแล้วด้วย `wrangler secret put GATEWAY_SECRET`

### 4. Cloudflare

- ครั้งแรก: `pnpm --filter @infu-foundry/gateway exec wrangler login`
- Deploy: `pnpm deploy:cloudflare` (build เว็บแล้ว `wrangler deploy` ด้วย config ฟรี)
- Production build ต้อง **ไม่ตั้ง** `NEXT_PUBLIC_API_BASE_URL` เพื่อเรียก same-origin `/api/v1`
- ตั้ง secrets ผ่าน `wrangler secret put` เท่านั้น ห้ามใส่ใน `wrangler.jsonc` หรือ repo

## ข้อจำกัดที่ยอมรับในช่วง MVP

- Render Free หลับเมื่อไม่มีคนใช้ (~15 นาที) — request แรกหลังหลับช้า 30–60 วินาที
- Neon Free ก็พักเมื่อไม่ใช้งาน — request แรกอาจช้าเพิ่มอีกเล็กน้อย
- ผู้ใช้มี 2 คน จึงยอมรับ cold start ได้; ถ้าเริ่มรำคาญคือสัญญาณให้ย้ายไป Containers

## ย้าย API กลับ Cloudflare Containers (เมื่อพร้อมจ่าย)

1. อัปเกรด account เป็น Workers Paid
2. `wrangler secret put DATABASE_URL --config wrangler.containers.jsonc` (Neon pooled URL)
3. ลบ `API_ORIGIN` แล้ว deploy ด้วย `wrangler deploy --config wrangler.containers.jsonc`
   — Worker เลือก Container อัตโนมัติเมื่อมี binding `BACKEND` และไม่มี `API_ORIGIN`
4. ตรวจ `/api/v1/health` และการบันทึกเนื้อหาผ่านโดเมนจริง
5. ปิด/ลบ service บน Render เมื่อยืนยันแล้ว

หน้าเว็บ, Go code และ Neon ไม่ต้องเปลี่ยนในขั้นนี้
