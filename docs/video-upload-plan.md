# Video upload plan — Cloudflare R2

สถานะ: **ทำแล้วใน local + ทดสอบกับ R2 dev สำเร็จ** (2026-10-30) — production ยังไม่เปิด
(ต้องใส่ R2 env ของ prod ใน Render และ deploy เว็บ)

### สิ่งที่ทำแล้ว

- ขั้น 0–3 เสร็จ: R2 + buckets + CORS + tokens, migration `media` (มี CHECK ชนิด/ขนาด/สถานะ), Go endpoints,
  ปุ่มแนบวิดีโอพร้อม progress/ยกเลิก, player, ลบ และป้ายจำนวนวิดีโอในคลังเนื้อหา
- เหลือ: ตั้ง `R2_*` ของ prod ใน Render (token `infu-prod`, bucket `infu-media-prod`), push, deploy เว็บ, ทดสอบ production

### ความคืบหน้าขั้น 0

- R2 subscription เปิดโดยเจ้าของ account (free tier: 10 GB, 1M Class A, 10M Class B ต่อเดือน)
- Buckets (location hint APAC): `infu-media-dev`, `infu-media-prod`
- CORS: dev อนุญาตเฉพาะ `http://localhost:3000`, prod เฉพาะ `https://infu-foundry.ponkrit-wo.workers.dev`;
  methods GET/HEAD/PUT, headers `content-type`/`range`, expose `ETag`/`Content-Length`/`Content-Range`/`Accept-Ranges`
- Env ของ Go: `R2_ACCOUNT_ID`, `R2_BUCKET`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`
  (token แยก: dev bucket ใน `apps/api/.env`, prod bucket ใน Render)

## เป้าหมาย

แนบวิดีโอเข้ากับเนื้อหาในหน้า editor ได้ทั้งจากมือถือและคอม ดูได้เฉพาะผู้ที่ login
ค่าใช้จ่าย $0 ภายในโควตาฟรีของ R2

## การตัดสินใจ

| เรื่อง      | ตัดสินใจ                                                                |
| ----------- | ----------------------------------------------------------------------- |
| ที่เก็บไฟล์ | Cloudflare R2 — ไม่มีค่า egress, free tier ~10 GB, account เดียวกับเว็บ |
| ผูกกับอะไร  | แนบกับเนื้อหา (`contents`) ไม่ใช่คลังวิดีโอแยก                          |
| ขนาดสูงสุด  | 1 GB ต่อไฟล์ (single PUT, ไม่มี multipart/resume ในรอบแรก)              |
| ชนิดไฟล์    | `video/mp4`, `video/quicktime`, `video/webm`                            |
| การลบวิดีโอ | ลบ object ใน R2 ทันที + soft delete แถว (ประหยัดพื้นที่)                |
| ลบเนื้อหา   | soft delete เนื้อหาไม่ลบวิดีโอ กู้เนื้อหาแล้ววิดีโอยังอยู่              |
| Bucket      | `infu-media-dev` (local) และ `infu-media-prod` แยกข้อมูลทดสอบออกจากจริง |
| การดูวิดีโอ | Go ออก presigned GET URL อายุ 1 ชั่วโมงให้ผู้ที่ login เท่านั้น         |

## สิ่งที่เก็บใน Neon

DB เก็บ **path ของไฟล์ใน R2 (`object_key`) ไม่ใช่ไฟล์และไม่ใช่ URL**
URL สำหรับดูเป็นลิงก์มีลายเซ็นที่หมดอายุ จึงสร้างใหม่จาก key ทุกครั้ง; ย้าย bucket/โดเมนได้โดยไม่แก้ข้อมูล
ชื่อ object ใช้ UUID (`media/<uuid>.<ext>`) ส่วนชื่อไฟล์เดิมเก็บใน `filename` เพื่อแสดงผล

```sql
CREATE TABLE media (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  content_id   uuid NOT NULL REFERENCES contents(id),
  uploader_id  uuid NOT NULL REFERENCES users(id),
  object_key   text NOT NULL UNIQUE,
  filename     text NOT NULL,
  content_type text NOT NULL,
  size_bytes   bigint NOT NULL,
  status       text NOT NULL DEFAULT 'pending',  -- pending | ready
  created_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz
);
```

(ร่าง — SQL จริงจะผ่านการรีวิวก่อนรัน)

## Flow

```text
1. Browser → Worker → Go: ขออัปโหลด (ชื่อ, ชนิด, ขนาด) — Go ตรวจ login/ชนิด/ขนาด
2. Go: สร้างแถว media (pending) + presigned PUT URL อายุ 15 นาที
3. Browser → R2: PUT ไฟล์ตรง (แสดง progress) — ไฟล์ไม่ผ่าน Go/Render
4. Browser → Go: แจ้งอัปเสร็จ — Go HEAD object ตรวจขนาด/ชนิด แล้วตั้ง status = ready
5. ดูวิดีโอ: Go ออก presigned GET URL อายุ 1 ชั่วโมง → <video> เล่นจาก R2
```

ไฟล์ไม่ผ่าน Go เพราะ API รับ body สูงสุด 1 MB และ Render Free ไม่เหมาะกับ upload ใหญ่

## ขั้นตอน

0. **ตั้งค่า R2** — เปิด R2, สร้าง bucket dev/prod, CORS (PUT/GET จาก production และ localhost:3000),
   R2 API token จำกัดเฉพาะ 2 bucket; เก็บ key ใน `apps/api/.env` และ Render env เท่านั้น
1. **Migration `media`** — รีวิว SQL ก่อน `db:up`
2. **Go API** — S3 SDK (R2 รองรับ S3 API) และ endpoints:
   `POST /contents/:id/media`, `POST /media/:id/complete`, `GET /contents/:id/media` (แยก endpoint แทนการรวมใน `GET /contents/:id`),
   `DELETE /media/:id`; test ด้วย fake + e2e กับ bucket dev
3. **Web** — ปุ่ม "แนบวิดีโอ" ใน editor (`accept="video/*"`), progress + ยกเลิก, รายการวิดีโอพร้อม player
   และปุ่มลบ; ไอคอนในคลังเนื้อหาเมื่อมีวิดีโอ
4. **ทดสอบ/deploy/บันทึก** — local กับ dev bucket → push (Render auto deploy) → deploy เว็บ →
   ทดสอบ production → อัปเดต docs, `contracts/openapi.yaml`, `CLAUDE.md`

## ข้อจำกัดรอบแรก

- อัปทีละไฟล์แบบ single PUT: เน็ตหลุดต้องอัปใหม่
- ไม่ transcode: วิดีโอ iPhone แบบ HEVC อาจเล่นไม่ได้บน Chrome/Windows
  (ตั้ง iPhone เป็น Settings → Camera → Formats → Most Compatible) — ทางเลือกในอนาคตคือ Cloudflare Stream (เสียเงิน)
- ไฟล์ที่ขออัปแต่ไม่เสร็จค้างใน R2 — ตั้ง lifecycle rule ภายหลัง

## ไฟล์ทดสอบ

`gemini_generated_video_e2375a58.mp4` — 3.86 MB, MP4/H.264 (อ่านจาก header) เล่นได้ทุก browser
