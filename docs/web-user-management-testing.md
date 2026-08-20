# Kiểm thử giao diện quản lý người dùng

Kịch bản này kiểm tra danh sách, tìm kiếm, lọc, tạo tài khoản và khóa/mở khóa người dùng trên Web Admin. Dùng memory storage và dữ liệu demo nên không cần Docker hoặc database.

## 1. Khởi động backend

Mở terminal thứ nhất:

```powershell
cd "D:\Đồ án\backend"
$env:STORAGE_DRIVER = "memory"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
$env:HTTP_ADDRESS = ":8081"
$env:SEED_DEMO_DATA = "true"
go run ./cmd/api
```

Giữ terminal này chạy. Backend phải báo `http server started` tại cổng `8081` và đã seed dữ liệu demo.

## 2. Khởi động web

Mở terminal thứ hai:

```powershell
cd "D:\Đồ án\web"
npm.cmd run dev
```

Mở địa chỉ Vite hiển thị trong terminal, thông thường là `http://localhost:5173`.

## 3. Kiểm tra danh sách bằng Admin

1. Đăng nhập bằng `admin@local.test` / `ChangeMe123!`.
2. Chọn **Người dùng** trong menu bên trái.
3. Xác nhận menu được đánh dấu đang hoạt động và bảng có dữ liệu demo.
4. Với dữ liệu demo mặc định, các thẻ phải hiển thị:
   - Tổng người dùng: `6`.
   - Đang hoạt động: `6`.
   - Nhóm quản lý: `2`.
   - Đã khóa: `0`.
5. Tài khoản đang đăng nhập phải có nhãn **Bạn** và nút khóa bị vô hiệu hóa.

## 4. Kiểm tra tìm kiếm và bộ lọc

1. Tìm `DEMO-002`, kết quả phải có Nguyễn Hoàng Anh.
2. Tìm `Information Technology`, kết quả phải có người thuộc phòng IT.
3. Tìm `it.manager@local.test`, kết quả phải có Trần Minh Quân.
4. Lọc vai trò **Quản lý IT**, chỉ tài khoản IT Manager còn lại.
5. Lọc vai trò **Nhân viên**, chỉ các tài khoản Employee còn lại.
6. Chọn lại **Tất cả vai trò** và lọc **Đã khóa**, bảng ban đầu phải rỗng.
7. Đưa cả hai bộ lọc về **Tất cả**.

## 5. Tạo tài khoản

Chọn **Thêm người dùng** và nhập:

| Trường | Giá trị |
| --- | --- |
| Họ và tên | Nhân Viên Kiểm Thử |
| Email | test.user@local.test |
| Mã nhân viên | TEST-001 |
| Phòng ban | Finance |
| Vai trò | Nhân viên |
| Mật khẩu | TestPassword123 |

Chọn **Tạo tài khoản**. Kết quả đúng:

- Hộp thoại đóng và xuất hiện thông báo thành công.
- Tổng người dùng tăng lên `7`.
- Đang hoạt động tăng lên `7`.
- Người dùng mới có phòng ban Finance, vai trò Nhân viên và trạng thái Hoạt động.

## 6. Kiểm tra dữ liệu không hợp lệ

1. Mở form và bỏ trống họ tên hoặc mã nhân viên: giao diện phải yêu cầu nhập đầy đủ.
2. Nhập email sai định dạng: giao diện phải báo email không hợp lệ.
3. Nhập mật khẩu `weak`: giao diện phải nhắc quy tắc tối thiểu 10 ký tự, có chữ hoa, chữ thường và chữ số.
4. Tạo lại email `test.user@local.test`: backend phải báo email đã được sử dụng.
5. Đổi email nhưng dùng lại mã `TEST-001`: backend phải báo mã nhân viên đã được sử dụng.

## 7. Khóa và mở khóa

1. Tìm `TEST-001` và chọn nút khóa ở cuối dòng.
2. Xác nhận hộp thoại cảnh báo người dùng sẽ không thể đăng nhập.
3. Chọn **Xác nhận khóa**.
4. Trạng thái tài khoản chuyển thành **Đã khóa**; số Đã khóa tăng lên `1` và Đang hoạt động giảm còn `6`.
5. Thử đăng nhập `test.user@local.test` / `TestPassword123`: backend phải từ chối tài khoản bị khóa.
6. Đăng nhập lại Admin, mở trang Người dùng và chọn nút mở khóa của `TEST-001`.
7. Chọn **Xác nhận mở khóa**; trạng thái trở lại **Hoạt động** và tài khoản có thể đăng nhập lại.

## 8. Kiểm tra quyền IT Manager

1. Đăng xuất Admin.
2. Đăng nhập `it.manager@local.test` / `ChangeMe123!`.
3. Mở trang **Người dùng**.
4. Danh sách, tìm kiếm và bộ lọc phải hoạt động.
5. Phải xuất hiện thông báo chế độ chỉ xem.
6. Không được có nút **Thêm người dùng** hoặc nút khóa/mở khóa ở từng dòng.

## 9. Kiểm tra quyền Employee

Đăng nhập bằng một tài khoản Employee demo, ví dụ `anh.nguyen@local.test` / `ChangeMe123!`, rồi mở `#/users`. Kết quả đúng là màn hình thông báo không có quyền truy cập, vì backend trả về `403 Forbidden`.

## 10. Kiểm tra responsive

Thu nhỏ trình duyệt xuống chiều rộng điện thoại:

- Các thẻ thống kê tự chuyển thành hai hoặc một cột.
- Nút menu mobile mở được sidebar.
- Thanh tìm kiếm và bộ lọc không tràn màn hình.
- Bảng có thể cuộn ngang.
- Form tạo người dùng chiếm toàn màn hình và cuộn được.
