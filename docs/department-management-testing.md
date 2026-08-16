# Kiểm tra quản lý phòng ban

Module phòng ban hỗ trợ Admin quản lý danh mục phòng ban và gán phòng ban khi tạo người dùng. IT Manager chỉ được xem danh sách; Employee không được truy cập.

## API

| Method | Endpoint | Quyền | Chức năng |
| --- | --- | --- | --- |
| GET | `/api/v1/departments` | Admin, IT Manager | Xem danh sách phòng ban |
| POST | `/api/v1/departments` | Admin | Tạo phòng ban |
| PUT | `/api/v1/departments/:id` | Admin | Cập nhật phòng ban |
| POST | `/api/v1/users` | Admin | Tạo người dùng, có thể gửi `department_id` |

## Quy tắc nghiệp vụ

- Tên và mã phòng ban là bắt buộc.
- Tên và mã không được trùng, không phân biệt chữ hoa/chữ thường.
- Mã phòng ban được chuẩn hóa thành chữ hoa.
- `department_id` khi tạo người dùng là tùy chọn; nếu có thì phải trỏ tới phòng ban tồn tại.
- Phản hồi người dùng có `department_id` và `department_name` khi người dùng thuộc một phòng ban.

## Kết quả HTTP cần kiểm tra

- Tạo phòng ban hợp lệ: `201 Created`.
- Tạo trùng tên hoặc mã: `409 Conflict`.
- Cập nhật phòng ban hợp lệ: `200 OK`.
- Cập nhật ID không tồn tại: `404 Not Found`.
- Tạo người dùng với `department_id` không tồn tại: `422 Unprocessable Entity`.
- IT Manager tạo/cập nhật phòng ban hoặc Employee xem danh sách: `403 Forbidden`.

Chạy kiểm thử tự động từ thư mục `backend`:

```powershell
go test -count=1 ./...
go vet ./...
```
