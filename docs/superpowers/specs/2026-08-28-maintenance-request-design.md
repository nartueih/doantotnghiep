# Thiết kế module yêu cầu bảo trì thiết bị

## 1. Mục tiêu

Xây dựng một module độc lập cho phép Employee gửi yêu cầu bảo trì đối với thiết bị đang được giao, theo dõi tiến độ và nhận phản hồi ngay trong website. Admin hoặc IT Manager có thể tìm kiếm, tiếp nhận, hoàn thành hoặc từ chối yêu cầu. Toàn bộ dữ liệu phải được lưu trong PostgreSQL và API phải sẵn sàng để ứng dụng Android sử dụng lại sau này.

Module không thay đổi quan hệ cấp phát thiết bị hiện tại. Một thiết bị vẫn thuộc Employee trong suốt quá trình bảo trì; giao diện hiển thị thêm nhãn `Đang bảo trì` dựa trên yêu cầu đang mở.

## 2. Phạm vi

### Trong phạm vi

- Employee tạo yêu cầu cho thiết bị đang được giao cho chính mình.
- Employee xem lịch sử và hủy yêu cầu đang chờ tiếp nhận.
- Admin/IT Manager xem, tìm kiếm, lọc, tiếp nhận, hoàn thành hoặc từ chối yêu cầu.
- Hiển thị đầy đủ thông tin thiết bị, bao gồm mã tài sản, serial, tên, loại, hãng, model, ngày mua và hạn bảo hành.
- Gửi notification website khi yêu cầu được tiếp nhận, hoàn thành hoặc từ chối.
- Ghi audit log cho tạo, hủy và mọi lần chuyển trạng thái.
- Hỗ trợ memory repository cho kiểm thử nhanh và PostgreSQL repository cho dữ liệu chính thức.
- Cập nhật OpenAPI và tài liệu kiểm thử thủ công.

### Ngoài phạm vi phiên bản đầu

- Tải ảnh, video hoặc file đính kèm.
- Push notification trên Android.
- Bình luận nhiều lượt hoặc hội thoại trong một yêu cầu.
- Tự động phân công theo ca trực hoặc nhóm kỹ thuật.
- Chi phí sửa chữa, kho linh kiện và nhà cung cấp bảo hành.
- Tự động thay đổi trường `devices.status` hoặc gỡ thiết bị khỏi Employee.

## 3. Vai trò và quyền

### Employee

- Chỉ xem và tạo yêu cầu cho thiết bị có `assigned_user_id` bằng ID trong access token.
- Chỉ xem yêu cầu do chính mình tạo.
- Chỉ hủy yêu cầu của mình khi trạng thái là `pending`.
- Không được tự tiếp nhận, hoàn thành hoặc từ chối.

### Admin và IT Manager

- Xem toàn bộ yêu cầu.
- Tìm kiếm và lọc danh sách.
- Tiếp nhận yêu cầu `pending`.
- Hoàn thành yêu cầu `in_progress`.
- Từ chối yêu cầu `pending` hoặc `in_progress`.
- Người bấm tiếp nhận được lưu là người phụ trách. Admin/IT Manager khác vẫn được phép cập nhật yêu cầu để tránh yêu cầu bị kẹt khi người phụ trách vắng mặt.

Backend luôn xác minh quyền; giao diện ẩn nút không thay thế kiểm tra phân quyền tại API.

## 4. Mô hình dữ liệu

Tạo bảng `maintenance_requests` trong migration phiên bản `005` và tăng `migrations.LatestVersion` lên `5`.

### Trường định danh và người yêu cầu

- `id UUID PRIMARY KEY`
- `requester_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT`
- `requester_name VARCHAR(150) NOT NULL`
- `device_id UUID NOT NULL REFERENCES devices(id) ON DELETE RESTRICT`

### Bản chụp thông tin thiết bị

- `device_asset_code VARCHAR(80) NOT NULL`
- `device_serial_number VARCHAR(150)`
- `device_name VARCHAR(150) NOT NULL`
- `device_type VARCHAR(80) NOT NULL`
- `device_manufacturer VARCHAR(150)`
- `device_model VARCHAR(150)`
- `device_purchased_at DATE`
- `device_warranty_expires_at DATE`

Các trường này là snapshot tại thời điểm tạo yêu cầu. Việc chỉnh sửa thiết bị sau đó không làm thay đổi lịch sử bảo trì.

### Nội dung sự cố

- `category VARCHAR(30) NOT NULL`
- `priority VARCHAR(20) NOT NULL`
- `title VARCHAR(200) NOT NULL`
- `description TEXT NOT NULL`

Giá trị hợp lệ:

- `category`: `hardware`, `software`, `network`, `accessory`, `other`.
- `priority`: `normal`, `high`, `urgent`.

Tiêu đề và mô tả được trim và không được để trống.

### Trạng thái và xử lý

- `status VARCHAR(20) NOT NULL`
- `assigned_to UUID REFERENCES users(id) ON DELETE RESTRICT`
- `assigned_to_name VARCHAR(150)`
- `last_actor_id UUID REFERENCES users(id) ON DELETE RESTRICT`
- `last_actor_name VARCHAR(150)`
- `response_note TEXT`
- `created_at TIMESTAMPTZ NOT NULL`
- `updated_at TIMESTAMPTZ NOT NULL`
- `accepted_at TIMESTAMPTZ`
- `completed_at TIMESTAMPTZ`
- `rejected_at TIMESTAMPTZ`
- `cancelled_at TIMESTAMPTZ`

Giá trị `status`: `pending`, `in_progress`, `completed`, `rejected`, `cancelled`.

Constraint trạng thái bảo đảm:

- `pending`: chưa có người phụ trách và chưa có mốc kết thúc.
- `in_progress`: có người phụ trách và `accepted_at`.
- `completed`: có người phụ trách, `accepted_at`, `completed_at` và `response_note` không trống.
- `rejected`: có người thao tác, `rejected_at` và `response_note` không trống; có thể có hoặc chưa có người phụ trách tùy yêu cầu bị từ chối trước hay sau khi tiếp nhận.
- `cancelled`: có `cancelled_at`, không có người phụ trách và không có mốc xử lý khác.

Tạo unique partial index trên `device_id` khi `status IN ('pending', 'in_progress')` để mỗi thiết bị chỉ có tối đa một yêu cầu đang mở. Bổ sung index cho requester, status, priority, category và thời gian tạo.

## 5. Vòng đời yêu cầu

```text
pending --accept--> in_progress --complete--> completed
   |                    |
   |                    +--reject-----------> rejected
   +--reject-------------------------------> rejected
   +--cancel-------------------------------> cancelled
```

- Employee chỉ được `cancel` từ `pending`.
- Admin/IT Manager chỉ được `accept` từ `pending`.
- Admin/IT Manager chỉ được `complete` từ `in_progress`.
- Admin/IT Manager được `reject` từ `pending` hoặc `in_progress`.
- Mọi trạng thái kết thúc là bất biến.
- `complete` và `reject` bắt buộc có ghi chú phản hồi.
- Các cập nhật trạng thái dùng conditional update và kiểm tra affected rows để ngăn hai thao tác đồng thời cùng thành công.

## 6. API

### Employee API

#### `GET /api/v1/me/maintenance-requests`

Trả về yêu cầu của Employee hiện tại, sắp xếp mới nhất trước. Response gồm `items` và `open_count`.

#### `POST /api/v1/me/maintenance-requests`

Request:

```json
{
  "device_id": "uuid",
  "category": "hardware",
  "priority": "high",
  "title": "Máy không khởi động",
  "description": "Thiết bị không phản hồi khi nhấn nút nguồn."
}
```

Backend tìm thiết bị bằng `device_id`, kiểm tra thiết bị đang được giao cho Employee hiện tại và tự tạo snapshot. Thành công trả `201 Created`.

#### `POST /api/v1/me/maintenance-requests/{id}/cancel`

Chỉ hủy yêu cầu `pending` thuộc Employee hiện tại. Thành công trả yêu cầu đã cập nhật.

### Admin/IT Manager API

#### `GET /api/v1/maintenance-requests`

Hỗ trợ query `status`, `priority`, `category`, `search`. `search` tìm theo tên Employee, mã tài sản, serial, tên thiết bị và tiêu đề sự cố. Kết quả sắp xếp theo thời gian tạo giảm dần.

#### `POST /api/v1/maintenance-requests/{id}/accept`

Không cần request body. Lưu người thao tác làm người phụ trách, chuyển sang `in_progress`, tạo notification tiếp nhận trong cùng transaction.

#### `POST /api/v1/maintenance-requests/{id}/complete`

Request bắt buộc có `response_note`. Chuyển từ `in_progress` sang `completed` và tạo notification hoàn thành trong cùng transaction.

#### `POST /api/v1/maintenance-requests/{id}/reject`

Request bắt buộc có `response_note`. Chuyển từ `pending` hoặc `in_progress` sang `rejected` và tạo notification từ chối trong cùng transaction.

### Mã lỗi

- `400 Bad Request`: trường bắt buộc thiếu hoặc enum không hợp lệ.
- `401 Unauthorized`: thiếu hoặc access token không hợp lệ.
- `403 Forbidden`: vai trò không có quyền gọi endpoint.
- `404 Not Found`: thiết bị/yêu cầu không tồn tại hoặc Employee truy cập tài nguyên không thuộc mình.
- `409 Conflict`: đã có yêu cầu mở cho thiết bị hoặc trạng thái đã thay đổi.

## 7. Kiến trúc backend

Tạo package `backend/internal/modules/maintenancerequests` với các thành phần tách biệt:

- `model.go`: entity, input, filter, enum và lỗi domain.
- `repository.go`: interface lưu trữ.
- `memory_repository.go`: triển khai memory có mutex.
- `postgres_repository.go`: câu lệnh SQL tham số hóa, conditional update và row lock.
- `service.go`: validation, kiểm tra quyền sở hữu thiết bị và workflow transaction.
- `http.go`: binding request, phân quyền và ánh xạ lỗi HTTP.

Service phụ thuộc vào interface nhỏ cho user/device lookup, notification creator và transaction manager. Memory dùng `database.DirectTransactor`; PostgreSQL dùng `database.NewPostgresTransactor`.

Các thao tác `accept`, `complete` và `reject` khóa yêu cầu bằng `SELECT ... FOR UPDATE`, cập nhật trạng thái và tạo notification trong một transaction. Nếu notification thất bại, thay đổi trạng thái phải rollback.

Mở rộng notification với:

- `maintenance_accepted`
- `maintenance_completed`
- `maintenance_rejected`
- entity type `maintenance_request`

Mở rộng audit bằng entity type `maintenance_request`. Tái sử dụng action `request`, `cancel` và `reject`; bổ sung action `accept` và `complete`. Audit không chứa thông tin nhạy cảm hoặc toàn bộ mô tả sự cố.

## 8. Giao diện web

### Employee Portal

- Thêm khu vực **Bảo trì thiết bị**.
- Mỗi thiết bị được giao hiển thị nhãn `Đang bảo trì` khi có yêu cầu mở và nút **Yêu cầu bảo trì** khi chưa có yêu cầu mở.
- Modal tạo yêu cầu hiển thị đầy đủ thông tin thiết bị ở chế độ chỉ đọc.
- Form chỉ nhập loại sự cố, mức độ, tiêu đề và mô tả; không có dấu sao đỏ, placeholder và validation nói rõ trường không được bỏ trống.
- Lịch sử yêu cầu hiển thị trạng thái, người phụ trách, phản hồi, thời gian tạo/tiếp nhận/hoàn thành.
- Chỉ yêu cầu `pending` có nút **Hủy yêu cầu**.

### Admin/IT Manager

- Thêm menu **Bảo trì** trong sidebar.
- Trang danh sách có thống kê số chờ tiếp nhận, đang xử lý, hoàn thành và khẩn cấp.
- Có bộ lọc trạng thái, mức độ, loại sự cố và ô tìm kiếm.
- Bảng/card hiển thị Employee, thiết bị, serial, sự cố, mức độ, trạng thái, người phụ trách và thời gian.
- Panel chi tiết hiển thị toàn bộ snapshot thiết bị, nội dung sự cố và lịch sử xử lý.
- Nút hành động chỉ xuất hiện theo trạng thái hợp lệ.
- `complete` và `reject` mở form bắt buộc nhập ghi chú phản hồi.

Giao diện tuân theo typography, màu trạng thái, modal và responsive pattern hiện có. Mọi màn hình phải có loading, empty state, API error và session-expired state.

## 9. Thông báo và hiển thị nhãn bảo trì

- Khi `accept`, Employee nhận thông báo yêu cầu đã được tiếp nhận và tên người phụ trách.
- Khi `complete`, Employee nhận ghi chú kết quả sửa chữa.
- Khi `reject`, Employee nhận lý do từ chối.
- Thay đổi dữ liệu không đổi trạng thái không tạo notification.
- Danh sách thiết bị không thay đổi `devices.status`. Web lấy yêu cầu `pending`/`in_progress` của user để tính nhãn `Đang bảo trì`.

## 10. Kiểm thử

### Unit và service

- Validation category, priority, title và description.
- Chỉ tạo yêu cầu cho thiết bị thuộc Employee và user đang active.
- Chặn yêu cầu đang mở trùng thiết bị.
- Kiểm tra đầy đủ chuyển trạng thái hợp lệ/không hợp lệ.
- Employee chỉ hủy yêu cầu của mình khi `pending`.
- Người tiếp nhận và các timestamp được lưu chính xác.
- Notification được tạo đúng user, đúng loại và không chứa dữ liệu ngoài phạm vi.

### PostgreSQL integration

- Migration 005 áp dụng một lần và có constraint/index đúng.
- Hai goroutine tạo yêu cầu cùng thiết bị: đúng một thành công, một nhận conflict.
- Hai Admin cùng tiếp nhận/hoàn thành: đúng một thao tác thành công.
- Notification lỗi làm rollback chuyển trạng thái.
- Dữ liệu, snapshot và notification tồn tại sau khi đóng/mở lại pool.

### HTTP và phân quyền

- `401`, `403`, `404`, `409` được ánh xạ đúng.
- Employee không xem hoặc hủy yêu cầu của người khác.
- Admin/IT có route quản trị; Employee không có quyền.
- OpenAPI chứa đầy đủ route, schema, enum và role requirement.

### Web

- View-model ánh xạ nhãn category, priority và status ổn định.
- Nút hành động xuất hiện đúng trạng thái.
- Badge bảo trì được tính đúng theo yêu cầu mở.
- Test, lint và production build đều thành công.

## 11. Tiêu chí hoàn thành

- Employee tạo, xem và hủy yêu cầu hợp lệ trên web.
- Admin/IT tìm kiếm, lọc, tiếp nhận, hoàn thành và từ chối trên web.
- Trang bảo trì hiển thị đầy đủ serial và thông tin thiết bị đã thống nhất.
- Một thiết bị không thể có hai yêu cầu đang mở, kể cả khi có request đồng thời.
- Workflow và notification được commit nguyên tử trong PostgreSQL.
- Thiết bị vẫn được gán cho Employee và không bị đổi `devices.status` bởi workflow bảo trì.
- Audit log ghi nhận mọi thao tác quan trọng.
- Memory và PostgreSQL đều hỗ trợ module.
- Backend test/vet và web test/lint/build đều đạt.
- OpenAPI và tài liệu kiểm thử thủ công được cập nhật để Android dùng lại API sau này.
