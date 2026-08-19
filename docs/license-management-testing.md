# Kiểm tra module quản lý license

Module hỗ trợ license `subscription` và `perpetual`, cấp theo `user`, `device` hoặc `mixed`. Mã license được mã hóa AES-256-GCM; API chỉ trả `key_hint` và không trả plaintext hoặc ciphertext.

## API

| Method | Endpoint | Chức năng |
| --- | --- | --- |
| GET | `/api/v1/licenses` | Xem danh sách license và số seat |
| POST | `/api/v1/licenses` | Tạo license |
| PUT | `/api/v1/licenses/:id` | Cập nhật license |
| PATCH | `/api/v1/licenses/:id/archive` | Lưu trữ license không còn cấp phát hoạt động |

Admin và IT Manager được sử dụng các API này. Employee nhận `403 Forbidden`.

## 1. Khởi động lại backend

Ở memory development, ứng dụng tự dùng một encryption key tạm. PostgreSQL và production bắt buộc phải cung cấp `LICENSE_ENCRYPTION_KEY` riêng.

```powershell
$env:STORAGE_DRIVER = "memory"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
go run ./cmd/api
```

## 2. Đăng nhập Admin

```powershell
$loginBody = @{ email = "admin@local.test"; password = "ChangeMe123!" } | ConvertTo-Json
$login = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/auth/login -ContentType "application/json" -Body $loginBody
$adminHeaders = @{ Authorization = "Bearer $($login.tokens.access_token)" }
```

## 3. Tạo software product

License luôn phải thuộc một software product tồn tại.

```powershell
$softwareBody = @{
    name = "Adobe Photoshop"
    publisher = "Adobe"
    version = "2026"
    description = "Phần mềm chỉnh sửa hình ảnh"
} | ConvertTo-Json

$software = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/software -Headers $adminHeaders -ContentType "application/json" -Body $softwareBody
```

## 4. Tạo subscription license

```powershell
$licenseBody = @{
    software_product_id = $software.id
    name = "Photoshop Teams"
    license_type = "subscription"
    assignment_type = "user"
    seat_count = 25
    license_key = "SECRET-LICENSE-KEY-1234567890"
    vendor = "Adobe"
    purchased_at = "2026-01-10"
    starts_at = "2026-01-10"
    expires_at = "2027-01-09"
    cost = 1200
    currency = "USD"
    notes = "Annual company subscription"
} | ConvertTo-Json

$createdLicense = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/licenses -Headers $adminHeaders -ContentType "application/json" -Body $licenseBody
$createdLicense | Format-List *
```

Kết quả cần có:

- `key_hint` bằng `****7890`.
- `used_seats` bằng `0`.
- `available_seats` bằng `25`.
- `lifecycle_status` bằng `active` nếu license đang trong thời hạn.
- Không có field `license_key` hoặc `encrypted_key`.

## 5. Xem danh sách

```powershell
$licenses = Invoke-RestMethod -Method Get -Uri http://localhost:8080/api/v1/licenses -Headers $adminHeaders
$licenses.total
$licenses.items | Format-Table name, license_type, seat_count, used_seats, available_seats, expires_at, lifecycle_status, key_hint
```

## 6. Kiểm tra subscription thiếu ngày hết hạn

```powershell
$invalidBody = @{
    software_product_id = $software.id
    name = "Invalid subscription"
    license_type = "subscription"
    assignment_type = "user"
    seat_count = 1
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/licenses -Headers $adminHeaders -ContentType "application/json" -Body $invalidBody
```

Kết quả đúng là `400 Bad Request` với lỗi `subscription licenses require an expiration date`.

## 7. Cập nhật mà không thay license key

```powershell
$updatedBody = @{
    software_product_id = $software.id
    name = "Photoshop Teams - Updated"
    license_type = "subscription"
    assignment_type = "user"
    seat_count = 50
    vendor = "Adobe"
    purchased_at = "2026-01-10"
    starts_at = "2026-01-10"
    expires_at = "2027-01-09"
    cost = 1500
    currency = "USD"
    notes = "Expanded to 50 seats"
} | ConvertTo-Json

$updatedLicense = Invoke-RestMethod -Method Put -Uri "http://localhost:8080/api/v1/licenses/$($createdLicense.id)" -Headers $adminHeaders -ContentType "application/json" -Body $updatedBody
$updatedLicense | Format-List name, seat_count, available_seats, key_hint
```

Seat phải tăng lên `50` và `key_hint` vẫn là `****7890`, chứng minh key cũ được giữ lại.

## 8. Tạo perpetual license không có ngày hết hạn

```powershell
$perpetualBody = @{
    software_product_id = $software.id
    name = "Photoshop Perpetual"
    license_type = "perpetual"
    assignment_type = "device"
    seat_count = 5
    vendor = "Adobe"
    purchased_at = "2026-08-17"
    cost = 5000
    currency = "USD"
} | ConvertTo-Json

$perpetual = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/licenses -Headers $adminHeaders -ContentType "application/json" -Body $perpetualBody
```

Request phải thành công dù không có `expires_at`.

## Khóa mã hóa cho production

Sinh một khóa ngẫu nhiên bằng PowerShell:

```powershell
$keyBytes = New-Object byte[] 32
$rng = [Security.Cryptography.RandomNumberGenerator]::Create()
$rng.GetBytes($keyBytes)
$env:LICENSE_ENCRYPTION_KEY = [Convert]::ToBase64String($keyBytes)
$rng.Dispose()
```

Khóa này phải được lưu trong secret manager hoặc biến môi trường production. Nếu mất khóa, các license key đã mã hóa sẽ không thể giải mã lại.

