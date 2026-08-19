# Kiểm tra Audit Log

Audit Log ghi lại các thao tác thay đổi dữ liệu quan trọng và việc xem plaintext license key. Admin và IT Manager được xem log; Employee không được truy cập.

## API

| Method | Endpoint | Chức năng |
| --- | --- | --- |
| GET | `/api/v1/audit-logs` | Xem audit log mới nhất |
| GET | `/api/v1/licenses/:id/key` | Xem license key và tạo log `view_key` |

API audit hỗ trợ các query parameter:

- `action`: lọc theo hành động, ví dụ `create`, `update`, `assign`, `revoke`, `view_key`, `archive`.
- `entity_type`: lọc theo đối tượng, ví dụ `license`, `device`, `license_assignment`.
- `actor_id`: lọc theo người thao tác.
- `limit`: số bản ghi trả về, mặc định 50 và tối đa 200.

Ví dụ:

```powershell
Invoke-RestMethod `
    -Method Get `
    -Uri "http://localhost:8080/api/v1/audit-logs?action=create&entity_type=license&limit=20" `
    -Headers $adminHeaders
```

## Sự kiện được ghi

| Action | Entity type | Thao tác |
| --- | --- | --- |
| `create` | `user`, `department`, `software_product`, `license`, `device` | Tạo dữ liệu |
| `update` | `department`, `software_product`, `license`, `device` | Cập nhật dữ liệu |
| `status_change` | `user`, `device` | Khóa/mở user hoặc đổi trạng thái thiết bị |
| `assign` | `device`, `license_assignment` | Giao thiết bị hoặc cấp license |
| `revoke` | `license_assignment` | Thu hồi license |
| `view_key` | `license` | Xem plaintext license key |
| `archive` | `license` | Lưu trữ license không còn cấp phát hoạt động |

## Quy tắc bảo mật

- Audit log lưu actor, action, đối tượng, metadata an toàn, IP và thời gian.
- Metadata tự loại các trường chứa `password`, `token`, `secret`, `license_key` hoặc `encrypted_key`.
- License key chỉ được trả về sau khi sự kiện `view_key` được ghi thành công.
- Audit log không cung cấp API sửa hoặc xóa.

Chạy kiểm thử tự động từ thư mục `backend`:

```powershell
go test -count=1 ./...
go vet ./...
```
