# Infu Foundry — กติกาสำหรับ AI agents

คลังเนื้อหาส่วนตัวที่ใช้กัน 2 คน (Horn, Ploy) ทุกบัญชีเห็นเนื้อหาร่วมกัน ช่วงนี้เน้น MVP และค่าใช้จ่าย $0
ระบบ วิธีตั้งค่า และผลตรวจรับอยู่ใน docs อย่าเขียนซ้ำที่นี่:
[README](README.md) · [Architecture](docs/architecture.md) · [Deployment](docs/deployment.md) · [Validation](docs/validation.md)

กติกาการทำงานอยู่ที่ [.cursor/rules/](.cursor/rules/) ไฟล์ละหัวข้อ
ไฟล์นี้เป็นสารบัญเท่านั้น แก้รายละเอียดในไฟล์ `.mdc` แล้วอัปเดตตารางถ้ามีไฟล์ใหม่หรือไฟล์หาย
[AGENTS.md](AGENTS.md) ชี้มาที่นี่

| ไฟล์                                                                     | อ่านเมื่อ                                      |
| ------------------------------------------------------------------------ | ---------------------------------------------- |
| [git-workflow.mdc](.cursor/rules/git-workflow.mdc)                       | ทุกงาน: branch, commit, PR, งานซ้อน, ความลับ   |
| [database-migrations.mdc](.cursor/rules/database-migrations.mdc)         | migration, Neon, ลำดับ deploy ของ API          |
| [data-model.mdc](.cursor/rules/data-model.mdc)                           | contents, หมวดหมู่, การเรียงหรือปักหมุด, การลบ |
| [frontend.mdc](.cursor/rules/frontend.mdc)                               | หน้าเว็บ                                       |
| [go-api.mdc](.cursor/rules/go-api.mdc)                                   | โค้ด Go                                        |
| [github-actions-security.mdc](.cursor/rules/github-actions-security.mdc) | GitHub Actions                                 |
| [code-review.mdc](.cursor/rules/code-review.mdc)                         | เปิดหรือรีวิว PR                               |

คำสั่งที่ใช้บ่อยอยู่ใน README รัน `pnpm check` ก่อนส่งงาน
