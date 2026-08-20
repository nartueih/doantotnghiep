# Kiểm tra Web Employee Portal

Employee Portal là không gian riêng cho nhân viên xem hồ sơ, thiết bị công ty và license đang được cấp. Dữ liệu luôn được giới hạn theo user ID trong access token.

## 1. Chạy backend

Mở terminal thứ nhất tại thư mục `backend`:

```powershell
$env:STORAGE_DRIVER = "memory"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
$env:HTTP_ADDRESS = ":8081"
$env:SEED_DEMO_DATA = "true"
go run ./cmd/api
```

## 2. Chạy web

Mở terminal thứ hai tại thư mục `web`:

```powershell
npm.cmd run dev
```

Mở địa chỉ Vite hiển thị, thường là `http://localhost:5173`.

## 3. Kiểm tra chuyển hướng theo vai trò

Đăng nhập bằng tài khoản nhân viên:

```text
Email: anh.nguyen@local.test
Mật khẩu: ChangeMe123!
```

Kết quả mong đợi:

- Tự động chuyển tới `#/portal`.
- Hiển thị Cổng thông tin nhân viên thay cho sidebar quản trị.
- Không có menu quản lý người dùng, license, thiết bị hoặc audit log.
- Header hiển thị đúng tên và mã nhân viên.

## 4. Kiểm tra hồ sơ và thống kê

Với dữ liệu demo của Nguyễn Hoàng Anh, xác nhận:

- Hồ sơ hiển thị email, mã nhân viên và phòng ban.
- Có `1` thiết bị đang giữ.
- Có `2` license: Microsoft 365 cấp trực tiếp và Windows 11 đi theo thiết bị.
- Thống kê nguồn cấp trực tiếp/theo thiết bị khớp với danh sách.
- Số license cần chú ý thay đổi đúng theo ngày chạy dữ liệu demo.

## 5. Kiểm tra thiết bị của tôi

Thiết bị demo mong đợi:

- Mã tài sản `LT-001`.
- Tên `Laptop Dell Latitude`.
- Hãng Dell và model Latitude 7450.
- Trạng thái đang sử dụng.
- Có serial number và ngày hết hạn bảo hành.

Nhấn nút làm mới ở tiêu đề khu vực và xác nhận dữ liệu vẫn tải bình thường.

## 6. Kiểm tra license của tôi

Xác nhận:

- Microsoft 365 hiển thị nguồn **Cấp trực tiếp cho bạn**.
- Windows 11 hiển thị nguồn **Theo thiết bị LT-001**.
- License thuê bao hiển thị ngày hết hạn.
- License vĩnh viễn không yêu cầu ngày hết hạn.
- Ghi chú của cấp phát được hiển thị.

Kiểm tra ba bộ lọc:

1. **Tất cả**: hiển thị cả hai license.
2. **Trực tiếp**: chỉ hiển thị Microsoft 365.
3. **Theo thiết bị**: chỉ hiển thị Windows 11.

## 7. Kiểm tra cảnh báo hết hạn

Khi license đã hết hạn hoặc còn tối đa 30 ngày:

- Thẻ **Cần chú ý** tăng số lượng.
- Banner cảnh báo xuất hiện.
- License có nhãn `Đã hết hạn`, `Hết hạn hôm nay` hoặc `Còn ... ngày`.
- Link **Xem license** cuộn tới danh sách license.

## 8. Kiểm tra bảo mật dữ liệu

Xác nhận giao diện không hiển thị:

- Key của license chưa được IT cho phép chia sẻ hoặc không thuộc Employee hiện tại.
- License được cấp cho nhân viên khác.
- Thiết bị được giao cho nhân viên khác.
- Chức năng sửa, cấp phát hoặc thu hồi. Nút xem key chỉ xuất hiện khi backend trả `can_view_key=true`.

Thử sửa URL thành `#/users`, `#/licenses` hoặc `#/audit`. Employee vẫn phải ở portal và không được thấy màn hình quản trị.

## 9. Kiểm tra các tài khoản khác

Đăng xuất rồi đăng nhập Admin hoặc IT Manager:

```text
admin@local.test / ChangeMe123!
it.manager@local.test / ChangeMe123!
```

Kết quả mong đợi:

- Admin và IT Manager tiếp tục vào khu quản trị cũ.
- Không bị chuyển nhầm sang Employee Portal.
- Các module quản trị vẫn hoạt động bình thường.

## 10. Kiểm tra responsive và trạng thái lỗi

- Thu nhỏ cửa sổ: hero, thống kê và hai danh sách chuyển sang một cột.
- Header mobile vẫn hiển thị logo, avatar và nút đăng xuất.
- Bộ lọc license sử dụng được trên màn hình nhỏ.
- Bảng hoặc thẻ không tràn ngang.
- Khi backend dừng, portal hiển thị lỗi và nút **Thử lại**.
- Khi phiên hết hạn, portal yêu cầu đăng nhập lại.

## 11. Kiểm tra tự động

```powershell
Set-Location backend
go test ./...
go vet ./...

Set-Location ..\web
npm.cmd run lint
npm.cmd run build
```

