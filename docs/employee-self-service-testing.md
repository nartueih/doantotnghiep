# Kiểm tra Employee Self-service

Self-service cho phép người dùng đã đăng nhập xem thiết bị và license của chính mình. User ID luôn được lấy từ access token; API không nhận user ID từ URL, query hoặc request body.

## API

| Method | Endpoint | Chức năng |
| --- | --- | --- |
| GET | `/api/v1/auth/me` | Xem thông tin tài khoản hiện tại |
| GET | `/api/v1/me/devices` | Xem thiết bị đang được giao |
| GET | `/api/v1/me/licenses` | Xem license đang được cấp |

## Nguồn cấp license

- `assignment_source=user`: license được cấp trực tiếp cho người dùng.
- `assignment_source=device`: license được cấp cho một thiết bị đang thuộc người dùng; response có thêm `device_id` và `device_asset_code`.

## Quy tắc bảo mật

- Mọi endpoint `/me` đều yêu cầu access token hợp lệ.
- Query như `?user_id=...` bị bỏ qua hoàn toàn.
- Chỉ assignment có trạng thái `active` được hiển thị.
- Không trả license của user khác hoặc thiết bị của user khác.
- Không trả plaintext, ciphertext hoặc key hint của license.

## Ví dụ

```powershell
$myDevices = Invoke-RestMethod `
    -Method Get `
    -Uri "http://localhost:8080/api/v1/me/devices" `
    -Headers $employeeHeaders

$myLicenses = Invoke-RestMethod `
    -Method Get `
    -Uri "http://localhost:8080/api/v1/me/licenses" `
    -Headers $employeeHeaders

$myDevices.items
$myLicenses.items | Format-Table license_name, assignment_source, device_asset_code, expires_at
```

Chạy kiểm thử tự động từ thư mục `backend`:

```powershell
go test -count=1 ./...
go vet ./...
```
