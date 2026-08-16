# Chạy và kiểm tra Authentication khi chưa có database

Hướng dẫn này dùng `STORAGE_DRIVER=memory`. Backend tạo một tài khoản Admin tạm trong RAM, vì vậy chưa cần Docker hoặc PostgreSQL. Dữ liệu sẽ mất mỗi khi tắt ứng dụng.

## Bước 1: kiểm tra Go

Mở PowerShell và chạy:

```powershell
go version
```

Kết quả cần có dạng `go version go1.x.x windows/amd64`. Nếu PowerShell báo không tìm thấy `go`, cần cài Go hoặc thêm Go vào biến môi trường `PATH` trước khi tiếp tục.

## Bước 2: mở đúng thư mục backend

```powershell
Set-Location "D:\Đồ án\backend"
```

Kiểm tra thư mục hiện tại:

```powershell
Get-Location
Get-ChildItem
```

Bạn phải nhìn thấy `cmd`, `internal`, `migrations`, `go.mod` và `go.sum`.

## Bước 3: chạy kiểm thử tự động

```powershell
go test ./...
```

Các package `internal/config`, `internal/httpapi` và `internal/modules/auth` phải hiển thị `ok`. Dòng `[no test files]` không phải lỗi.

Muốn xem tên từng test:

```powershell
go test -v ./internal/modules/auth
```

## Bước 4: cấu hình chế độ không database

Chạy các lệnh sau trong cùng cửa sổ PowerShell:

```powershell
$env:APP_ENV = "development"
$env:HTTP_ADDRESS = ":8080"
$env:STORAGE_DRIVER = "memory"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
$env:JWT_ISSUER = "license-manager"
$env:ACCESS_TOKEN_TTL = "15m"
$env:REFRESH_TOKEN_TTL = "168h"
$env:DEV_ADMIN_EMAIL = "admin@local.test"
$env:DEV_ADMIN_PASSWORD = "ChangeMe123!"
```

`JWT_SECRET` phải có ít nhất 32 ký tự. Tất cả giá trị trong phần này chỉ dùng cho môi trường phát triển.

## Bước 5: chạy backend

```powershell
go run ./cmd/api
```

Giữ cửa sổ này mở. Log cần cho biết HTTP server chạy tại `:8080`, storage là `memory`, đồng thời cảnh báo dữ liệu sẽ mất khi tắt server.

## Bước 6: kiểm tra health API

Mở một cửa sổ PowerShell thứ hai:

```powershell
Invoke-RestMethod http://localhost:8080/health/live
Invoke-RestMethod http://localhost:8080/health/ready
```

Kết quả mong đợi:

```text
status
------
ok

status
------
ready
```

## Bước 7: đăng nhập

Trong cửa sổ PowerShell thứ hai:

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

$login
```

Kết quả phải có `user`, `tokens.access_token` và `tokens.refresh_token`.

## Bước 8: lấy thông tin người đang đăng nhập

```powershell
$headers = @{
    Authorization = "Bearer $($login.tokens.access_token)"
}

$me = Invoke-RestMethod `
    -Method Get `
    -Uri http://localhost:8080/api/v1/auth/me `
    -Headers $headers

$me
```

Kết quả phải có email `admin@local.test` và role `admin`.

Thử bỏ token để kiểm tra bảo vệ endpoint:

```powershell
Invoke-RestMethod -Method Get -Uri http://localhost:8080/api/v1/auth/me
```

PowerShell phải báo HTTP `401 Unauthorized`. Đây là kết quả đúng.

## Bước 9: làm mới token

```powershell
$refreshBody = @{
    refresh_token = $login.tokens.refresh_token
} | ConvertTo-Json

$refreshed = Invoke-RestMethod `
    -Method Post `
    -Uri http://localhost:8080/api/v1/auth/refresh `
    -ContentType "application/json" `
    -Body $refreshBody

$refreshed.tokens
```

Refresh token mới phải khác refresh token cũ. Token cũ không thể dùng lại vì backend thực hiện token rotation.

## Bước 10: đăng xuất

Phải đăng xuất bằng refresh token mới nhận ở bước 9:

```powershell
$logoutBody = @{
    refresh_token = $refreshed.tokens.refresh_token
} | ConvertTo-Json

Invoke-WebRequest `
    -Method Post `
    -Uri http://localhost:8080/api/v1/auth/logout `
    -ContentType "application/json" `
    -Body $logoutBody
```

Kết quả đúng là HTTP `204 No Content`. Sau đó refresh token này không thể dùng lại.

## Bước 11: tắt backend

Quay lại cửa sổ đang chạy server và nhấn `Ctrl+C`. Server sẽ thực hiện graceful shutdown. Vì đang dùng memory storage, tài khoản và các phiên đăng nhập tạm sẽ được tạo lại từ đầu ở lần chạy tiếp theo.

## Khi nào mới cần tạo database?

Sau khi hoàn thành và hiểu toàn bộ luồng trên, bước tiếp theo mới là:

1. Cài PostgreSQL trực tiếp hoặc sửa Docker Desktop.
2. Tạo database `license_manager` và user `license_admin`.
3. Chạy migration `001_initial_schema.sql`.
4. Chuyển `STORAGE_DRIVER` từ `memory` sang `postgres`.
5. Tạo tài khoản Admin thật trong bảng `users`.

Không cần thực hiện các bước database ở thời điểm hiện tại.

