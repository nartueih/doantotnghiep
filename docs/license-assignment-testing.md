# Kiểm tra cấp phát và thu hồi license

Admin và IT Manager có thể cấp phát/thu hồi. Employee không được truy cập API quản trị.

## API

| Method | Endpoint | Chức năng |
| --- | --- | --- |
| GET | `/api/v1/license-assignments` | Xem lịch sử cấp phát |
| POST | `/api/v1/license-assignments` | Cấp license cho user hoặc device |
| PATCH | `/api/v1/license-assignments/:id/revoke` | Thu hồi license |

## Luồng kiểm thử chính

1. Đăng nhập Admin và tạo hai Employee.
2. Tạo software product và license có một seat.
3. Cấp seat cho Employee thứ nhất.
4. Xác nhận license có `used_seats=1`, `available_seats=0`.
5. Cấp cho Employee thứ hai phải trả `409 Conflict`.
6. Thu hồi assignment thứ nhất; bản ghi chuyển thành `revoked`.
7. Cấp lại cho Employee thứ hai phải thành công.
8. Danh sách lịch sử phải giữ cả bản ghi revoked và active.

## Quy tắc bắt buộc

- Request phải có đúng một trong `user_id` hoặc `device_id`.
- License loại `user` không được cấp cho device và ngược lại.
- License `mixed` được cấp cho một trong hai loại đối tượng.
- Không cấp license chưa bắt đầu hoặc đã hết hạn.
- Không cấp cho user bị khóa hoặc device đã thanh lý/thất lạc.
- Không cấp trùng cùng license cho cùng đối tượng khi assignment cũ còn active.
- Thu hồi không xóa lịch sử và phải trả lại seat.
- Giới hạn seat được kiểm tra trong vùng khóa giao dịch để tránh vượt seat khi có request đồng thời.

