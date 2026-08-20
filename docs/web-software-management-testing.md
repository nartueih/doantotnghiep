# Kiểm thử giao diện quản lý phần mềm

Kịch bản này kiểm tra danh mục sản phẩm phần mềm, dữ liệu license liên kết, tìm kiếm, lọc, thêm và chỉnh sửa trên Web Admin. Dùng memory storage và dữ liệu demo nên không cần Docker hoặc database.

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

## 2. Kiểm tra danh mục bằng Admin

1. Đăng nhập `admin@local.test` / `ChangeMe123!`.
2. Chọn **Phần mềm** trong menu bên trái.
3. Với dữ liệu demo mặc định, các thẻ phải hiển thị:
   - Tổng sản phẩm: `6`.
   - Nhà phát hành: `5`.
   - License liên kết: `6`.
   - Tổng số seat: `32`, trong đó `14` seat đang sử dụng.
4. Bảng phải có Microsoft 365, Adobe Creative Cloud, Figma, JetBrains All Products Pack, Windows 11 Pro và Zoom Workplace.
5. Kiểm tra một số dữ liệu liên kết:
   - Microsoft 365 có `1` license và `4/5` seat.
   - Adobe Creative Cloud có `1` license và `3/3` seat.
   - Windows 11 Pro có `1` license và `4/6` seat.
   - Zoom Workplace có `1` license và `0/10` seat.

## 3. Kiểm tra tìm kiếm và lọc

1. Tìm `Adobe`, kết quả phải có Adobe Creative Cloud.
2. Tìm `24H2`, kết quả phải có Windows 11 Pro.
3. Tìm `Zoom Workplace Business`, kết quả phải có Zoom Workplace vì từ khóa khớp tên license liên kết.
4. Lọc publisher **Microsoft**, chỉ Microsoft 365 và Windows 11 Pro còn lại.
5. Lọc **Adobe**, chỉ Adobe Creative Cloud còn lại.
6. Đưa bộ lọc về **Tất cả nhà phát hành**.

## 4. Thêm phần mềm

Chọn **Thêm phần mềm** và nhập:

| Trường | Giá trị |
| --- | --- |
| Tên phần mềm | Notion |
| Nhà phát hành | Notion Labs |
| Phiên bản | Business |
| Mô tả | Không gian tài liệu và cộng tác nội bộ |

Chọn **Thêm phần mềm**. Kết quả đúng:

- Hộp thoại đóng và có thông báo thành công.
- Tổng sản phẩm tăng thành `7`.
- Nhà phát hành tăng thành `6`.
- Notion có trạng thái chưa có license.
- Tổng license và seat không thay đổi.

## 5. Kiểm tra dữ liệu không hợp lệ

1. Bỏ trống tên hoặc nhà phát hành: giao diện phải yêu cầu nhập đầy đủ.
2. Tạo lại Notion với cùng publisher và phiên bản nhưng khác chữ hoa/thường: backend phải báo sản phẩm đã tồn tại.
3. Đổi phiên bản thành một giá trị khác thì có thể tạo, vì tính duy nhất dựa trên tên, publisher và phiên bản.

## 6. Chỉnh sửa phần mềm

1. Tại dòng Notion, chọn nút chỉnh sửa.
2. Đổi tên thành `Notion Workspace`.
3. Đổi phiên bản thành `Enterprise`.
4. Đổi mô tả thành `Không gian tri thức và cộng tác cho doanh nghiệp`.
5. Lưu thay đổi.

Kết quả đúng là tên, phiên bản, mô tả và ngày cập nhật thay đổi ngay; số license vẫn bằng `0`.

## 7. Kiểm tra quyền IT Manager

1. Đăng xuất Admin và đăng nhập `it.manager@local.test` / `ChangeMe123!`.
2. Mở **Phần mềm**.
3. Danh sách, thống kê, tìm kiếm và lọc phải hoạt động.
4. IT Manager phải có nút **Thêm phần mềm** và nút chỉnh sửa, đúng với quyền backend hiện tại.
5. Thử thêm một sản phẩm hoặc sửa Notion Workspace để xác nhận thao tác thành công.

## 8. Kiểm tra quyền Employee

Đăng nhập `anh.nguyen@local.test` / `ChangeMe123!`, rồi mở `#/software`. Kết quả đúng là thông báo không có quyền truy cập vì backend trả về `403 Forbidden`.

## 9. Kiểm tra responsive

Thu nhỏ trình duyệt xuống chiều rộng điện thoại:

- Các thẻ thống kê chuyển thành hai hoặc một cột.
- Menu mobile mở và đóng bình thường.
- Ô tìm kiếm và bộ lọc không tràn màn hình.
- Bảng có thể cuộn ngang.
- Form thêm/chỉnh sửa chiếm toàn màn hình và cuộn được.
