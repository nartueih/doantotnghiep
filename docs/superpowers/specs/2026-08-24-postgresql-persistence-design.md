# Thiết kế hoàn thiện PostgreSQL persistence

## 1. Mục tiêu

Chuyển backend License Manager từ chế độ lưu trữ tạm thời trong RAM sang PostgreSQL cho toàn bộ luồng nghiệp vụ hiện có. Dữ liệu phải tồn tại sau khi backend khởi động lại, các thay đổi nhiều bước phải có tính nguyên tử, và môi trường phát triển phải có cách chạy migration, tạo tài khoản Admin và kiểm thử database lặp lại được.

PostgreSQL chỉ được truy cập bởi Go backend. Web và ứng dụng Android sau này tiếp tục gọi REST API; chúng không nhận connection string và không kết nối trực tiếp tới database.

## 2. Hiện trạng

Máy phát triển đã có PostgreSQL 18.4, dịch vụ `postgresql-x64-18` đang chạy và `localhost:5432` đang nhận kết nối.

Các module sau đã có PostgreSQL repository:

- authentication và users;
- audit log;
- departments;
- software products;
- licenses;
- devices;
- license assignments.

Các migration `001_initial_schema.sql`, `002_license_archiving.sql` và `003_employee_license_key_access.sql` đã mô tả schema của các module trên. Hai module `licenserequests` và `notifications` mới chỉ có memory repository. Trong `cmd/api/main.go`, handler của hai module này chỉ được khởi tạo khi `STORAGE_DRIVER=memory`, nên các route tương ứng chưa hoạt động với PostgreSQL.

## 3. Phạm vi

### Trong phạm vi

- Tạo database phát triển `license_manager` và database kiểm thử `license_manager_test` bằng một PostgreSQL role riêng.
- Thêm migration có phiên bản cho license requests và website notifications.
- Thêm công cụ migration chạy từ Go, theo dõi migration đã áp dụng và từ chối chạy sai thứ tự.
- Viết PostgreSQL repository cho `licenserequests` và `notifications`.
- Kết nối hai repository mới trong `cmd/api/main.go` khi `STORAGE_DRIVER=postgres`.
- Làm cho luồng duyệt hoặc từ chối yêu cầu chạy trong transaction PostgreSQL.
- Bảo vệ chuyển trạng thái và số seat khi nhiều request chạy đồng thời hoặc khi backend có nhiều instance.
- Thêm lệnh seed idempotent để tạo tài khoản Admin phát triển.
- Thêm integration test PostgreSQL và kịch bản kiểm thử persistence thủ công.
- Cập nhật cấu hình mẫu, README và tài liệu chạy không dùng Docker.

### Ngoài phạm vi

- Di chuyển dữ liệu đang nằm trong memory; dữ liệu này không tồn tại sau khi tiến trình cũ dừng nên database mới bắt đầu sạch.
- Thay toàn bộ repository hiện tại bằng ORM hoặc GORM.
- Kết nối trực tiếp web/mobile tới PostgreSQL.
- Triển khai PostgreSQL cloud, replication, high availability hoặc tự động backup production.
- Email, SMS hoặc mobile push notification.
- Seed toàn bộ dữ liệu demo. Sau khi seed Admin, dữ liệu nghiệp vụ được tạo qua UI/API để kiểm thử đúng luồng thật.

## 4. Kiến trúc

Kiến trúc repository interface hiện tại được giữ nguyên. Memory repository tiếp tục dùng cho unit test nhanh. PostgreSQL repository dùng `pgx/v5` và chia sẻ connection pool do `cmd/api` tạo.

Thêm package migration dưới `backend/migrations`. Package này embed các file SQL, sắp xếp theo tiền tố số và cung cấp danh sách migration cho lệnh `backend/cmd/migrate`. Lệnh hỗ trợ:

- `go run ./cmd/migrate up`: áp dụng các migration chưa chạy;
- `go run ./cmd/migrate status`: hiển thị migration đã chạy và còn thiếu.

Mỗi migration được chạy trong transaction riêng. Bảng `schema_migrations` lưu `version`, `name` và `applied_at`. Migration runner dùng PostgreSQL advisory lock để hai tiến trình không cùng chạy migration. Không cung cấp lệnh `down` trong giai đoạn này nhằm tránh thao tác xóa schema ngoài ý muốn.

Lệnh `backend/cmd/seed` dùng cấu hình PostgreSQL và password hasher hiện có để tạo tài khoản Admin. Lệnh này idempotent theo email và employee code: chạy lại không tạo bản ghi trùng và không ghi mật khẩu thô vào source hoặc migration.

## 5. Schema mới

Migration `004_license_requests_and_notifications.sql` tạo hai bảng.

### `license_requests`

- `id UUID PRIMARY KEY`;
- `requester_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT`;
- `requester_name VARCHAR(150) NOT NULL`;
- `software_product_id UUID NOT NULL REFERENCES software_products(id) ON DELETE RESTRICT`;
- `software_product_name VARCHAR(150) NOT NULL`;
- `priority VARCHAR(20) NOT NULL` với check `normal`, `high`, `urgent`;
- `reason TEXT NOT NULL` và không chấp nhận chuỗi rỗng sau khi trim ở service;
- `status VARCHAR(20) NOT NULL` với check `pending`, `approved`, `rejected`, `cancelled`;
- `selected_license_id UUID REFERENCES licenses(id) ON DELETE RESTRICT`;
- `selected_license_name VARCHAR(150)`;
- `assignment_id UUID REFERENCES license_assignments(id) ON DELETE RESTRICT`;
- `reviewed_by UUID REFERENCES users(id) ON DELETE RESTRICT`;
- `reviewed_by_name VARCHAR(150)`;
- `decision_reason VARCHAR(30)` với check nullable hoặc `out_of_stock`, `not_approved`, `other`;
- `response_note TEXT`;
- `created_at`, `updated_at` là `TIMESTAMPTZ NOT NULL`;
- `reviewed_at`, `cancelled_at` là `TIMESTAMPTZ` nullable.

Các check constraint bảo đảm trạng thái kết thúc có dữ liệu phù hợp: `approved` phải có license, assignment, reviewer và `reviewed_at`; `rejected` phải có reviewer, decision reason, response note và `reviewed_at`; `cancelled` phải có `cancelled_at`; `pending` chưa được chứa dữ liệu quyết định.

Một unique partial index trên `(requester_id, software_product_id) WHERE status = 'pending'` ngăn yêu cầu đang chờ bị tạo trùng ngay cả khi nhiều request chạy đồng thời. Index bổ sung phục vụ `requester_id, created_at DESC`, `status, created_at DESC` và bộ lọc `priority`.

Tên người dùng, phần mềm, license và reviewer được lưu snapshot cùng foreign key để lịch sử không đổi khi tên hiển thị của thực thể liên quan được sửa.

### `notifications`

- `id UUID PRIMARY KEY`;
- `user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE`;
- `type VARCHAR(80) NOT NULL`;
- `title VARCHAR(200) NOT NULL`;
- `message TEXT NOT NULL`;
- `entity_type VARCHAR(80) NOT NULL`;
- `entity_id UUID NOT NULL`;
- `created_at TIMESTAMPTZ NOT NULL`;
- `read_at TIMESTAMPTZ` nullable.

Index `(user_id, created_at DESC)` phục vụ danh sách. Partial index `(user_id, created_at DESC) WHERE read_at IS NULL` phục vụ badge chưa đọc. Notification không lưu activation key, password, JWT hoặc encrypted license key.

## 6. Transaction và đồng thời

Service hiện dùng mutex để bảo vệ memory trong một tiến trình. Mutex không đủ khi chạy nhiều backend instance, nên PostgreSQL phải dùng row lock và transaction.

Thêm transaction manager có hai implementation:

- memory transaction manager gọi callback trực tiếp;
- PostgreSQL transaction manager mở `pgx.Tx`, đặt transaction vào context và commit hoặc rollback callback.

Các PostgreSQL repository liên quan nhận một abstraction `DBTX` để thực thi query qua pool hoặc transaction lấy từ context. `assignments.PostgresRepository.Create` phải tham gia transaction có sẵn thay vì luôn mở và commit transaction riêng; khi được gọi độc lập, repository vẫn tự mở transaction để giữ hành vi hiện tại.

Luồng approve trong một transaction:

1. Khóa hàng `license_requests` bằng `SELECT ... FOR UPDATE` và xác nhận còn `pending`.
2. Đọc reviewer và license; kiểm tra license thuộc đúng software product.
3. Khóa hàng license, đếm active assignments và kiểm tra seat.
4. Tạo đúng một license assignment.
5. Cập nhật request sang `approved` bằng điều kiện trạng thái `pending`.
6. Tạo notification cho requester.
7. Commit; bất kỳ lỗi nào ở bước 1-6 đều rollback assignment, request và notification.

Luồng reject khóa request, cập nhật trạng thái và tạo notification trong cùng transaction. Luồng cancel dùng `UPDATE ... WHERE id = ? AND requester_id = ? AND status = 'pending'`, nên chỉ chủ sở hữu có thể hủy và thao tác đồng thời không tạo trạng thái sai.

Unique index active assignment hiện có và row lock trên license tiếp tục bảo vệ seat. Hai thao tác approve đồng thời cho cùng request chỉ có một thao tác thành công. Audit log hiện vẫn được ghi sau khi transaction nghiệp vụ commit theo kiến trúc HTTP handler hiện tại; việc đưa audit vào cùng transaction không nằm trong phạm vi này.

## 7. Repository PostgreSQL

`licenserequests.PostgresRepository` thực hiện:

- list toàn bộ với status, priority và search;
- list theo requester;
- find và lock request cho quyết định;
- create với ánh xạ lỗi unique thành `ErrPendingDuplicate`;
- cancel, approve, reject bằng conditional update;
- ánh xạ `pgx.ErrNoRows` và số hàng cập nhật về `ErrNotFound` hoặc `ErrInvalidState` nhất quán với memory repository.

`notifications.PostgresRepository` thực hiện:

- list notification của đúng user, mới nhất trước;
- create;
- mark một notification là đã đọc với điều kiện `user_id`;
- mark toàn bộ notification của một user là đã đọc;
- giữ thao tác mark-read idempotent.

Các query luôn dùng tham số `$1`, `$2`, không nối chuỗi từ input. Search dùng `ILIKE` với giá trị tham số. Lỗi PostgreSQL được bọc với ngữ cảnh nhưng vẫn giữ khả năng kiểm tra lỗi nghiệp vụ bằng `errors.Is`.

## 8. Cấu hình và bảo mật

Môi trường local dùng:

- `STORAGE_DRIVER=postgres`;
- `DATABASE_URL=postgres://license_admin:<password>@localhost:5432/license_manager?sslmode=disable`;
- `TEST_DATABASE_URL=postgres://license_admin:<password>@localhost:5432/license_manager_test?sslmode=disable`;
- `JWT_SECRET` tối thiểu 32 ký tự;
- `LICENSE_ENCRYPTION_KEY` là 32 byte được mã hóa Base64.

`sslmode=disable` chỉ dùng trên máy local. Password database, JWT secret và encryption key không được commit. File `.env.example` chỉ chứa placeholder. PostgreSQL role ứng dụng không dùng superuser. Database phát triển và database test tách riêng để test không xóa dữ liệu đang dùng.

Backend phải thất bại sớm với thông báo rõ ràng nếu không kết nối được database hoặc schema migration chưa đủ. Endpoint readiness tiếp tục kiểm tra pool bằng `Ping`.

## 9. Bootstrap local

Trình tự thao tác sau khi code hoàn tất:

1. Kết nối bằng role `postgres` cục bộ.
2. Tạo role `license_admin` có `LOGIN` nhưng không có quyền superuser.
3. Tạo `license_manager` và `license_manager_test`, owner là `license_admin`.
4. Đặt `DATABASE_URL` rồi chạy `go run ./cmd/migrate up`.
5. Đặt thông tin Admin phát triển rồi chạy `go run ./cmd/seed`.
6. Chạy backend với `STORAGE_DRIVER=postgres`.
7. Tạo dữ liệu nghiệp vụ qua UI/API.

Nếu `psql` không nằm trong PATH, tài liệu sử dụng đường dẫn đầy đủ `C:\Program Files\PostgreSQL\18\bin\psql.exe`.

## 10. Kiểm thử

### Unit test

- Toàn bộ unit test memory hiện có tiếp tục pass.
- Transaction manager memory giữ cho service tests không cần PostgreSQL.
- Thêm test mapping lỗi và validation nếu repository interface thay đổi.

### PostgreSQL integration test

Integration test chỉ chạy khi có `TEST_DATABASE_URL`; nếu biến này không tồn tại thì skip với lý do rõ ràng. Test không chạy song song và dọn dữ liệu trong database test theo thứ tự foreign key.

Các trường hợp bắt buộc:

- migration chạy từ database rỗng và chạy lại không tạo lỗi;
- CRUD và filter license requests;
- unique pending request hoạt động khi tạo đồng thời;
- cancel chỉ tác động request của đúng requester;
- CRUD, mark-read và mark-all-read notification chỉ tác động đúng user;
- approve tạo assignment, cập nhật request và tạo notification;
- lỗi ở notification hoặc update request rollback assignment;
- hai approve đồng thời chỉ tạo một assignment;
- reject cập nhật request và notification nguyên tử;
- dữ liệu còn nguyên sau khi đóng pool và mở kết nối mới.

### Kiểm thử toàn hệ thống

- `go test ./...`;
- `go vet ./...`;
- web tests, lint và production build;
- health live/ready;
- login bằng Admin PostgreSQL;
- tạo user, phòng ban, phần mềm, license, thiết bị và assignment;
- Employee tạo yêu cầu, Admin duyệt hoặc từ chối, Employee nhận notification;
- tắt backend, chạy lại và xác nhận toàn bộ dữ liệu vẫn còn.

## 11. Tài liệu và tiêu chí hoàn thành

Cập nhật README và hướng dẫn chạy local không Docker với lệnh tạo role/database, migrate, seed, chạy API và xử lý các lỗi thường gặp. Tài liệu phải phân biệt rõ database role `license_admin` với user Admin của ứng dụng.

Hạng mục hoàn thành khi:

- không còn route nào chỉ hoạt động với memory;
- toàn bộ module dùng PostgreSQL khi `STORAGE_DRIVER=postgres`;
- approve/reject có transaction và kiểm thử rollback;
- migration và seed chạy lặp lại an toàn;
- integration tests PostgreSQL pass trên database test;
- kiểm thử thủ công xác nhận persistence sau restart;
- web không cần thay đổi API contract;
- backend đã sẵn sàng làm nguồn dữ liệu duy nhất cho ứng dụng Android.
