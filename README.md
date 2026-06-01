# NIX Backend

API สำหรับแอปบันทึกรายรับรายจ่าย (Go + Gin + MongoDB)

## ตั้งค่า Local

1. คัดลอก `.env.example` เป็น `.env`
2. ใส่ค่าจาก MongoDB Atlas → **Connect** → **Drivers** → Go
3. รัน:

```bash
go run .
```

## Environment Variables

| ตัวแปร | คำอธิบาย |
|--------|----------|
| `PORT` | พอร์ต (Render ตั้งให้อัตโนมัติ) |
| `MONGODB_URI` | `mongodb+srv://...` จาก Atlas |
| `JWT_SECRET` | สตริงสุ่มยาว (production) |
| `CORS_ORIGIN` | URL frontend |
| `GIN_MODE` | `debug` หรือ `release` |

## Deploy บน Render

- **Build:** `go build -tags netgo -ldflags '-s -w' -o app`
- **Start:** `./app`

## แก้ error `tls: internal error` / `ReplicaSetNoPrimary`

1. **MongoDB Atlas → Network Access**
   - Add IP Address → **Allow Access from Anywhere** (`0.0.0.0/0`)
   - รอสถานะ Active (~1–2 นาที)

2. **Connection string ใน Render**
   - ใช้แบบ **`mongodb+srv://`** (ไม่ใช่ `mongodb://`)
   - คัดลอกจาก Atlas → Connect → Drivers → Go
   - **อย่า** ใส่เครื่องหมาย `"` รอบค่าใน Environment
   - รหัสผ่านมี `@ # %` ต้อง [URL encode](https://www.urlencoder.org/) เช่น `@` → `%40`

   ตัวอย่าง:

   ```
   mongodb+srv://user:MyP%40ss@cluster0.xxxxx.mongodb.net/nix?retryWrites=true&w=majority&appName=Cluster0
   ```

3. **Database Access**
   - User มีสิทธิ์ read/write บน database `nix`

4. Redeploy หลังแก้ Network Access / URI

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
