# Kiểm thử giao diện quản lý phòng ban

Kịch bản này kiểm tra danh sách, thống kê nhân sự, tìm kiếm, thêm và chỉnh sửa phòng ban trên Web Admin. Dùng memory storage và dữ liệu demo nên không cần Docker hoặc database.

## 1. Khởi động hệ thống

Terminal backend:

```powershell
cd "D:\Đồ án\backend"
$env:STORAGE_DRIVER = "memory"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
$env:HTTP_ADDRESS = ":8081"
$env:SEED_DEMO_DATA = "true"
go run ./cmd/api
```

Terminal web:

```powershell
cd "D:\Đồ án\web"
npm.cmd run dev
```

Mở địa chỉ Vite hiển thị trong terminal, thông thường là `http://localhost:5173`.

## 2. Kiểm tra danh sách bằng Admin

1. Đăng nhập `admin@local.test` / `ChangeMe123!`.
2. Chọn **Phòng ban** trong menu bên trái.
3. Với dữ liệu demo mặc định, các thẻ phải hiển thị:
   - Tổng phòng ban: `3`.
   - Đã phân phòng: `5`.
   - Phòng đông nhất: `2` người; tên phòng có thể là Finance hoặc Information Technology vì hai phòng có cùng số người.
   - Chưa phân phòng: `1`, tương ứng Admin phát triển.
4. Bảng phải có Finance, Information Technology và Operations.
5. Mỗi dòng hiển thị mã phòng ban, số thành viên và các avatar đại diện.

## 3. Kiểm tra tìm kiếm

1. Tìm `FIN`, kết quả phải có Finance.
2. Tìm `Information Technology`, kết quả phải có phòng IT.
3. Tìm `DEMO-002` hoặc `Nguyễn Hoàng Anh`, kết quả phải có phòng IT.
4. Tìm một từ khóa không tồn tại, giao diện phải hiển thị trạng thái rỗng.

## 4. Tạo phòng ban

Chọn **Thêm phòng ban** và nhập:

| Trường | Giá trị |
| --- | --- |
| Tên phòng ban | Human Resources |
| Mã phòng ban | hr |

Chọn **Tạo phòng ban**. Kết quả đúng:

- Hộp thoại đóng và có thông báo thành công.
- Tổng phòng ban tăng thành `4`.
- Mã được chuẩn hóa thành `HR`.
- Phòng mới có `0` thành viên.

## 5. Kiểm tra dữ liệu không hợp lệ

1. Bỏ trống tên hoặc mã: giao diện phải yêu cầu nhập đầy đủ.
2. Tạo tên `Human Resources` với mã khác: backend phải báo tên đã được sử dụng.
3. Tạo tên khác nhưng mã `HR`: backend phải báo mã đã được sử dụng.

## 6. Chỉnh sửa và đồng bộ người dùng

1. Tại dòng Human Resources, chọn nút chỉnh sửa.
2. Đổi tên thành `People Operations`, giữ mã `HR` và lưu.
3. Tên trong bảng phải cập nhật ngay, số lượng thành viên vẫn là `0`.
4. Mở trang **Người dùng**, tạo tài khoản mới và chọn phòng People Operations.
5. Quay lại **Phòng ban**, phòng HR phải tăng thành `1` thành viên và hiển thị người vừa tạo.
6. Đổi tên phòng thành `People & Culture`, rồi mở lại **Người dùng**. Tên phòng của tài khoản vừa tạo phải đổi theo ngay cả khi dùng memory storage.

## 7. Kiểm tra quyền IT Manager

1. Đăng xuất Admin và đăng nhập `it.manager@local.test` / `ChangeMe123!`.
2. Mở **Phòng ban**.
3. Danh sách, thống kê và tìm kiếm phải hoạt động.
4. Phải có thông báo chế độ chỉ xem.
5. Không có nút **Thêm phòng ban** hoặc nút chỉnh sửa ở từng dòng.

## 8. Kiểm tra quyền Employee

Đăng nhập `anh.nguyen@local.test` / `ChangeMe123!`, rồi mở `#/departments`. Kết quả đúng là thông báo không có quyền truy cập vì backend trả về `403 Forbidden`.

## 9. Kiểm tra responsive

Thu nhỏ trình duyệt xuống chiều rộng điện thoại:

- Các thẻ thống kê chuyển thành hai hoặc một cột.
- Menu mobile mở và đóng bình thường.
- Ô tìm kiếm không tràn màn hình.
- Bảng có thể cuộn ngang.
- Form phòng ban hiển thị dạng sheet ở cuối màn hình và vẫn thao tác được.
