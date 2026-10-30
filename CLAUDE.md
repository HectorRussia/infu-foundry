# Infu Foundry — กติกาสำหรับ AI agents

คลังเนื้อหาส่วนตัวที่ใช้กัน 2 คน (Horn, Ploy) ทุกบัญชีเห็นเนื้อหาร่วมกัน ช่วงนี้เน้น MVP และค่าใช้จ่าย $0
รายละเอียดอยู่ใน docs อย่าเขียนซ้ำที่นี่: [README](README.md) ·
[Architecture](docs/architecture.md) · [Deployment](docs/deployment.md) · [Validation](docs/validation.md)

## กติกาการทำงาน (ต้องทำตามเสมอ)

1. **ฐานข้อมูล: ถามก่อนทุกครั้ง** — เขียนไฟล์ migration ได้ แต่ห้ามรัน `db:up`, `db:down`,
   `db:create-user` หรือ SQL ใด ๆ บน Neon จนกว่าผู้ใช้จะเห็น SQL และตอบอนุมัติ
2. **ทำงานบน `main`** — ห้ามแตก branch จนกว่าผู้ใช้จะสั่ง
3. **Commit แยกเป็นชุด** ตามส่วนงาน (db → api → web → deploy → docs) แบบ conventional commits
   เช่น `feat(api): ...`, `docs: ...` และ stage ทีละ path; commit/push เฉพาะเมื่อผู้ใช้สั่ง
4. **อัปเดต docs ทุกครั้ง** ในงานเดียวกัน: การตัดสินใจ/วิธีตั้งค่าใน README, architecture, deployment
   และเพิ่มหัวข้อใหม่ด้านบนของ `docs/validation.md` (ทดสอบอะไร ผลเป็นยังไง อะไรยังไม่ยืนยัน)
   endpoint ใหม่ต้องอัปเดต `contracts/openapi.yaml`
5. **ห้ามเริ่มงานต่อเอง** — เสนอได้ แต่รอผู้ใช้สั่ง
6. **ความลับไม่เข้า repo** — `apps/api/.env` (Neon URLs, R2 dev key) และ `.env.render` (gateway secret, R2 prod key) ถูก gitignore;
   ห้ามแสดงค่าใน chat/log และห้ามใส่รหัสผ่านผู้ใช้ในไฟล์ใด ๆ

## ระบบ

```text
Browser → Cloudflare Worker (apps/gateway, Workers Free)
            ├─ static Next.js export (apps/web)
            └─ /api/* + X-Gateway-Secret → Go บน Render Free (apps/api) → Neon Postgres (Singapore)
Browser ⇄ Cloudflare R2 (วิดีโอ) ผ่าน presigned URL ที่ Go ออกให้ — ใช้ `infu-media-prod` ทั้ง local และ production
```

- Production: https://infu-foundry.ponkrit-wo.workers.dev — API เรียกผ่าน Worker เท่านั้น
- Auth: email/password + session cookie `infu_session` (HttpOnly); ไม่มีหน้าสมัคร
- อนาคต: เมื่อจ่าย Workers Paid ย้าย Go ไป Cloudflare Containers (`wrangler.containers.jsonc`)

## คำสั่ง

| คำสั่ง                                      | ใช้ทำอะไร                                             |
| ------------------------------------------- | ----------------------------------------------------- |
| `pnpm dev`                                  | web :3000 + Go :8080 (อ่าน `apps/api/.env`)           |
| `pnpm check`                                | lint, tsc, gateway tests, go vet/test — รันก่อนส่งงาน |
| `pnpm --filter @infu-foundry/web build`     | ตรวจ static export                                    |
| `pnpm db:create <name>` / `db:status`       | สร้าง migration / ดูสถานะ (status อ่านอย่างเดียว)     |
| `pnpm db:up` / `db:down` / `db:create-user` | **แตะ Neon — ต้องได้รับอนุมัติก่อน**                  |
| `pnpm deploy:cloudflare`                    | build เว็บและ deploy Worker (production — ถามก่อน)    |

Render deploy อัตโนมัติเมื่อ push ที่แก้ `apps/api/**` ขึ้น `main`

## ข้อควรระวังที่เคยเจอ

- แก้ Go แล้วต้องรีสตาร์ท `pnpm dev:api` เอง (Next.js reload เอง แต่ Go ไม่) — อาการ: หน้าเว็บขึ้น "เชื่อมต่อไม่ได้"/404
- `pnpm deploy` เป็นคำสั่ง built-in ของ pnpm ต้องใช้ `pnpm run deploy`
- GORM `Raw`/`Exec` แตก `[]byte` เป็น list → ส่ง hex string แล้ว `decode(?, 'hex')`;
  และอย่า Scan เข้า struct ที่ embed struct อื่น (ได้ค่าว่าง)
- Migration ใช้ direct URL (`MIGRATION_DATABASE_URL`), runtime ใช้ pooled URL (`DATABASE_URL`, host มี `-pooler`)
- Test ด้วย fake store จับบั๊ก SQL ไม่ได้ — งาน DB ต้องทดสอบกับ Neon จริง (หลังได้รับอนุมัติ) โดยใช้ API ชั่วคราวพอร์ตอื่น เช่น 8099
- Web เป็น static export: ไม่มี dynamic route ใช้ `?id=` แทน
- AWS S3 SDK รุ่นใหม่ใส่ checksum ใน presigned PUT ที่ browser ส่งไม่ได้ → ต้องตั้ง checksum เป็น "when required" (ดู `internal/media/r2.go`)
- คำเตือน CRLF จาก git/prettier บน Windows ไม่ใช่การแก้เนื้อหา ไม่ต้องแก้
- โค้ดที่อ่านคอลัมน์ใหม่จะพังทันทีถ้ารีสตาร์ท API ก่อน `db:up` (เคยทำคลังเนื้อหาในเครื่องโหลดไม่ได้) — เตือนผู้ใช้ไม่ให้รีสตาร์ทจนกว่า migration จะรัน และห้าม push ขึ้น Render ก่อน
- **DB กับ storage ต้องแยกหรือรวมไปพร้อมกัน**: ตอนนี้ local กับ production ใช้ Neon ก้อนเดียว จึงต้องใช้ R2 bucket เดียวด้วย
  (เคยแยก bucket แล้ววิดีโอที่อัปจาก local เล่นบน production ไม่ได้) ถ้าจะแยก dev ให้แยก Neon branch และ bucket พร้อมกัน
- `git add` แล้ว commit ด้วย pathspec ทีละชุด และตรวจ `git show --stat` ก่อน push ว่าไฟล์ครบ
