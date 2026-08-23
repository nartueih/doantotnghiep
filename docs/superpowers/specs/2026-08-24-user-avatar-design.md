# Thiết kế quản lý ảnh cá nhân người dùng

Ngày: 2026-08-24
Trạng thái: Đã được duyệt

## 1. Mục tiêu

Cho phép quản trị viên gắn ảnh cá nhân tùy chọn cho tài khoản người dùng để dễ nhận diện trong trang quản lý và cổng nhân viên. Quản trị viên có thể tải lên, thay thế hoặc xóa ảnh. Nhân viên chỉ được xem ảnh của chính mình và không được tự chỉnh sửa.

Giai đoạn này tiếp tục sử dụng storage memory. Ảnh và dữ liệu người dùng sẽ mất khi backend dừng. Thiết kế tách kho ảnh khỏi module người dùng để có thể thay memory bằng filesystem hoặc object storage khi triển khai PostgreSQL mà không đổi hợp đồng frontend/mobile.

## 2. Phạm vi

### Bao gồm

- Tải ảnh từ máy khi tạo tài khoản mới.
- Tải, thay thế và xóa ảnh của tài khoản đã tồn tại.
- Hiển thị ảnh trong danh sách người dùng và cổng nhân viên.
- Avatar chữ cái khi người dùng chưa có ảnh hoặc tải ảnh thất bại.
- Phân quyền và kiểm tra nội dung tệp tại backend.
- Audit log cho thao tác tải/thay/xóa ảnh.
- OpenAPI và tài liệu kiểm thử thủ công.

### Chưa bao gồm

- Nhân viên tự cập nhật ảnh.
- Cắt, xoay hoặc chỉnh sửa ảnh.
- Đồng bộ ảnh từ Google Workspace, Microsoft Entra ID hoặc dịch vụ bên ngoài.
- Lưu ảnh bền vững trên PostgreSQL, filesystem hoặc object storage.
- Nén hoặc tạo nhiều kích thước thumbnail ở backend.

## 3. Kiến trúc backend

Tạo module `backend/internal/modules/avatars` với các thành phần độc lập:

- `Image`: dữ liệu ảnh gồm `UserID`, `ContentType`, `Data` và `UpdatedAt`.
- `Store`: hợp đồng `Get`, `Put` và `Delete` theo `user_id`.
- `MemoryStore`: map an toàn cho truy cập đồng thời, sao chép byte khi đọc/ghi để không làm lộ vùng nhớ nội bộ.
- `Service`: xác minh user tồn tại, phân quyền nghiệp vụ và điều phối kho ảnh.
- `HTTPHandler`: đọc multipart, giới hạn dung lượng, xác minh MIME và trả ảnh nhị phân.

Module avatar chỉ phụ thuộc vào một interface tìm user tối thiểu, không phụ thuộc trực tiếp vào repository cụ thể. `main.go` khởi tạo một `MemoryStore` và truyền cùng instance cho handler trong suốt vòng đời server.

Ảnh không được nhúng vào `auth.User`. Cách này giữ cho response đăng nhập và danh sách người dùng nhỏ, đồng thời tránh đưa Base64 vào JSON.

## 4. API và phân quyền

Tất cả endpoint yêu cầu Bearer access token.

### `PUT /api/v1/users/{id}/avatar`

- Chỉ role `admin`.
- Body là `multipart/form-data`, trường tệp tên `avatar`.
- Tạo ảnh mới hoặc ghi đè ảnh hiện tại.
- Thành công trả `204 No Content`.

### `GET /api/v1/users/{id}/avatar`

- `admin` và `it_manager` được đọc ảnh của mọi user.
- `employee` chỉ được đọc ảnh khi `{id}` trùng user trong access token.
- Trả byte ảnh với `Content-Type` thực tế.
- Trả `Cache-Control: private, no-store` để tránh ảnh cũ sau khi thay thế và hạn chế lưu dữ liệu cá nhân ngoài ý muốn.

### `DELETE /api/v1/users/{id}/avatar`

- Chỉ role `admin`.
- Xóa ảnh hiện có và trả `204 No Content`.

### Quy tắc chống dò dữ liệu

Với nhân viên yêu cầu avatar của user khác, backend trả `403 Forbidden` trước khi tiết lộ user hoặc ảnh có tồn tại hay không.

## 5. Kiểm tra tệp và lỗi

- Chỉ nhận JPEG và PNG.
- Dung lượng tối đa là `2 * 1024 * 1024` byte.
- Handler giới hạn request trước khi parse multipart để không đọc quá nhiều dữ liệu vào RAM.
- Backend xác minh nội dung bằng `http.DetectContentType`; không tin tên file hoặc header do client gửi.
- SVG không được chấp nhận để tránh nội dung chủ động.
- Không lưu tên file gốc và không dùng tên file vào đường dẫn hệ thống.

Mã lỗi:

- `400 Bad Request`: thiếu trường `avatar` hoặc multipart không hợp lệ.
- `401 Unauthorized`: không có token hoặc token không hợp lệ.
- `403 Forbidden`: không đủ quyền.
- `404 Not Found`: user không tồn tại hoặc avatar chưa tồn tại.
- `413 Payload Too Large`: tệp vượt quá 2 MB.
- `415 Unsupported Media Type`: nội dung không phải JPEG/PNG.
- `500 Internal Server Error`: lỗi kho ảnh không dự kiến.

Response lỗi tiếp tục dùng JSON `{ "error": "..." }` theo quy ước hiện tại. Response lỗi không chứa dữ liệu ảnh hoặc thông tin nội bộ.

## 6. Audit log

Mỗi thao tác thành công của Admin ghi audit vào entity user tương ứng:

- tải lần đầu hoặc thay ảnh: action `update`, metadata `field: "avatar"`, `content_type` và `size_bytes`;
- xóa ảnh: action `delete`, metadata `field: "avatar"`.

Không ghi byte ảnh, Base64 hoặc tên file gốc vào audit log.

## 7. Luồng frontend quản trị

Form “Thêm người dùng” có vùng “Ảnh cá nhân (không bắt buộc)” ở đầu phần thông tin nhân sự:

- input chỉ nhận `.jpg`, `.jpeg`, `.png`;
- kiểm tra dung lượng 2 MB trước khi submit;
- xem trước ảnh bằng Object URL;
- cho phép bỏ ảnh đã chọn trước khi tạo.

Quy trình lưu gồm hai bước:

1. Gọi API tạo user JSON hiện có.
2. Nếu Admin đã chọn ảnh, gọi tiếp API upload bằng `FormData` với ID user vừa tạo.

Nếu bước 1 thất bại, không upload ảnh. Nếu bước 1 thành công nhưng bước 2 thất bại, không rollback tài khoản vì ảnh là tùy chọn; dialog đóng và hiển thị cảnh báo “Đã tạo tài khoản nhưng chưa tải được ảnh”. Admin có thể tải lại từ danh sách.

Trong danh sách người dùng:

- frontend gọi API avatar có Authorization và chuyển Blob thành Object URL;
- khi không có ảnh hoặc request lỗi, hiển thị avatar chữ cái hiện tại;
- action của mỗi user có “Thay ảnh” và “Xóa ảnh”; “Xóa ảnh” chỉ hiện khi frontend đã tải được ảnh;
- thao tác thay ảnh có xem trước và validation giống form tạo.

Object URL phải được `URL.revokeObjectURL` khi ảnh thay đổi hoặc component unmount.

## 8. Luồng cổng nhân viên

Sau khi đăng nhập, frontend dùng `session.user.id` để gọi `GET /users/{id}/avatar` bằng access token. Ảnh xuất hiện tại khu vực tài khoản/cổng nhân viên. Không có nút tải lên, thay thế hoặc xóa.

Nếu avatar trả `404`, `403`, lỗi mạng hoặc dữ liệu ảnh không hiển thị được, giao diện dùng avatar chữ cái và các chức năng khác vẫn hoạt động bình thường.

Hợp đồng HTTP này dùng được cho app Kotlin sau này: app tải byte ảnh bằng Bearer token và hiển thị bằng thư viện ảnh của Android mà không cần xử lý Data URL.

## 9. Kiểm thử

### Backend

- MemoryStore lưu, đọc bản sao, ghi đè và xóa ảnh đúng user.
- Admin upload JPEG/PNG hợp lệ.
- Admin thay thế ảnh cũ và đọc được dữ liệu mới.
- Admin xóa ảnh; lần đọc sau trả 404.
- IT Manager và Employee không thể upload/xóa.
- IT Manager đọc được avatar user bất kỳ.
- Employee đọc được avatar chính mình nhưng không đọc được user khác.
- Tệp thiếu, multipart lỗi, tệp quá 2 MB và MIME giả bị từ chối đúng status.
- User không tồn tại trả 404 cho thao tác của Admin.
- Audit chỉ ghi metadata an toàn sau thao tác thành công.

### Frontend

- Hàm validation chấp nhận JPEG/PNG hợp lệ và từ chối sai loại/quá dung lượng.
- Form tạo user gọi upload sau khi user được tạo thành công.
- Upload thất bại không biến thành thông báo tạo user thất bại.
- Component avatar dùng ảnh Blob khi có và fallback chữ cái khi 404/lỗi.
- Object URL được thu hồi khi thay ảnh hoặc unmount.
- Employee portal không hiển thị thao tác chỉnh ảnh.

### Xác minh toàn dự án

- `go test ./...`
- `go vet ./...`
- `npm.cmd test`
- `npm.cmd run lint`
- `npm.cmd run build`
- `git diff --check`

## 10. Tài liệu và nâng cấp sau này

OpenAPI bổ sung ba endpoint avatar, request multipart, response nhị phân và các status lỗi. Tài liệu kiểm thử thủ công mô tả cả Admin và Employee.

Khi chuyển sang lưu trữ bền vững, giữ nguyên `Store` và API; chỉ thêm implementation mới lưu object/file và metadata cần thiết. PostgreSQL chỉ nên lưu metadata hoặc object key, không lưu Base64 trong bảng user. Việc chuyển đổi này nằm ngoài phạm vi hiện tại.
