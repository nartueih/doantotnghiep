# Kiểm tra Employee License Key Access

Chức năng này cho phép Employee tự lấy activation key khi Admin/IT bật quyền trên license. Quyền sở hữu, trạng thái cấp phát và trạng thái license luôn được kiểm tra ở backend.

## Quy tắc bảo mật

- `allow_employee_key_view` mặc định là `false`.
- Employee chỉ xem được key qua assignment đang `active` của chính họ hoặc thiết bị đang giao cho họ.
- License phải đang ở trạng thái `active` và đã cấu hình key.
- Không thể dùng assignment ID của người khác hoặc assignment đã thu hồi.
- Giao diện chỉ gọi endpoint sau khi Employee xác nhận.
- Mỗi lần xem thành công tạo Audit Log `view_key` với `access_scope=employee_self_service`.
- Plaintext key không được ghi vào metadata, danh sách license hoặc session storage.

## API

| Method | Endpoint | Chức năng |
| --- | --- | --- |
| GET | `/api/v1/me/licenses` | Trả `can_view_key` cho từng cấp phát |
| GET | `/api/v1/me/licenses/:assignment_id/key` | Giải mã key sau khi kiểm tra quyền |

## Kiểm tra giao diện

1. Đăng nhập Admin bằng `admin@local.test` / `ChangeMe123!`.
2. Vào **License**, chỉnh sửa một license đang cấp cho Nguyễn Hoàng Anh.
3. Bật **Cho phép nhân viên tự xem key kích hoạt** và lưu.
4. Xác nhận bảng hiển thị **Nhân viên được xem key**.
5. Đăng xuất và đăng nhập `anh.nguyen@local.test` / `ChangeMe123!`.
6. Vào License của tôi, nhấn **Xem key**.
7. Xác nhận dialog cảnh báo xuất hiện trước, plaintext chưa được tải.
8. Nhấn **Xác nhận xem key**, kiểm tra key hiển thị và nút sao chép hoạt động.
9. Đăng nhập lại Admin, vào **Nhật ký**, lọc hành động **Xem khóa**.
10. Xác nhận có log của Employee nhưng metadata không chứa plaintext key.

## Kiểm tra từ chối quyền

1. Admin tắt công tắc cho phép Employee xem key.
2. Employee tải lại portal; nút **Xem key** phải biến mất.
3. Nếu gọi endpoint trực tiếp bằng assignment ID đó, API trả `403`.
4. Assignment của người khác hoặc đã thu hồi trả `404`.
5. License hết hạn, sắp hiệu lực hoặc đã lưu trữ trả `409`.
6. License chưa cấu hình key trả `404`.

## PostgreSQL migration

Sau khi đã chạy migration cũ, chạy thêm:

```powershell
psql $env:DATABASE_URL -f migrations/003_employee_license_key_access.sql
```

## Kiểm tra tự động

```powershell
Set-Location backend
go test ./...
go vet ./...

Set-Location ..\web
npm.cmd run lint
npm.cmd run build
```

