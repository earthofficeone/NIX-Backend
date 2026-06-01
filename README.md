# NIX Backend

API สำหรับแอปบันทึกรายรับรายจ่าย (Go + Gin + MongoDB)

## ตั้งค่า

1. คัดลอก `.env.example` เป็น `.env` แล้วใส่ `MONGODB_URI` และ `JWT_SECRET`
2. รันเซิร์ฟเวอร์:

```bash
go run .
```

## Endpoints

| Method | Path | คำอธิบาย |
|--------|------|----------|
| GET | `/health` | ตรวจสถานะ |
| POST | `/api/auth/register` | สมัครสมาชิก |
| POST | `/api/auth/login` | เข้าสู่ระบบ |
| GET | `/api/auth/me` | ข้อมูลผู้ใช้ (Bearer token) |
| GET | `/api/transactions?month=YYYY-MM` | รายการ |
| POST | `/api/transactions` | สร้างรายการ |
| PUT | `/api/transactions/:id` | แก้ไข |
| DELETE | `/api/transactions/:id` | ลบ |
