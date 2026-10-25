# Infu Foundry

A creative workspace for organizing ideas and content — from first draft to published work.

## สถานะของรอบนี้

นี่คือ foundation scaffold: หน้าเว็บตัวอย่าง + Fiber health API + GORM connection layer + Goose migrations + Cloudflare gateway configuration
ยังไม่มี authentication, application schema, การบันทึกเนื้อหา หรือการเรียก AI ภายนอก และยังไม่ได้เชื่อม Neon จริง
ทุกหน้าตัวอย่างเปิดได้โดยไม่ต้อง Login จึงไม่ควรนำข้อมูลจริงหรือข้อมูลลับมาทดลอง

## เริ่มพัฒนาในเครื่อง

ต้องมี Node.js 24 ขึ้นไป, pnpm 10.32.1 และ Go 1.26 ขึ้นไป
โปรเจกต์ pin toolchain `go1.26.10` ใน go.mod; Go launcher รุ่นเก่าที่เปิด `GOTOOLCHAIN=auto`
จะดาวน์โหลด toolchain นี้เมื่อรันคำสั่งจาก `apps/api` หรือผ่าน pnpm scripts โดยไม่เปลี่ยน Go ที่ติดตั้งทั้งเครื่อง

```powershell
pnpm install --frozen-lockfile
pnpm dev
```

เปิด http://localhost:3000 — API อยู่ที่ http://localhost:8080/api/v1/health
คำสั่งนี้ไม่ต้องมี Docker, Cloudflare account หรือ Neon database
กด Ctrl+C เพื่อหยุดทั้งสอง process ถ้าแก้ Go ให้หยุดแล้วรันใหม่ (ยังไม่มี Go hot reload)
ฝั่ง Next.js มี hot reload ตามปกติ

| คำสั่ง                    | หน้าที่                                                                                |
| ------------------------- | -------------------------------------------------------------------------------------- |
| `pnpm dev`                | Build และรัน Go พร้อม Next.js dev server                                               |
| `pnpm dev:web`            | รัน frontend อย่างเดียว                                                                |
| `pnpm dev:api`            | Build และรัน Go อย่างเดียว                                                             |
| `pnpm check`              | ESLint, TypeScript, Go vet และ Go tests                                                |
| `pnpm build`              | Export หน้าเว็บ, build Go binary และ bundle Worker แบบ dry-run                         |
| `pnpm preview:cloudflare` | Build เว็บและรัน Workers + Container ในเครื่องที่ port 8787                            |
| `pnpm preview:assets`     | ตรวจ Worker/static assets บน Windows โดยไม่รัน Container; API จะแสดงว่าเชื่อมต่อไม่ได้ |

`pnpm build` ไม่ deploy และไม่ยืนยันว่า Docker image หรือ Containers รันได้จริง
ใช้คำสั่ง preview เพื่อตรวจการทำงานครบเส้นทาง
Web build มีขั้นตอนแก้ชื่อไฟล์ RSC ที่ Next.js static export บน Windows สร้างไม่ตรงกับ client navigation
โดยเพิ่ม aliases ใน output อัตโนมัติ รายละเอียดอยู่ใน Architecture

## หน้าตัวอย่าง

- `/` หน้าแรกเป็นการ์ด Login เดี่ยว ไม่มี sidebar หรือหน้าภาพรวม
- `/login/` ฟอร์มตัวอย่าง ปุ่มเข้าสู่ระบบปิดไว้ ไม่มีการส่ง credentials
- `/contents/` คลังเนื้อหาว่าง ไม่มีข้อมูลสมมติที่แสดงว่าเก็บแล้ว
- `/contents/editor/` มีเฉพาะหัวข้อและ Content พิมพ์และคัดลอก Content ได้ ปุ่มบันทึกปิดไว้

UI ใช้ Tailwind CSS ธีมมืด ออกแบบสำหรับมือถือเป็นหลัก: ช่องกรอกขนาด 16px ขึ้นไป
และปุ่มคัดลอก/บันทึกที่เข้าถึงได้จากด้านล่างหน้าจอ ภายในมีเพียงคลังเนื้อหาและพื้นที่เขียน
ไม่มี sidebar, bottom navigation หรือ dashboard; `/login/` ยังเปิดการ์ด Login เดียวกับหน้าแรกได้
เมื่อทำ authentication จริงสำเร็จจะพาไป `/contents/`; ตอนนี้ยังไม่มี session หรือ route protection

ข้อความใน editor อยู่ใน React state เท่านั้น ออกจากหน้าหรือ refresh แล้วหาย
ไม่มี localStorage, cookies หรือฐานข้อมูลสำหรับเก็บข้อความในรอบนี้
การคัดลอกต้องใช้ browser ที่รองรับ Clipboard API บน localhost หรือ HTTPS

## โครงสร้าง

```text
apps/web/         Next.js static frontend
apps/api/         Go HTTP server และ Dockerfile
apps/gateway/     Cloudflare Worker, Container lifecycle และ static asset routing
contracts/       OpenAPI ของ endpoint ที่ทำจริงแล้ว
docs/            ข้อตกลงสถาปัตยกรรมและผลตรวจรับ
scripts/         Node.js scripts สำหรับ Windows/macOS/Linux
```

อ่านรายละเอียดและข้อตกลงรอบถัดไปใน [Architecture](docs/architecture.md)

## Backend stack และฐานข้อมูล

- Fiber v3.6.1: routes, middleware, error handling และ graceful shutdown
- GORM v1.31.2 + PostgreSQL driver v1.6.3: optional connection layer, ไม่มี AutoMigrate
- Goose v3.29.0: SQL migrations แยกเป็นคำสั่ง ไม่รันเมื่อเปิด API

หากไม่ตั้ง `DATABASE_URL` API จะรัน health-only ได้ตามเดิม เมื่อกำหนดแล้วจะเชื่อมและ ping
PostgreSQL ภายใน 5 วินาทีก่อนเริ่มรับ request; ถ้าเชื่อมไม่ได้จะหยุดพร้อม error ที่ไม่แสดง DSN/password
Connection pool จำกัด 5 connections, idle 2, idle timeout 5 นาที และ lifetime 30 นาที
GORM ใช้ simple protocol เพื่อไม่ผูก prepared statement cache กับ session ของ pooled connection

| คำสั่ง                        | หน้าที่                                 |
| ----------------------------- | --------------------------------------- |
| `pnpm db:create create_users` | สร้าง template SQL สำหรับเติมและ review |
| `pnpm db:status`              | ดูสถานะ migrations                      |
| `pnpm db:up`                  | ใช้ migrations ที่ยังไม่ได้รัน          |
| `pnpm db:down`                | ย้อนกลับ migration ล่าสุดหนึ่งรายการ    |

ตั้ง `MIGRATION_DATABASE_URL` เป็น **direct URL** สำหรับ migration โดยเฉพาะ; URL สำหรับ runtime
ใน `DATABASE_URL` สามารถเป็น pooled URL ได้ คำสั่ง migration ไม่ fallback ไปใช้ runtime URL
รายละเอียดอยู่ใน [Migrations](apps/api/migrations/README.md)
ขณะนี้ยังไม่มี SQL migrations: status/up/down จะแจ้งว่าไม่มีงานและไม่เชื่อมฐานข้อมูล

## Environment variables

ไม่ต้องสร้างไฟล์ environment เพื่อรันตามค่าเริ่มต้น

| ส่วน | ตัวแปร                     | ค่าเริ่มต้น / วิธีใช้                                 |
| ---- | -------------------------- | ----------------------------------------------------- |
| Web  | `NEXT_PUBLIC_API_BASE_URL` | dev: `http://localhost:8080/api/v1`; build: `/api/v1` |
| Go   | `PORT`                     | `8080`                                                |
| Go   | `APP_ENV`                  | `development`; Container ตั้งเป็น `production`        |

- Web อ่าน `.env.local` ของ Next.js ได้ ใช้ `.env.example` เป็นตัวอย่าง
- Go อ่าน environment ของ process เท่านั้น ไม่โหลด `.env` อัตโนมัติ
- `pnpm dev` กำหนด Go port 8080 และ development mode ให้ตรงกับ frontend
- หากต้องปรับ Go port ให้ใช้ `pnpm dev:api` และ `pnpm dev:web` แยก terminal พร้อมปรับ web API URL
- Production build ต้องปล่อย `NEXT_PUBLIC_API_BASE_URL` ว่าง/ไม่ตั้ง เพื่อเรียก same-origin gateway ห้ามนำ local override ไปใช้ build สำหรับ deploy
- ตัวแปร `NEXT_PUBLIC_*` ถูกใส่ใน browser bundle จึงห้ามใช้เก็บ secrets
- `DATABASE_URL` เป็น optional สำหรับ runtime; `MIGRATION_DATABASE_URL` ใช้เฉพาะ migration command เมื่อมี SQL files ทั้งสองต้องเก็บเป็น server-side secrets

ตัวอย่างรัน Go แยกด้วย PowerShell:

```powershell
$env:PORT = '8080'
$env:APP_ENV = 'development'
pnpm dev:api
```

## Cloudflare local preview

1. ใช้ Linux/macOS หรือ WSL บน Windows: Wrangler รุ่นนี้ยังไม่รองรับ local Containers บน Windows native; เปิด Docker engine และตรวจ `docker info` ให้ผ่าน
2. ปิด local web API override ถ้ามีใน `.env.local`
3. รัน `pnpm preview:cloudflare`
4. เปิด http://localhost:8787 และ http://localhost:8787/api/v1/health
5. ทดสอบ refresh `/login/`, `/contents/`, `/contents/editor/` และตรวจว่า URL ที่ไม่มีคืน 404

การเริ่ม Container ครั้งแรกอาจใช้เวลา หน้าเว็บมีปุ่มตรวจ API ซ้ำได้
API routing ส่ง 503 พร้อม `Retry-After` เมื่อ backend ยังไม่พร้อม
ถ้า Docker engine ไม่ทำงาน การทดสอบ Container จะยังทำไม่ได้ ถึงแม้ build และ typecheck ผ่านแล้ว

สำหรับ Windows ที่ยังไม่พร้อมใช้ WSL ให้ใช้ `pnpm dev` ทำงานตามปกติ หรือ `pnpm preview:assets`
เพื่อตรวจ production HTML/JS และ 404 ผ่าน Workers runtime จริง โดย API ในโหมด assets จะคืน 503
โหมดนี้สร้าง config ชั่วคราวใน `.wrangler/` ที่ Git ignore และไม่เปลี่ยน production configuration
หากจะใช้ WSL ให้ติดตั้ง Node/pnpm/Go และ dependencies ในสภาพแวดล้อม Linux แยกจาก Windows

ยังไม่ได้สร้างบริการ Cloudflare, Neon, domain หรือ secrets ใด ๆ
Cloudflare Containers สำหรับ deploy จริงต้องใช้ Workers Paid plan

## การตรวจรับ

```powershell
pnpm check
pnpm build
```

จากนั้นตรวจหน้าจอ desktop/mobile และ health API ที่ `/api/v1/health` (ไม่มีการ์ดสถานะ API บนหน้าเว็บแล้ว)
Go tests ตรวจ HTTP boundary: health, unknown path, unsupported method และ CORS ของ local development
ผลการตรวจในสภาพแวดล้อมเริ่มต้นอยู่ใน [Validation](docs/validation.md)

## แหล่งอ้างอิง

- [Next.js static export](https://nextjs.org/docs/app/guides/static-exports)
- [Cloudflare static frontend + container backend](https://developers.cloudflare.com/containers/examples/container-backend/)
- [Cloudflare Containers](https://developers.cloudflare.com/containers/)
- [Fiber](https://docs.gofiber.io/)
- [GORM PostgreSQL](https://gorm.io/docs/connecting_to_the_database.html)
- [Goose](https://pressly.github.io/goose/)

Repository: [HectorRussia/infu-foundry](https://github.com/HectorRussia/infu-foundry) ใช้ branch `main`
แบ่ง commit ตามส่วนงานด้วย Conventional Commits เช่น `feat(api)`, `feat(web)`, `feat(gateway)` และ `docs`
