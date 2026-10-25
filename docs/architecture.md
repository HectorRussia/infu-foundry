# Architecture — Infu Foundry

## ขอบเขตและเจ้าของความรับผิดชอบ

```text
Browser
  → Cloudflare Worker
      → Workers Static Assets: Next.js export
      → /api/*: Go Container
                   → Neon PostgreSQL (รอบถัดไป)
```

ใช้ pnpm workspace สำหรับ web/gateway และ Go module แยกสำหรับ backend
Next.js เป็น UI อย่างเดียว ไม่มี Next.js route handlers, Server Actions หรือ runtime SSR
Go เป็นเจ้าของ business logic, authentication และ database access ในอนาคต
Gateway ทำ routing และ Container lifecycle เท่านั้น

## Frontend

- App Router + TypeScript strict + Tailwind CSS utilities; global CSS กำหนด theme tokens และ base styles
- ธีมมืดและ mobile-first: หน้า Login แยกจาก workspace shell โดยไม่มี sidebar; ภายในมี sidebar บน desktop และเมนูพับได้บนมือถือ มีเพียงคลังเนื้อหาและ Editor หัวข้อ/Content ไม่มี dashboard หรือ bottom navigation
- `/login/` เป็น alias ของหน้าตา Login ที่ `/`; local dev ใช้บัญชีทดสอบกับ sessionStorage และ client redirect ไป `/contents/` ส่วน production ยังไม่มี authentication หรือ route protection
- Dev login เปิดเฉพาะ NODE_ENV=development และไม่ส่ง credentials ไป Go/Neon; sessionStorage ไม่เก็บรหัสผ่านและไม่ใช่ security boundary ต้องแทนที่ด้วย server-side authentication ในรอบจริง
- `src/app`: routes/layout; `components`: shared UI; `features`: auth/content UI; `lib`: API configuration
- Static export และ trailing slash ทำให้ทุก route มี HTML สำหรับเปิดตรงและ refresh
- หลัง build มีขั้นตอนเติม filename aliases สำหรับ RSC segment files ที่ Next.js 16 export บน Windows ผิดรูปแบบ เป็น no-op เมื่อ output ถูกต้อง และไม่แก้ไฟล์ใน node_modules; ดู [upstream issue](https://github.com/vercel/next.js/issues/85374)
- Editor เป็น `/contents/editor/` คงที่ รอบถัดไปใช้ query parameter ระบุ ID ของรายการ ไม่ใช้ dynamic route ที่ต้องรู้ ID ตอน build
- Local dev มี Login จำลองที่ระบุชัดเจน; production Login และปุ่มบันทึกเนื้อหายังปิดไว้ ไม่มี content persistence
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
การต่อ Neon จริงและเพิ่ม schema จะทำในรอบฟีเจอร์ต่อไป

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
