# Kiểm thử Web Device Management

Trang **Thiết bị** cho phép Admin và IT Manager quản lý tài sản, bảo hành, người đang sử dụng và các license cấp trực tiếp cho thiết bị.

## 1. Khởi động môi trường memory

Terminal backend tại `D:\Đồ án\backend`:

```powershell
$env:STORAGE_DRIVER = "memory"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
$env:HTTP_ADDRESS = ":8081"
$env:SEED_DEMO_DATA = "true"
go run ./cmd/api
```

Terminal web tại `D:\Đồ án\web`:

```powershell
npm.cmd run dev
```

Mở `http://localhost:5173`, đăng nhập bằng `admin@local.test` / `ChangeMe123!` và chọn **Thiết bị**.

## 2. Kiểm tra danh sách và thống kê demo

Kết quả ban đầu đúng:

- Tổng thiết bị: `6`.
- Khả dụng: `2`.
- Đã bàn giao: `2`.
- Cần chú ý: `2` gồm bảo trì và thanh lý.

Thử tìm theo mã tài sản, serial, người dùng hoặc tên license. Sau đó thử tất cả bộ lọc trạng thái.

## 3. Thêm và chỉnh sửa thiết bị

1. Nhấn **Thêm thiết bị**.
2. Nhập mã `TEST-LT-001`, tên `Laptop kiểm thử`, loại `Laptop` và serial `TEST-SERIAL-001`.
3. Nhập ngày mua và ngày hết hạn bảo hành hợp lệ rồi lưu.
4. Kiểm tra thiết bị mới có trạng thái **Khả dụng**.
5. Nhấn biểu tượng chỉnh sửa, đổi model rồi lưu lại.

Các rào chắn cần kiểm tra:

- Mã tài sản và serial không được trùng.
- Mã tài sản, tên và loại thiết bị không được bỏ trống.
- Ngày hết hạn bảo hành không được trước ngày mua.

## 4. Bàn giao và thu hồi thiết bị

1. Tại `TEST-LT-001`, nhấn biểu tượng người dùng.
2. Chọn một tài khoản đang hoạt động và xác nhận bàn giao.
3. Kiểm tra trạng thái chuyển thành **Đã bàn giao** và tên người nhận xuất hiện.
4. Nút đổi trạng thái phải bị vô hiệu hóa khi thiết bị đang được bàn giao.
5. Nhấn biểu tượng thu hồi, kiểm tra thông báo rồi xác nhận.
6. Thiết bị phải trở lại trạng thái **Khả dụng**.

## 5. Đổi trạng thái

1. Nhấn biểu tượng bánh răng tại thiết bị khả dụng.
2. Chuyển sang **Đang bảo trì** và xác nhận.
3. Kiểm tra thiết bị không thể được bàn giao khi đang bảo trì.
4. Chuyển lại **Khả dụng**.
5. Thử các bộ lọc bảo trì, thanh lý và thất lạc.

## 6. Kiểm tra liên kết license thiết bị

Thiết bị demo `DEMO-LT-001`, `DEMO-LT-002`, `DEMO-WS-001` và `DEMO-MB-001` đang có Windows 11 Pro Volume. Tên license phải xuất hiện tại cột **License thiết bị**.

Khi thiết bị có nhiều license, bảng hiển thị license đầu tiên và số license còn lại. Việc bàn giao hoặc thu hồi thiết bị khỏi người dùng không tự động thu hồi license cấp trực tiếp cho thiết bị; thao tác đó được thực hiện tại trang **Cấp phát**.
