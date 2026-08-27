# Thiết lập PostgreSQL local

Hướng dẫn này cấu hình database thật cho backend, Web và Android. Các client chỉ gọi REST API; không kết nối trực tiếp PostgreSQL.

## 1. Kiểm tra PostgreSQL

```powershell
psql --version
Get-Service postgresql*
pg_isready -h localhost -p 5432
```

Service phải ở trạng thái `Running` và `pg_isready` phải báo `accepting connections`.

## 2. Tạo role và hai database

```powershell
psql -U postgres -h localhost -d postgres
```

Trong `psql`, chạy:

```sql
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'license_admin') THEN
        CREATE ROLE license_admin LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE;
    END IF;
END
$$;
\password license_admin
SELECT 'CREATE DATABASE license_manager OWNER license_admin'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'license_manager')\gexec
SELECT 'CREATE DATABASE license_manager_test OWNER license_admin'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'license_manager_test')\gexec
\q
```

`license_manager_test` chỉ dùng cho kiểm thử tự động và có thể bị xóa dữ liệu.

## 3. Khai báo biến môi trường trong PowerShell

Mở terminal tại `backend`:

```powershell
Set-Location "D:\Đồ án\backend"
$credential = Get-Credential -UserName "license_admin" -Message "Nhập mật khẩu PostgreSQL"
$databasePassword = $credential.GetNetworkCredential().Password
$encodedDatabasePassword = [Uri]::EscapeDataString($databasePassword)
$mainDatabaseURL = "postgres://license_admin:$encodedDatabasePassword@localhost:5432/license_manager?sslmode=disable"
$testDatabaseURL = "postgres://license_admin:$encodedDatabasePassword@localhost:5432/license_manager_test?sslmode=disable"
$env:DATABASE_URL = $mainDatabaseURL
$env:TEST_DATABASE_URL = $testDatabaseURL
$env:STORAGE_DRIVER = "postgres"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
```

Tạo khóa mã hóa license đúng một lần cho máy local:

```powershell
$keyBytes = New-Object byte[] 32
$generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
$generator.GetBytes($keyBytes)
$generator.Dispose()
$env:LICENSE_ENCRYPTION_KEY = [Convert]::ToBase64String($keyBytes)
[Environment]::SetEnvironmentVariable("LICENSE_ENCRYPTION_KEY", $env:LICENSE_ENCRYPTION_KEY, "User")
```

Không tạo khóa mới sau khi đã lưu license. Khóa mới không giải mã được dữ liệu cũ. Không commit URL có mật khẩu, JWT secret hoặc encryption key.

## 4. Chạy migration cho hai database

```powershell
$env:DATABASE_URL = $mainDatabaseURL
go run ./cmd/migrate up
go run ./cmd/migrate status

$env:DATABASE_URL = $testDatabaseURL
go run ./cmd/migrate up
go run ./cmd/migrate status

$env:DATABASE_URL = $mainDatabaseURL
```

Cả hai lệnh `status` phải hiển thị version `001` đến `004` ở trạng thái `applied`.

## 5. Tạo Admin ban đầu

```powershell
$env:DEV_ADMIN_EMAIL = "admin@local.test"
$env:DEV_ADMIN_PASSWORD = Read-Host "Nhập mật khẩu Admin phát triển"
go run ./cmd/seed
```

Chạy lại lệnh seed phải báo `already exists`; lệnh không tạo trùng và không đổi mật khẩu đã lưu.

## 6. Khởi động backend

```powershell
$env:DATABASE_URL = $mainDatabaseURL
$env:STORAGE_DRIVER = "postgres"
$env:SEED_DEMO_DATA = "false"
go run ./cmd/api
```

Log phải có `storage":"postgres"`. Trong terminal khác:

```powershell
Invoke-RestMethod http://localhost:8080/health/live
Invoke-RestMethod http://localhost:8080/health/ready
```

Kết quả lần lượt là `ok` và `ready`.

## 7. Xử lý lỗi thường gặp

- `psql is not recognized`: mở terminal mới sau khi cài PostgreSQL hoặc thêm thư mục `bin` của PostgreSQL vào `PATH`.
- `password authentication failed for user "Tran Hieu"`: biến `DATABASE_URL` đang trống hoặc gõ sai. Dùng đúng `$env:DATABASE_URL`, không thêm dấu `\` vào tên biến.
- Không kết nối được cổng `5432`: chạy `Get-Service postgresql*` và `pg_isready -h localhost -p 5432`; kiểm tra ứng dụng khác có chiếm cổng không.
- `database migration required`: chạy `go run ./cmd/migrate up`, rồi kiểm tra bằng `go run ./cmd/migrate status`.
- `LICENSE_ENCRYPTION_KEY must be a base64-encoded 32-byte key`: tạo khóa theo Bước 3 và chạy `[Convert]::FromBase64String($env:LICENSE_ENCRYPTION_KEY).Length`; kết quả phải là `32`.
- PowerShell tự hỏi password cho user Windows: URL kết nối không được truyền vào `psql`. Dùng `psql "$env:DATABASE_URL" -c "SELECT current_database(), current_user;"`.

## 8. Backup local

```powershell
pg_dump "$env:DATABASE_URL" -Fc -f license_manager.backup
```

Không commit file `.backup`; file có thể chứa dữ liệu người dùng và license đã mã hóa.

