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

Sau khi kiểm tra Authentication, tiếp tục với [hướng dẫn kiểm tra quản lý người dùng](docs/user-management-testing.md).

Giao diện quản trị người dùng được kiểm tra theo [hướng dẫn Web User Management](docs/web-user-management-testing.md).

Danh mục sản phẩm phần mềm được kiểm tra theo [hướng dẫn software catalog](docs/software-catalog-testing.md).

Giao diện quản trị phần mềm được kiểm tra theo [hướng dẫn Web Software Management](docs/web-software-management-testing.md).

Quy tắc và mã hóa license được kiểm tra theo [hướng dẫn license management](docs/license-management-testing.md).

Quy trình lưu trữ license an toàn được kiểm tra theo [hướng dẫn license archiving](docs/license-archiving-testing.md).

Quy trình tạo, bàn giao và thu hồi thiết bị nằm trong [hướng dẫn device management](docs/device-management-testing.md).

Giao diện quản trị thiết bị được kiểm tra theo [hướng dẫn Web Device Management](docs/web-device-management-testing.md).

Luồng sử dụng seat, cấp phát và thu hồi license nằm trong [hướng dẫn license assignment](docs/license-assignment-testing.md).

Giao diện quản trị cấp phát được kiểm tra theo [hướng dẫn Web Assignment Management](docs/web-assignment-management-testing.md).

Danh mục phòng ban và liên kết người dùng nằm trong [hướng dẫn department management](docs/department-management-testing.md).

Giao diện quản trị phòng ban được kiểm tra theo [hướng dẫn Web Department Management](docs/web-department-management-testing.md).

Nhật ký thao tác và quy tắc bảo vệ dữ liệu nhạy cảm nằm trong [hướng dẫn Audit Log](docs/audit-log-testing.md).

Giao diện xem và lọc nhật ký hoạt động được kiểm tra theo [hướng dẫn Web Audit Log](docs/web-audit-log-testing.md).

Số liệu tổng quan, chi phí và cảnh báo license nằm trong [hướng dẫn Dashboard & Alerts](docs/dashboard-alerts-testing.md).

Luồng nhân viên tự xem thiết bị và license của mình nằm trong [hướng dẫn Employee Self-service](docs/employee-self-service-testing.md).

Giao diện cổng thông tin riêng cho nhân viên được kiểm tra theo [hướng dẫn Web Employee Portal](docs/web-employee-portal-testing.md).

Quyền xem activation key có kiểm soát cho Employee được kiểm tra theo [hướng dẫn Employee License Key Access](docs/employee-license-key-testing.md).

Hợp đồng API và cách sử dụng Swagger UI nằm trong [hướng dẫn OpenAPI](docs/openapi-testing.md).

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
