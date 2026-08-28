# Kiểm thử yêu cầu cấp license và thông báo website

Tài liệu này kiểm tra luồng lưu trữ PostgreSQL: nhân viên gửi hoặc hủy yêu cầu, Admin/IT duyệt hoặc từ chối, hệ thống tự cấp phát khi duyệt và gửi thông báo trong website cho nhân viên.

> Hoàn thành [thiết lập PostgreSQL local](postgresql-local-setup.md) trước. Dữ liệu phải còn nguyên sau khi backend khởi động lại.

## 1. Khởi động backend

Mở Terminal thứ nhất trong VS Code:

```powershell
Set-Location "D:\Đồ án\backend"
$env:STORAGE_DRIVER = "postgres"
$env:SEED_DEMO_DATA = "false"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
go run ./cmd/api
```

Giữ nguyên cửa sổ này. Kiểm tra backend từ Terminal thứ hai:

```powershell
Invoke-RestMethod http://localhost:8080/health/live
Invoke-RestMethod http://localhost:8080/health/ready
```

Cả hai kết quả phải có `status = ok`.

## 2. Khởi động web

Trong Terminal thứ hai:

```powershell
Set-Location "D:\Đồ án\web"
npm.cmd install
$env:VITE_API_PROXY_TARGET = "http://localhost:8080"
npm.cmd run dev
```

Mở địa chỉ Vite in trong Terminal, thường là `http://localhost:5173`.

Tài khoản demo dùng chung mật khẩu `ChangeMe123!`:

- Nhân viên: `anh.nguyen@local.test`
- Admin: `admin@local.test`
- IT Manager: `it.manager@local.test`

## 3. Nhân viên tạo yêu cầu

1. Đăng nhập bằng tài khoản nhân viên.
2. Tại khu vực **Yêu cầu license**, chọn **Tạo yêu cầu**.
3. Chọn một phần mềm, chọn độ ưu tiên và nhập lý do.
4. Bấm **Gửi yêu cầu**.
5. Xác nhận yêu cầu mới xuất hiện với trạng thái **Đang chờ**.
6. Thử gửi tiếp cùng phần mềm. Hệ thống phải chặn yêu cầu đang chờ bị trùng.

## 4. Nhân viên hủy yêu cầu

1. Tạo một yêu cầu khác đang ở trạng thái **Đang chờ**.
2. Bấm **Hủy yêu cầu** và xác nhận.
3. Trạng thái phải chuyển thành **Đã hủy**.
4. Yêu cầu đã hủy không còn nút hủy và Admin/IT không thể duyệt nó.

## 5. Admin hoặc IT duyệt yêu cầu

Mở cửa sổ ẩn danh hoặc một hồ sơ trình duyệt khác để giữ đồng thời hai phiên đăng nhập.

1. Đăng nhập bằng Admin hoặc IT Manager.
2. Chọn menu **Yêu cầu**.
3. Kiểm tra tìm kiếm và bộ lọc trạng thái/độ ưu tiên.
4. Tại một yêu cầu **Đang chờ**, bấm **Duyệt**.
5. Chọn license đúng phần mềm và còn seat; có thể nhập phản hồi.
6. Bấm **Duyệt và cấp license**.
7. Yêu cầu phải chuyển thành **Đã duyệt**.
8. Mở menu **Cấp phát** và xác nhận có bản cấp phát mới cho đúng nhân viên.
9. Quay lại phiên nhân viên, làm mới dữ liệu và xác nhận license mới xuất hiện trong danh sách của tôi.

Nếu không có license phù hợp còn seat, hộp thoại duyệt phải không cho xác nhận. Nếu seat vừa bị dùng hết, yêu cầu vẫn ở trạng thái chờ và giao diện hướng dẫn Admin/IT gửi phản hồi **Tạm hết license**.

## 6. Admin/IT từ chối vì tạm hết license

1. Nhân viên tạo thêm một yêu cầu đang chờ.
2. Trong phiên Admin/IT, mở **Yêu cầu** và bấm **Tạm hết**.
3. Lý do phải được chọn sẵn là **Tạm hết license**.
4. Nhập phản hồi, ví dụ `Hiện tại công ty đã hết seat, chúng tôi sẽ liên hệ khi có thêm.`
5. Xác nhận từ chối.
6. Yêu cầu phải chuyển thành **Đã từ chối** và không tạo bản cấp phát.

Có thể kiểm tra thêm hai lý do **Không được phê duyệt** và **Lý do khác**. Phản hồi không được để trống.

## 7. Nhân viên nhận và đọc thông báo

1. Quay lại phiên nhân viên.
2. Chờ tối đa 30 giây hoặc tải lại trang.
3. Chuông thông báo phải hiện số lượng chưa đọc.
4. Mở chuông và xác nhận nội dung duyệt/từ chối cùng phản hồi của Admin/IT.
5. Bấm một thông báo chưa đọc; số chưa đọc phải giảm.
6. Bấm **Đánh dấu tất cả đã đọc**; số chưa đọc phải về `0`.
7. Đóng/mở lại bảng thông báo; trạng thái đã đọc phải được giữ trong thời gian backend còn chạy.

## 8. Kiểm tra nhật ký

Trong phiên Admin/IT, mở **Nhật ký** và lọc thực thể `license_request` nếu giao diện hỗ trợ. Các thao tác tương ứng phải xuất hiện:

- `request`: nhân viên gửi yêu cầu;
- `cancel`: nhân viên hủy yêu cầu;
- `approve`: Admin/IT duyệt;
- `reject`: Admin/IT từ chối.

Metadata nhật ký và nội dung thông báo không được chứa `license_key` hoặc activation key.

## 9. Kiểm tra nhanh bằng API PowerShell

Đăng nhập nhân viên và lấy token:

```powershell
$employeeLogin = Invoke-RestMethod -Method Post `
  -Uri http://localhost:8080/api/v1/auth/login `
  -ContentType "application/json" `
  -Body (@{ email = "anh.nguyen@local.test"; password = "ChangeMe123!" } | ConvertTo-Json)
$employeeHeaders = @{ Authorization = "Bearer $($employeeLogin.tokens.access_token)" }
```

Lấy phần mềm có thể yêu cầu, tạo yêu cầu và xem danh sách của tôi:

```powershell
$catalog = Invoke-RestMethod -Uri http://localhost:8080/api/v1/me/requestable-software -Headers $employeeHeaders
$productID = $catalog.items[0].id

$createJSON = @{ software_product_id = $productID; priority = "normal"; reason = "Cần cho công việc" } | ConvertTo-Json
$created = Invoke-RestMethod -Method Post `
  -Uri http://localhost:8080/api/v1/me/license-requests `
  -Headers $employeeHeaders `
  -ContentType "application/json" `
  -Body ([System.Text.Encoding]::UTF8.GetBytes($createJSON))

Invoke-RestMethod -Uri http://localhost:8080/api/v1/me/license-requests -Headers $employeeHeaders
Invoke-RestMethod -Uri http://localhost:8080/api/v1/me/notifications -Headers $employeeHeaders
```

Đăng nhập Admin và xem yêu cầu:

```powershell
$adminLogin = Invoke-RestMethod -Method Post `
  -Uri http://localhost:8080/api/v1/auth/login `
  -ContentType "application/json" `
  -Body (@{ email = "admin@local.test"; password = "ChangeMe123!" } | ConvertTo-Json)
$adminHeaders = @{ Authorization = "Bearer $($adminLogin.tokens.access_token)" }

Invoke-RestMethod -Uri http://localhost:8080/api/v1/license-requests -Headers $adminHeaders
```

Từ chối yêu cầu vừa tạo để kiểm tra thông báo:

```powershell
$rejectJSON = @{ decision_reason = "out_of_stock"; response_note = "Tạm hết license, sẽ phản hồi lại sau." } | ConvertTo-Json
Invoke-RestMethod -Method Patch `
  -Uri "http://localhost:8080/api/v1/license-requests/$($created.id)/reject" `
  -Headers $adminHeaders `
  -ContentType "application/json" `
  -Body ([System.Text.Encoding]::UTF8.GetBytes($rejectJSON))

$notifications = Invoke-RestMethod -Uri http://localhost:8080/api/v1/me/notifications -Headers $employeeHeaders
$notifications

Invoke-RestMethod -Method Patch `
  -Uri "http://localhost:8080/api/v1/me/notifications/$($notifications.items[0].id)/read" `
  -Headers $employeeHeaders
```

## Kết quả đạt

Luồng đạt khi quyền truy cập đúng vai trò, yêu cầu chỉ chuyển trạng thái một lần, duyệt tạo đúng một cấp phát, từ chối không tạo cấp phát, thông báo chỉ đến đúng nhân viên và không có license key trong request/thông báo/nhật ký.
