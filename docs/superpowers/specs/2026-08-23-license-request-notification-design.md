# Thiết kế yêu cầu cấp license và thông báo website

## 1. Mục tiêu

Cho phép nhân viên tự gửi yêu cầu sử dụng phần mềm, theo dõi quá trình xử lý và nhận phản hồi ngay trong website. Admin hoặc IT Manager có thể duyệt yêu cầu bằng một license còn seat hoặc từ chối với lý do rõ ràng. Khi duyệt thành công, hệ thống tự tạo một bản ghi cấp phát license cho nhân viên.

Giai đoạn này chạy và kiểm thử với `STORAGE_DRIVER=memory`, giống các module đã triển khai trước. Dữ liệu yêu cầu và thông báo sẽ mất khi backend khởi động lại. PostgreSQL được tách thành giai đoạn sau.

## 2. Phạm vi

### Trong phạm vi

- Nhân viên tạo, xem và hủy yêu cầu cấp license của chính mình.
- Admin và IT Manager xem, lọc, duyệt hoặc từ chối yêu cầu.
- Duyệt yêu cầu tự động tạo license assignment cho nhân viên.
- Từ chối hỗ trợ các lý do có cấu trúc và phản hồi bắt buộc.
- Nhân viên nhận thông báo website khi yêu cầu được duyệt hoặc từ chối.
- Nhân viên xem số thông báo chưa đọc, đọc từng thông báo hoặc đánh dấu tất cả đã đọc.
- Ghi audit log cho thao tác tạo, hủy, duyệt và từ chối yêu cầu.
- Tài liệu OpenAPI và kịch bản kiểm thử PowerShell.

### Ngoài phạm vi

- Email, SMS và push notification.
- PostgreSQL migration, PostgreSQL repository và kiểm thử transaction database.
- Tự động mua thêm license hoặc tự động duyệt lại yêu cầu đã bị từ chối.
- Quy trình phê duyệt nhiều cấp.
- Yêu cầu bảo trì thiết bị; module này sẽ được triển khai sau và có thể tái sử dụng notification.

## 3. Vai trò và quyền

### Employee

- Xem danh mục phần mềm có thể yêu cầu.
- Tạo yêu cầu cho bản thân; user ID luôn lấy từ access token.
- Chỉ xem yêu cầu và thông báo của chính mình.
- Chỉ hủy yêu cầu đang ở trạng thái `pending`.
- Không được chọn license cụ thể và không được tự duyệt yêu cầu.

### Admin và IT Manager

- Xem toàn bộ yêu cầu.
- Lọc theo trạng thái, mức ưu tiên và tìm kiếm theo nhân viên hoặc phần mềm.
- Duyệt yêu cầu bằng một license hợp lệ còn seat.
- Từ chối yêu cầu với lý do và phản hồi.

## 4. Mô hình nghiệp vụ

### License request

Một yêu cầu gồm các trường chính:

- `id`
- `requester_id`, `requester_name`
- `software_product_id`, `software_product_name`
- `priority`: `normal`, `high`, `urgent`
- `reason`
- `status`: `pending`, `approved`, `rejected`, `cancelled`
- `selected_license_id`, `selected_license_name` khi đã duyệt
- `assignment_id` khi đã duyệt
- `reviewed_by`, `reviewed_by_name`
- `decision_reason`: `out_of_stock`, `not_approved`, `other` khi bị từ chối
- `response_note`
- `created_at`, `updated_at`, `reviewed_at`, `cancelled_at`

Chuyển trạng thái hợp lệ:

- `pending -> approved`
- `pending -> rejected`
- `pending -> cancelled`

Trạng thái kết thúc không được thay đổi. Sau khi một yêu cầu bị `rejected` hoặc `cancelled`, nhân viên được tạo yêu cầu mới cho cùng phần mềm. Hệ thống không cho một nhân viên có nhiều hơn một yêu cầu `pending` cho cùng sản phẩm phần mềm.

### Notification

Một thông báo gồm:

- `id`
- `user_id`
- `type`: `license_request_approved` hoặc `license_request_rejected`
- `title`, `message`
- `entity_type`: `license_request`
- `entity_id`
- `created_at`
- `read_at`, để trống khi chưa đọc

Thông báo được lưu trong memory trong suốt vòng đời tiến trình backend. Thông báo không chứa license key.

## 5. Kiến trúc backend

### Module `licenserequests`

Module gồm model, repository interface, memory repository, service, HTTP handler và tests. Service phụ thuộc vào:

- software finder để kiểm tra sản phẩm tồn tại;
- license finder để kiểm tra license được chọn thuộc đúng sản phẩm;
- assignment service để tái sử dụng toàn bộ quy tắc cấp phát hiện có;
- user finder để bổ sung thông tin người yêu cầu và người xử lý;
- notification writer để tạo phản hồi cho nhân viên.

Các thao tác thay đổi trạng thái và kiểm tra yêu cầu trùng được tuần tự hóa trong service khi chạy memory. Repository vẫn tự bảo vệ dữ liệu bằng mutex. Cách này ngăn hai request đồng thời duyệt cùng một yêu cầu trong một tiến trình.

Luồng duyệt:

1. Khóa vùng chuyển trạng thái.
2. Đọc lại yêu cầu và xác nhận trạng thái là `pending`.
3. Kiểm tra license thuộc đúng sản phẩm được yêu cầu.
4. Gọi assignment service với `user_id` của người yêu cầu.
5. Assignment service kiểm tra ngày hiệu lực, loại cấp phát, người dùng, cấp trùng và seat; memory assignment repository giữ chỗ seat an toàn.
6. Cập nhật yêu cầu thành `approved` và lưu assignment ID.
7. Tạo thông báo duyệt thành công cho nhân viên.

Nếu bước cấp phát thất bại do hết seat, license không hợp lệ hoặc cấp trùng, yêu cầu vẫn là `pending`. Admin có thể chọn thao tác từ chối với lý do `out_of_stock` và nhập phản hồi.

Luồng từ chối kiểm tra yêu cầu vẫn `pending`, yêu cầu `decision_reason` hợp lệ và `response_note` không rỗng, sau đó cập nhật `rejected` và tạo thông báo. Luồng hủy kiểm tra quyền sở hữu và chỉ cập nhật yêu cầu đang `pending`.

### Module `notifications`

Module gồm model, repository interface, memory repository, service, HTTP handler và tests. Mọi thao tác đọc hoặc cập nhật đều nhận user ID từ access token. Repository không trả về hay cập nhật thông báo thuộc người khác.

### Audit

HTTP handler ghi các action mới sau khi nghiệp vụ thành công:

- `request` cho thao tác gửi yêu cầu;
- `cancel` cho thao tác nhân viên hủy;
- `approve` cho thao tác duyệt;
- `reject` cho thao tác từ chối;
- entity type là `license_request`.

Metadata chỉ chứa ID, trạng thái, lý do quyết định và các liên kết nghiệp vụ; không chứa license key hoặc dữ liệu nhạy cảm.

## 6. API

### Employee API

- `GET /api/v1/me/requestable-software`
  - Trả danh sách sản phẩm phần mềm để tạo yêu cầu.
- `GET /api/v1/me/license-requests`
  - Trả lịch sử yêu cầu của user hiện tại, mới nhất trước.
- `POST /api/v1/me/license-requests`
  - Body: `software_product_id`, `priority`, `reason`.
- `PATCH /api/v1/me/license-requests/:id/cancel`
  - Chỉ chủ sở hữu được hủy yêu cầu `pending`.
- `GET /api/v1/me/notifications`
  - Trả danh sách thông báo của user hiện tại và `unread_count`.
- `PATCH /api/v1/me/notifications/:id/read`
  - Đánh dấu một thông báo của user hiện tại là đã đọc; thao tác lặp lại vẫn thành công.
- `PATCH /api/v1/me/notifications/read-all`
  - Đánh dấu toàn bộ thông báo của user hiện tại là đã đọc.

### Admin/IT API

- `GET /api/v1/license-requests`
  - Query tùy chọn: `status`, `priority`, `search`.
- `PATCH /api/v1/license-requests/:id/approve`
  - Body: `license_id`, `response_note` tùy chọn.
- `PATCH /api/v1/license-requests/:id/reject`
  - Body: `decision_reason`, `response_note`; cả hai bắt buộc.

### Mã lỗi

- `400 Bad Request`: JSON hoặc trường dữ liệu không hợp lệ.
- `403 Forbidden`: vai trò không được phép.
- `404 Not Found`: tài nguyên không tồn tại hoặc nhân viên truy cập tài nguyên không thuộc mình.
- `409 Conflict`: yêu cầu pending trùng, chuyển trạng thái không hợp lệ, license hết seat, hết hạn, đã lưu trữ hoặc đã cấp trùng.
- `422 Unprocessable Entity`: license không thuộc sản phẩm được yêu cầu hoặc không hỗ trợ cấp cho user.

## 7. Giao diện web

### Employee Portal

- Thêm khu vực `Yêu cầu license` và nút `Tạo yêu cầu`.
- Form gồm phần mềm, mức ưu tiên và lý do; placeholder của trường bắt buộc nêu rõ không được bỏ trống, không dùng dấu sao đỏ.
- Hiển thị lịch sử yêu cầu cùng trạng thái, thời gian và phản hồi của Admin/IT.
- Chỉ hiển thị nút hủy cho yêu cầu `pending` và yêu cầu xác nhận trước khi hủy.
- Biểu tượng chuông hiển thị badge số chưa đọc và mở panel danh sách thông báo.
- Hỗ trợ đánh dấu một hoặc tất cả thông báo là đã đọc.

### Admin/IT Portal

- Thêm trang `Yêu cầu` trong sidebar và hash route riêng.
- Hiển thị bảng yêu cầu với tìm kiếm, lọc trạng thái và mức ưu tiên.
- Modal duyệt chỉ hiển thị license thuộc đúng sản phẩm, đang hiệu lực, hỗ trợ user và còn seat.
- Modal từ chối có lựa chọn `Tạm hết license`, `Không được phê duyệt`, `Khác` và bắt buộc nhập phản hồi.
- Nếu duyệt thất bại vì hết seat, giao diện giữ nguyên yêu cầu pending và hiển thị hướng dẫn dùng thao tác `Tạm hết license`.
- Giao diện tuân theo cỡ chữ và responsive layout hiện tại.

## 8. Kiểm thử

### Backend

- Tạo và liệt kê yêu cầu của đúng user.
- Validate phần mềm, mức ưu tiên và lý do.
- Chặn hai yêu cầu pending cùng user và phần mềm, kể cả khi gọi đồng thời.
- Chặn truy cập hoặc hủy yêu cầu của user khác.
- Chặn hủy, duyệt hoặc từ chối yêu cầu đã kết thúc.
- Duyệt thành công tạo đúng một active assignment cho requester.
- Chặn license sai sản phẩm, sai assignment type, không hoạt động, hết seat hoặc cấp trùng.
- Hai thao tác duyệt đồng thời chỉ có một thao tác thành công.
- Từ chối bắt buộc lý do và phản hồi.
- Duyệt/từ chối tạo đúng notification; thao tác đọc chỉ ảnh hưởng notification của user hiện tại.
- HTTP tests xác nhận authentication, role authorization, response body và status code.
- Audit tests xác nhận action và entity phù hợp, không có license key trong metadata.

### Web

- Chạy Oxlint và TypeScript/Vite production build.
- Kiểm thử thủ công bằng tài khoản Employee: tạo, xem, hủy, nhận và đọc thông báo.
- Kiểm thử thủ công bằng Admin/IT: lọc, duyệt, từ chối vì tạm hết license và xem assignment được tạo.
- Kiểm tra phiên hết hạn, lỗi mạng, loading, empty state và responsive layout.

## 9. Tiêu chí hoàn thành

- Toàn bộ backend tests hiện có và tests mới đều pass với memory storage.
- Web lint và production build pass.
- Nhân viên hoàn thành được luồng tạo yêu cầu đến khi nhận phản hồi mà không cần nhập user ID.
- Admin/IT duyệt thành công chỉ khi license hợp lệ và còn seat.
- Duyệt tạo đúng một assignment; từ chối không tạo assignment.
- Thông báo và audit log được tạo đúng, không làm lộ license key.
- OpenAPI và tài liệu kiểm thử phản ánh đúng các endpoint mới.
