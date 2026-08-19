# Kiểm thử lưu trữ license

Tính năng này dùng cơ chế lưu trữ mềm thay vì xóa vĩnh viễn. License, activation key, lịch sử cấp phát và Audit Log vẫn được giữ lại.

## 1. Chạy backend bằng memory

Trong terminal thứ nhất tại `D:\Đồ án\backend`:

```powershell
$env:STORAGE_DRIVER = "memory"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
$env:HTTP_ADDRESS = ":8081"
$env:SEED_DEMO_DATA = "true"
go run ./cmd/api
```

Trong terminal thứ hai tại `D:\Đồ án\web`:

```powershell
npm.cmd run dev
```

Mở `http://localhost:5173`, đăng nhập bằng `admin@local.test` / `ChangeMe123!`, sau đó vào trang **License**.

## 2. Kiểm thử trên giao diện

1. Tạo một license mới với ít nhất 1 seat và chưa cấp phát.
2. Nhấn biểu tượng hộp lưu trữ ở cuối dòng.
3. Kiểm tra hộp xác nhận nói rõ dữ liệu và lịch sử vẫn được giữ.
4. Chọn **Xác nhận lưu trữ**.
5. Kiểm tra dòng license có trạng thái **Đã lưu trữ**, không còn nút sửa hoặc nút lưu trữ.
6. Chọn bộ lọc **Đã lưu trữ** và kiểm tra license vừa thao tác xuất hiện.
7. Kiểm tra license đã lưu trữ không còn được tính vào số liệu vận hành trên Dashboard.

Với license đang có cấp phát hoạt động, nút lưu trữ bị vô hiệu hóa. Cần thu hồi tất cả cấp phát trước.

## 3. Kiểm thử trực tiếp API

Sau khi đăng nhập và có `$adminHeaders`, lấy một license chưa cấp phát rồi chạy:

```powershell
$licenses = Invoke-RestMethod -Method Get -Uri "http://localhost:8081/api/v1/licenses" -Headers $adminHeaders
$candidate = $licenses.items | Where-Object { $_.used_seats -eq 0 -and -not $_.archived_at } | Select-Object -First 1

$archived = Invoke-RestMethod `
    -Method Patch `
    -Uri "http://localhost:8081/api/v1/licenses/$($candidate.id)/archive" `
    -Headers $adminHeaders

$archived | Format-List name, lifecycle_status, archived_at
```

Kết quả đúng: `lifecycle_status` bằng `archived` và `archived_at` có thời gian.

Gọi lại cùng endpoint phải nhận `409 Conflict`. Nếu license còn cấp phát hoạt động, API cũng phải trả về `409 Conflict`.

## 4. PostgreSQL sau này

Khi chuyển sang PostgreSQL, chạy migration ban đầu rồi migration lưu trữ theo đúng thứ tự:

```powershell
psql $env:DATABASE_URL -f backend/migrations/001_initial_schema.sql
psql $env:DATABASE_URL -f backend/migrations/002_license_archiving.sql
```
