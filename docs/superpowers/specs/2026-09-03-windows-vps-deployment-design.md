# Thiết kế triển khai License Manager lên Windows Server 2016

## Mục tiêu

Triển khai bản demo của License Manager tại `https://tranhieu1834.duckdns.org` trên VPS Windows Server 2016 Desktop Experience có 8 GB RAM. Frontend React chạy bằng IIS, backend Go chạy dưới dạng file EXE và PostgreSQL lưu dữ liệu trên cùng VPS.

## Kiến trúc

```text
Internet (TCP 80/443)
        |
        v
       IIS
        |-- /*             -> React static files
        |-- /api/*         -> http://127.0.0.1:8080
        `-- /health/*      -> http://127.0.0.1:8080
                                  |
                                  v
                         PostgreSQL 15
                         localhost:5432
```

Router chỉ chuyển tiếp TCP 80 và 443 vào IP nội bộ cố định của VPS. PostgreSQL 5432 và Go 8080 không được công khai ra Internet.

## Cấu trúc thư mục

```text
C:\Deploy\LicenseManager\
|-- backend\
|   |-- license-api.exe
|   |-- license-migrate.exe
|   |-- license-seed.exe
|   `-- logs\
|-- web\
|-- migrations\
|-- service\
`-- backups\
```

Frontend và các EXE được build trên máy phát triển rồi chuyển lên VPS. VPS không cần cài Git, Go hoặc Node.js để vận hành ứng dụng.

## Thành phần máy chủ

- PostgreSQL 15 x64, chạy bằng Windows Service của bộ cài chính thức.
- IIS với Static Content, Default Document và HTTP Errors.
- IIS URL Rewrite và Application Request Routing (ARR), bật proxy để chuyển tiếp API.
- Công cụ ACME dành cho IIS để cấp và tự gia hạn chứng chỉ TLS công khai.
- Backend ban đầu chạy tương tác trong PowerShell. Sau khi kiểm thử đầy đủ, EXE được đăng ký thành Windows Service và tự khởi động lại khi máy chủ reboot.

## Cấu hình và bí mật

Backend chạy với:

- `APP_ENV=production`
- `HTTP_ADDRESS=127.0.0.1:8080`
- `STORAGE_DRIVER=postgres`
- `DATABASE_URL` trỏ tới database `license_manager` trên localhost
- `JWT_SECRET` ngẫu nhiên, tối thiểu 32 ký tự
- `LICENSE_ENCRYPTION_KEY` là 32 byte ngẫu nhiên được mã hóa Base64
- thời hạn access/refresh token giữ theo cấu hình ứng dụng đã kiểm thử

Mật khẩu database, JWT secret và encryption key được tạo mới trực tiếp trên VPS, không ghi vào Git, tài liệu, ảnh chụp hoặc tin nhắn. Encryption key phải được sao lưu an toàn; mất khóa sẽ khiến activation key đã mã hóa không thể giải mã.

## Database mới

Tạo role `license_admin` và database `license_manager` mới. Sau đó chạy `license-migrate.exe up` rồi chạy seed một lần để tạo admin và dữ liệu demo. PostgreSQL chỉ lắng nghe localhost; không tạo firewall rule cho cổng 5432.

## IIS và định tuyến

IIS dùng `C:\Deploy\LicenseManager\web` làm physical path. Quy tắc rewrite phải xử lý `/api` và `/health` trước quy tắc React SPA fallback. Những URL frontend không trỏ đến file/thư mục thật được trả về `index.html` để tải lại trang không gặp 404.

Web và Android cùng sử dụng base URL `https://tranhieu1834.duckdns.org`; không dùng URL có cổng 8080. HTTPS chuyển hướng toàn bộ HTTP sang HTTPS sau khi chứng chỉ hoạt động.

## Triển khai và cập nhật

1. Kiểm tra IP nội bộ tĩnh, DuckDNS, port forwarding và Windows Firewall.
2. Cài PostgreSQL 15 và tạo database.
3. Cài IIS cùng các module reverse proxy.
4. Build backend và frontend trên máy phát triển.
5. Chuyển artifact lên VPS và chạy migration/seed.
6. Chạy backend tương tác, kiểm tra health endpoint nội bộ.
7. Cấu hình IIS, kiểm tra web/API qua HTTP.
8. Cấp chứng chỉ và bắt buộc HTTPS.
9. Kiểm tra web và Android qua mạng ngoài.
10. Chuyển backend thành Windows Service, reboot VPS và kiểm tra lại.

Khi cập nhật, sao lưu database và thư mục artifact hiện tại, dừng backend service, thay file mới, chạy migration, khởi động lại và kiểm tra health. Nếu thất bại, khôi phục artifact và database backup gần nhất.

## Kiểm thử chấp nhận

- DuckDNS trỏ đúng IP công cộng và truy cập được từ mạng ngoài.
- HTTP tự chuyển sang HTTPS, chứng chỉ hợp lệ và đúng tên miền.
- `/health/live` và `/health/ready` trả về thành công qua IIS.
- Admin đăng nhập web và dùng được các module chính.
- Employee đăng nhập Android, tải dữ liệu, gửi yêu cầu license/bảo trì và nhận thông báo.
- Dữ liệu còn nguyên sau khi khởi động lại backend và reboot VPS.
- Truy cập công khai vào các cổng 5432 và 8080 thất bại.
- Không có bí mật nào nằm trong Git hoặc thư mục web công khai.

## Giới hạn phạm vi

Đây là môi trường demo bảo vệ đồ án trên một VPS. Không triển khai cân bằng tải, database replication, container orchestration hay CI/CD. Sao lưu thủ công trước mỗi lần cập nhật là đủ cho phạm vi hiện tại.
