# Kiểm tra danh mục phần mềm

Module dùng memory storage nên chưa yêu cầu database. Admin và IT Manager được quản lý danh mục; Employee không có quyền truy cập.

## API

| Method | Endpoint | Chức năng |
| --- | --- | --- |
| GET | `/api/v1/software` | Xem danh sách phần mềm |
| POST | `/api/v1/software` | Thêm phần mềm |
| PUT | `/api/v1/software/:id` | Cập nhật toàn bộ thông tin phần mềm |

## 1. Khởi động lại backend

Nhấn `Ctrl+C` tại terminal server rồi chạy lại:

```powershell
$env:STORAGE_DRIVER = "memory"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
go run ./cmd/api
```

## 2. Đăng nhập Admin

```powershell
$loginBody = @{
    email = "admin@local.test"
    password = "ChangeMe123!"
} | ConvertTo-Json

$login = Invoke-RestMethod `
    -Method Post `
    -Uri http://localhost:8080/api/v1/auth/login `
    -ContentType "application/json" `
    -Body $loginBody

$adminHeaders = @{ Authorization = "Bearer $($login.tokens.access_token)" }
```

## 3. Xem danh sách ban đầu

```powershell
$software = Invoke-RestMethod `
    -Method Get `
    -Uri http://localhost:8080/api/v1/software `
    -Headers $adminHeaders

$software.total
```

Kết quả ban đầu là `0`.

## 4. Thêm Adobe Photoshop

```powershell
$softwareBody = @{
    name = "Adobe Photoshop"
    publisher = "Adobe"
    version = "2026"
    description = "Phần mềm chỉnh sửa hình ảnh"
} | ConvertTo-Json

$createdSoftware = Invoke-RestMethod `
    -Method Post `
    -Uri http://localhost:8080/api/v1/software `
    -Headers $adminHeaders `
    -ContentType "application/json" `
    -Body $softwareBody

$createdSoftware | Format-List id, name, publisher, version, description
```

API trả HTTP `201 Created` và sinh UUID cho phần mềm.

## 5. Kiểm tra chống tạo trùng

Chạy lại chính request POST ở bước 4. Kết quả đúng là HTTP `409 Conflict` với lỗi `software product already exists`. Việc kiểm tra trùng không phân biệt chữ hoa/chữ thường.

## 6. Cập nhật phần mềm

```powershell
$updatedBody = @{
    name = "Adobe Photoshop"
    publisher = "Adobe Inc."
    version = "2026"
    description = "Phần mềm chỉnh sửa và thiết kế hình ảnh"
} | ConvertTo-Json

$updatedSoftware = Invoke-RestMethod `
    -Method Put `
    -Uri "http://localhost:8080/api/v1/software/$($createdSoftware.id)" `
    -Headers $adminHeaders `
    -ContentType "application/json" `
    -Body $updatedBody

$updatedSoftware | Format-List name, publisher, version, description
```

Publisher phải chuyển thành `Adobe Inc.`.

