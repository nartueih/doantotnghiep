# Enterprise License Manager

Hệ thống quản lý thiết bị và giấy phép phần mềm trong doanh nghiệp trên nền tảng Web và Android.

## Phạm vi MVP

- Quản lý phòng ban, người dùng và vai trò.
- Quản lý thiết bị của công ty.
- Quản lý danh mục phần mềm và license.
- Theo dõi số lượng seat, ngày hết hạn và chi phí.
- Cấp phát/thu hồi license cho người dùng hoặc thiết bị.
- Cảnh báo license sắp hết hạn.
- Ghi nhật ký các thao tác quan trọng.

## Công nghệ dự kiến

- Backend: Go, Gin, PostgreSQL.
- Web: React + TypeScript (giai đoạn sau).
- Android: Kotlin + Jetpack Compose (giai đoạn sau).
- Hạ tầng phát triển: Docker Compose.

## Chạy backend ở môi trường phát triển

Nếu chưa có Docker hoặc database, làm lần lượt theo [hướng dẫn chạy không Docker](docs/getting-started-without-docker.md).

Yêu cầu cơ bản: Go 1.25+. Docker Desktop chỉ cần khi chuyển sang PostgreSQL.

```powershell
docker compose up -d postgres
Set-Location backend
$env:APP_ENV = "development"
$env:HTTP_ADDRESS = ":8080"
$env:DATABASE_URL = "postgres://license_admin:license_admin@localhost:5432/license_manager?sslmode=disable"
$env:SHUTDOWN_TIMEOUT = "10s"
go mod download
go run ./cmd/api
```

File `backend/.env.example` là mẫu cấu hình để IDE, Docker hoặc công cụ quản lý môi trường sử dụng; ứng dụng chỉ đọc biến môi trường và không tự động nạp file `.env`.

Kiểm tra API:

```powershell
Invoke-RestMethod http://localhost:8080/health/live
Invoke-RestMethod http://localhost:8080/health/ready
```

Tài liệu nghiệp vụ ban đầu nằm trong [docs/product-requirements.md](docs/product-requirements.md).
