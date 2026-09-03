# Kiểm tra yêu cầu bảo trì thiết bị

Module này cho phép Employee báo sự cố trên thiết bị đang được giao. Admin hoặc IT Manager tiếp nhận, hoàn thành hay từ chối và phản hồi qua thông báo website. Thông tin asset, serial, loại, hãng, model, ngày mua và bảo hành được chụp lại tại thời điểm gửi.

## 1. Chạy nhanh bằng memory

Mở terminal thứ nhất tại thư mục `backend` của nhánh:

```powershell
$env:STORAGE_DRIVER = "memory"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
$env:HTTP_ADDRESS = ":8081"
$env:SEED_DEMO_DATA = "true"
go run ./cmd/api
```

Mở terminal thứ hai tại thư mục `web`:

```powershell
npm.cmd install
npm.cmd run dev
```

Mở địa chỉ Vite hiển thị, thường là `http://localhost:5173`.

## 2. Employee tạo yêu cầu

1. Đăng nhập `anh.nguyen@local.test` / `ChangeMe123!`.
2. Cuộn tới **Yêu cầu bảo trì**, nhấn **Báo sự cố**.
3. Chọn thiết bị `DEMO-LT-001 · Laptop Dell Latitude`.
4. Kiểm tra form hiển thị serial, loại thiết bị, hãng/model và hạn bảo hành.
5. Chọn nhóm **Phần cứng**, ưu tiên **Cao**.
6. Nhập tiêu đề `Bàn phím không nhận phím` và mô tả chi tiết rồi gửi.

Kết quả đúng:

- Yêu cầu mới có trạng thái **Chờ tiếp nhận**.
- Thẻ thiết bị `DEMO-LT-001` có nhãn **Có yêu cầu bảo trì**.
- Tạo thêm yêu cầu cho cùng thiết bị bị chặn vì chỉ được có một yêu cầu đang mở.
- Không xuất hiện nút quản trị trên portal nhân viên.

## 3. Admin/IT tiếp nhận

1. Đăng xuất và đăng nhập `admin@local.test` / `ChangeMe123!` hoặc `it.manager@local.test` / `ChangeMe123!`.
2. Chọn menu **Bảo trì**.
3. Tìm bằng `DEMO-LT-001`, serial hoặc tên nhân viên.
4. Xác nhận thẻ yêu cầu hiển thị đủ snapshot thiết bị và nội dung sự cố.
5. Nhấn **Tiếp nhận**.

Kết quả đúng:

- Trạng thái đổi thành **Đang xử lý**.
- Người tiếp nhận được lưu là người phụ trách.
- Nút còn lại là **Hoàn thành** và **Từ chối**.
- Employee nhận thông báo yêu cầu đã được tiếp nhận.

## 4. Hoàn thành yêu cầu

1. Ở yêu cầu đang xử lý, nhấn **Hoàn thành**.
2. Nhập `Đã vệ sinh và thay bàn phím, thiết bị hoạt động bình thường.` rồi xác nhận.
3. Đăng nhập lại Employee và mở chuông thông báo.

Kết quả đúng:

- Yêu cầu có trạng thái **Hoàn thành** và hiển thị phản hồi.
- Không còn nút chuyển trạng thái.
- Employee nhận thông báo cùng nội dung kết quả.
- Nhãn **Có yêu cầu bảo trì** trên thiết bị biến mất.
- Employee có thể tạo một yêu cầu mới cho thiết bị sau khi yêu cầu cũ kết thúc.

## 5. Kiểm tra từ chối và hủy

- Tạo một yêu cầu khác, Admin nhấn **Từ chối**, nhập lý do không được bỏ trống. Employee phải nhận thông báo và thấy trạng thái **Đã từ chối**.
- Tạo một yêu cầu mới rồi Employee nhấn **Hủy yêu cầu** trước khi IT tiếp nhận. Trạng thái phải thành **Đã hủy**.
- Employee không được hủy yêu cầu đã **Đang xử lý**.
- Yêu cầu đã hoàn thành, từ chối hoặc hủy không được xử lý lần hai.

## 6. Kiểm tra PostgreSQL thật

Từ thư mục `backend`, dùng lại các biến môi trường PostgreSQL đã cấu hình:

```powershell
$env:STORAGE_DRIVER = "postgres"
go run ./cmd/migrate up
go run ./cmd/migrate status
```

Kết quả cần có migration `005_maintenance_requests.sql` ở trạng thái đã áp dụng và schema version là `5`. Sau đó chạy backend và lặp lại các bước giao diện ở trên. Dừng rồi chạy lại backend; lịch sử yêu cầu và thông báo vẫn phải còn.

Nếu đã đặt `TEST_DATABASE_URL`, chạy riêng các test PostgreSQL:

```powershell
go test ./internal/modules/maintenancerequests -run Postgres -count=1 -v
```

Kết quả mong đợi gồm test lưu/search/chuyển trạng thái, rollback khi tạo thông báo lỗi và chống hai Admin tiếp nhận đồng thời.

## 7. Kiểm tra tự động

```powershell
Set-Location backend
go test ./... -count=1
go vet ./...

Set-Location ..\web
npm.cmd test
npm.cmd run lint
npm.cmd run build
```
