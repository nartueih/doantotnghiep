# Kiểm tra Web Audit Log

Màn hình Nhật ký hoạt động dùng API `GET /api/v1/audit-logs` và chỉ cho phép Admin hoặc Quản lý IT truy cập. Nhật ký là dữ liệu chỉ đọc, không có thao tác sửa hoặc xóa.

## 1. Chuẩn bị backend

Mở terminal thứ nhất tại thư mục `backend`:

```powershell
$env:STORAGE_DRIVER = "memory"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
$env:HTTP_ADDRESS = ":8081"
$env:SEED_DEMO_DATA = "true"
go run ./cmd/api
```

Giữ terminal này chạy. Với memory storage, nhật ký sẽ mất khi backend dừng.

## 2. Khởi động web

Mở terminal thứ hai tại thư mục `web`:

```powershell
npm.cmd run dev
```

Mở địa chỉ Vite hiển thị trong terminal, thường là `http://localhost:5173`.

## 3. Tạo dữ liệu nhật ký

Đăng nhập Admin bằng `admin@local.test` / `ChangeMe123!`, sau đó thực hiện một số thao tác:

1. Tạo hoặc sửa một phần mềm.
2. Tạo hoặc sửa một license.
3. Tạo một cấp phát rồi thu hồi cấp phát đó.
4. Khóa rồi mở khóa một người dùng hoặc đổi trạng thái thiết bị.
5. Xem license key hoặc lưu trữ một license nếu dữ liệu cho phép.

Mỗi thao tác thành công sẽ tạo một sự kiện. Chọn **Nhật ký** ở thanh điều hướng để mở màn hình.

## 4. Kiểm tra danh sách và thống kê

Xác nhận:

- Menu Nhật ký được chọn và trang không còn nhãn "Sắp có".
- Bốn thẻ thống kê hiển thị tổng sự kiện, sự kiện hôm nay, số người thao tác và sự kiện nhạy cảm.
- Sự kiện mới nhất nằm trên cùng.
- Mỗi dòng có thời gian, người thao tác, hành động, loại đối tượng, nội dung chính và IP.
- Nút **Làm mới** tải lại dữ liệu mới nhất.

Nếu vừa khởi động lại memory backend và chưa thao tác gì, trạng thái trống là kết quả đúng.

## 5. Kiểm tra tìm kiếm và bộ lọc

1. Tìm theo tên/email người thao tác.
2. Tìm theo tên đối tượng, IP hoặc nội dung metadata như mã tài sản.
3. Lọc lần lượt theo hành động, đối tượng và người thao tác.
4. Kết hợp nhiều bộ lọc và kiểm tra số kết quả thay đổi.
5. Nhấn **Xóa lọc** để trở lại toàn bộ danh sách.
6. Nhập từ khóa không tồn tại và kiểm tra trạng thái không có kết quả.

## 6. Kiểm tra chi tiết và bảo mật

1. Nhấn biểu tượng con mắt ở cuối một dòng.
2. Xác nhận chi tiết hiển thị ID nhật ký, ID đối tượng, thời điểm, IP và metadata.
3. Nhấn nút đóng để thu gọn dòng.
4. Với sự kiện xem khóa, xác nhận chỉ có thông tin sự kiện; plaintext license key không xuất hiện.
5. Tìm các từ `password`, `token`, `secret`, `license_key`, `encrypted_key`; không metadata nhạy cảm nào được hiển thị.

## 7. Kiểm tra phân quyền

- `it.manager@local.test` / `ChangeMe123!`: xem được đầy đủ Nhật ký.
- `anh.nguyen@local.test` / `ChangeMe123!`: nhận màn hình từ chối quyền truy cập (HTTP 403).
- Phiên hết hạn: màn hình yêu cầu đăng nhập lại.

## 8. Kiểm tra responsive

Thu nhỏ trình duyệt:

- Bốn thẻ thống kê chuyển còn hai rồi một cột.
- Bộ lọc tự xuống dòng và vẫn sử dụng được.
- Bảng có thể cuộn ngang, không làm vỡ bố cục.
- Nút menu và nút làm mới vẫn bấm được.

## 9. Kiểm tra tự động

Tại thư mục gốc dự án:

```powershell
Set-Location backend
go test ./...
go vet ./...

Set-Location ..\web
npm.cmd run lint
npm.cmd run build
```

