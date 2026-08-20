# Kiểm thử Web Assignment Management

Trang **Cấp phát** cho phép Admin và IT Manager xem lịch sử, cấp license cho người dùng hoặc thiết bị và thu hồi seat an toàn.

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

Mở `http://localhost:5173` và đăng nhập bằng `admin@local.test` / `ChangeMe123!`.

## 2. Kiểm tra danh sách

1. Chọn **Cấp phát** ở menu bên trái.
2. Kiểm tra bốn thẻ thống kê: đang hoạt động, theo người dùng, theo thiết bị và đã thu hồi.
3. Kiểm tra dữ liệu demo xuất hiện trong bảng, bản ghi mới nhất nằm trên cùng.
4. Thử tìm theo tên license hoặc người nhận.
5. Thử lần lượt các bộ lọc hoạt động, đã thu hồi, người dùng và thiết bị.

## 3. Cấp license cho người dùng

1. Nhấn **Cấp license**.
2. Chọn một license loại người dùng hoặc hỗn hợp còn seat.
3. Chọn một người dùng chưa nhận license đó.
4. Nhập ghi chú và xác nhận.
5. Kiểm tra thông báo thành công, bản ghi mới xuất hiện và số cấp phát hoạt động tăng một.
6. Sang trang **License**, kiểm tra `used_seats` tăng và `available_seats` giảm.

## 4. Cấp license cho thiết bị

1. Mở lại hộp **Cấp license**.
2. Chọn license loại thiết bị hoặc hỗn hợp.
3. Với license hỗn hợp, chuyển đối tượng sang **Thiết bị**.
4. Chọn một thiết bị không ở trạng thái thanh lý hoặc thất lạc và xác nhận.
5. Kiểm tra bản ghi có hình thức **Thiết bị**.

## 5. Thu hồi

1. Tại một bản ghi đang hoạt động, nhấn biểu tượng thu hồi ở cuối dòng.
2. Kiểm tra hộp thoại nói rõ seat sẽ được trả lại và lịch sử vẫn được giữ.
3. Nhấn **Xác nhận thu hồi**.
4. Kiểm tra trạng thái chuyển thành **Đã thu hồi**, nút thu hồi biến mất và thống kê được cập nhật.
5. Sang trang **License**, kiểm tra seat đã được trả lại.

## 6. Các rào chắn cần kiểm tra

- License hết hạn, sắp hiệu lực, đã lưu trữ hoặc hết seat không xuất hiện trong danh sách có thể cấp.
- License loại người dùng không cho chọn thiết bị và ngược lại.
- Người dùng bị khóa, thiết bị thanh lý hoặc thất lạc không thể được chọn.
- Đối tượng đang giữ cùng license không xuất hiện lại trong danh sách chọn.
- Bản ghi đã thu hồi vẫn tồn tại trong bộ lọc lịch sử.
