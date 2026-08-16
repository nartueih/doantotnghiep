# Kiểm tra module quản lý người dùng

Các bước này dùng `STORAGE_DRIVER=memory`, không yêu cầu Docker hoặc database. Phải khởi động lại backend sau khi cập nhật source code.

## API đã triển khai

| Method | Endpoint | Quyền | Chức năng |
| --- | --- | --- | --- |
| GET | `/api/v1/users` | Admin | Xem danh sách người dùng |
| POST | `/api/v1/users` | Admin | Tạo người dùng |
| PATCH | `/api/v1/users/:id/status` | Admin | Khóa hoặc mở khóa người dùng |

## 1. Khởi động lại backend

Tại cửa sổ đang chạy backend, nhấn `Ctrl+C`. Sau đó chạy lại các biến môi trường và `go run ./cmd/api` như trong `getting-started-without-docker.md`.

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

$adminHeaders = @{
    Authorization = "Bearer $($login.tokens.access_token)"
}
```

## 3. Xem danh sách ban đầu

```powershell
$users = Invoke-RestMethod `
    -Method Get `
    -Uri http://localhost:8080/api/v1/users `
    -Headers $adminHeaders

$users | ConvertTo-Json -Depth 5
```

Kết quả ban đầu phải có `total` bằng `1`, tương ứng tài khoản Admin tạm.

## 4. Tạo Employee

```powershell
$createUserBody = @{
    email = "employee@local.test"
    password = "Employee123!"
    full_name = "Test Employee"
    employee_code = "EMP-001"
    role = "employee"
} | ConvertTo-Json

$createdUser = Invoke-RestMethod `
    -Method Post `
    -Uri http://localhost:8080/api/v1/users `
    -Headers $adminHeaders `
    -ContentType "application/json" `
    -Body $createUserBody

$createdUser | Format-List *
```

Kết quả đúng là HTTP `201 Created`; user có role `employee`, status `active` và một UUID mới.

Mật khẩu phải có ít nhất 10 ký tự, bao gồm chữ hoa, chữ thường và chữ số.

## 5. Kiểm tra danh sách sau khi tạo

```powershell
$users = Invoke-RestMethod `
    -Method Get `
    -Uri http://localhost:8080/api/v1/users `
    -Headers $adminHeaders

$users.total
$users.items | Format-Table email, employee_code, role, status
```

`total` phải bằng `2`.

## 6. Kiểm tra phân quyền Employee

Đăng nhập bằng tài khoản vừa tạo:

```powershell
$employeeLoginBody = @{
    email = "employee@local.test"
    password = "Employee123!"
} | ConvertTo-Json

$employeeLogin = Invoke-RestMethod `
    -Method Post `
    -Uri http://localhost:8080/api/v1/auth/login `
    -ContentType "application/json" `
    -Body $employeeLoginBody

$employeeHeaders = @{
    Authorization = "Bearer $($employeeLogin.tokens.access_token)"
}
```

Thử xem danh sách người dùng:

```powershell
Invoke-RestMethod `
    -Method Get `
    -Uri http://localhost:8080/api/v1/users `
    -Headers $employeeHeaders
```

Kết quả đúng là `403 Forbidden` vì chỉ Admin được quản lý người dùng.

## 7. Khóa Employee

```powershell
$statusBody = @{ status = "locked" } | ConvertTo-Json

$lockedUser = Invoke-RestMethod `
    -Method Patch `
    -Uri "http://localhost:8080/api/v1/users/$($createdUser.id)/status" `
    -Headers $adminHeaders `
    -ContentType "application/json" `
    -Body $statusBody

$lockedUser | Format-List email, status
```

Status phải chuyển thành `locked`. Nếu đăng nhập lại bằng tài khoản Employee, API phải trả về `403 Forbidden`.

## 8. Mở khóa Employee

```powershell
$statusBody = @{ status = "active" } | ConvertTo-Json

$activeUser = Invoke-RestMethod `
    -Method Patch `
    -Uri "http://localhost:8080/api/v1/users/$($createdUser.id)/status" `
    -Headers $adminHeaders `
    -ContentType "application/json" `
    -Body $statusBody

$activeUser | Format-List email, status
```

Status phải trở lại `active` và Employee có thể đăng nhập lại.

