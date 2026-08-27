# Kiểm thử PostgreSQL persistence

## 1. Chuẩn bị

Thiết lập `DATABASE_URL` và `TEST_DATABASE_URL` theo [hướng dẫn PostgreSQL local](postgresql-local-setup.md). Xác nhận URL test thực sự trỏ tới database có hậu tố `_test`; test sẽ từ chối chạy nếu trỏ nhầm database chính.

## 2. Chạy kiểm thử tích hợp

```powershell
Set-Location "D:\Đồ án\backend"
go test ./internal/modules/notifications -run Postgres -count=1 -v
go test ./internal/modules/licenserequests -run Postgres -count=1 -v
go test ./internal/integration -run TestPostgresWorkflowSurvivesReconnect -count=1 -v
```

Các test phải chạy thật, không có dòng `SKIP`. Chúng xóa dữ liệu trong `license_manager_test`, không tác động `license_manager`.

## 3. Ma trận kiểm tra đầy đủ

```powershell
Set-Location "D:\Đồ án\backend"
go test ./... -count=1
go vet ./...

Set-Location "D:\Đồ án\web"
npm.cmd test
npm.cmd run lint
npm.cmd run build

Set-Location "D:\Đồ án"
git diff --check
```

Tất cả lệnh phải có exit code `0`.

## 4. Smoke test sau restart

1. Chạy backend với `STORAGE_DRIVER=postgres`.
2. Dùng Web tạo Employee, phần mềm và license.
3. Đăng nhập Employee, gửi yêu cầu license.
4. Đăng nhập Admin, duyệt yêu cầu; xác nhận có đúng một assignment và một notification.
5. Dừng backend bằng `Ctrl+C`.
6. Chạy lại backend với cùng `DATABASE_URL` và `LICENSE_ENCRYPTION_KEY`.
7. Xác nhận request vẫn `approved`, assignment vẫn tồn tại, notification vẫn hiển thị và key của license vẫn đọc được theo đúng quyền.

Luồng từ chối cũng phải giữ request và notification sau restart nhưng không tạo assignment.
