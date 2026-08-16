# Đặc tả sản phẩm MVP

## 1. Mục tiêu

Xây dựng một hệ thống tập trung giúp bộ phận IT biết công ty đang sở hữu những license phần mềm nào, còn bao nhiêu quyền sử dụng, đã cấp cho ai hoặc thiết bị nào và license nào sắp hết hạn.

## 2. Vai trò người dùng

| Vai trò | Quyền chính |
| --- | --- |
| Admin | Quản trị người dùng, vai trò, cấu hình và toàn bộ dữ liệu |
| IT Manager | Quản lý thiết bị, phần mềm, license, cấp phát và báo cáo |
| Employee | Xem thiết bị và license được cấp cho bản thân |

## 3. Quy tắc nghiệp vụ cốt lõi

1. Một license thuộc đúng một sản phẩm phần mềm.
2. License có thể là thuê bao hoặc vĩnh viễn.
3. Mỗi license có tổng số seat lớn hơn 0.
4. Một seat được cấp cho người dùng hoặc thiết bị, không được đồng thời cấp cho cả hai.
5. Số lượt cấp phát đang hoạt động không được vượt quá tổng số seat.
6. License hết hạn không được cấp phát mới.
7. Thu hồi không xóa lịch sử; bản ghi cấp phát được chuyển sang trạng thái `revoked`.
8. Mã kích hoạt là dữ liệu nhạy cảm: chỉ lưu dạng mã hóa, chỉ vai trò được phép mới có thể xem.
9. Các thao tác tạo, sửa, cấp phát, thu hồi và xem mã kích hoạt phải được ghi audit log.

## 4. Chức năng MVP

### Xác thực và phân quyền

- Đăng nhập bằng email và mật khẩu.
- Làm mới phiên đăng nhập bằng refresh token.
- Phân quyền theo vai trò.
- Khóa/mở khóa tài khoản.

### Thiết bị

- Thêm, sửa, xem và tìm kiếm thiết bị.
- Theo dõi mã tài sản, serial, loại thiết bị và trạng thái.
- Gán thiết bị cho nhân viên.
- Chuẩn bị dữ liệu QR để ứng dụng Android quét ở giai đoạn sau.

### Phần mềm và license

- Quản lý nhà sản xuất, tên và phiên bản phần mềm.
- Lưu loại license, số seat, ngày mua, ngày hết hạn, nhà cung cấp và chi phí.
- Hiển thị số seat đã dùng/còn lại.
- Cấp phát và thu hồi license.
- Lọc license đang hoạt động, sắp hết hạn hoặc đã hết hạn.

### Dashboard và thông báo

- Tổng số thiết bị, license và chi phí.
- Danh sách license hết hạn trong 30/60/90 ngày.
- Cảnh báo license hết seat hoặc đã vượt ngưỡng sử dụng.

## 5. Ngoài phạm vi MVP

- Ứng dụng Android, push notification và quét QR hoàn chỉnh.
- Đồng bộ tự động với Microsoft 365, Adobe Admin Console hoặc Google Workspace.
- Hệ thống mua sắm/phê duyệt nhiều cấp.
- Microservices và xử lý phân tán.

Các nội dung này có thể được chọn làm phần mở rộng sau khi MVP ổn định.

## 6. Tiêu chí hoàn thành MVP

- API có tài liệu OpenAPI và kiểm thử cho các luồng nghiệp vụ chính.
- Không thể cấp phát vượt quá số seat trong điều kiện có nhiều yêu cầu đồng thời.
- Mã license không xuất hiện dưới dạng plaintext trong cơ sở dữ liệu hoặc log.
- Có thể chạy toàn bộ backend và PostgreSQL bằng Docker Compose.
- Web quản trị thực hiện được luồng tạo license, cấp phát và thu hồi.

