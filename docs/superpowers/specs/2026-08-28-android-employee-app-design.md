# Thiết kế ứng dụng Android dành cho nhân viên

## 1. Mục tiêu

Xây dựng ứng dụng Android bằng Kotlin cho nhân viên sử dụng hệ thống License Manager. Ứng dụng dùng lại Go REST API hiện có và chỉ cung cấp các chức năng tự phục vụ của nhân viên, không đưa chức năng quản trị lên mobile.

Ứng dụng được phát triển theo từng giai đoạn nhỏ. Mỗi giai đoạn phải chạy được trên Android Emulator, có kiểm thử phù hợp và được xác nhận trước khi chuyển sang chức năng tiếp theo.

## 2. Phạm vi sản phẩm

### Chức năng cuối cùng

- Đăng nhập và duy trì phiên đăng nhập.
- Xem tổng quan tài sản cá nhân.
- Xem license đã được cấp và xem key khi được phép.
- Xem thiết bị đang được giao.
- Tạo, theo dõi và hủy yêu cầu cấp license.
- Tạo, theo dõi và hủy yêu cầu bảo trì thiết bị.
- Xem và đánh dấu thông báo website đã đọc.
- Xem thông tin tài khoản và đăng xuất.

### Không thuộc phạm vi

- Chức năng dành cho Admin hoặc IT Manager.
- Quản lý người dùng, phòng ban, phần mềm, license hoặc thiết bị.
- Phê duyệt yêu cầu trên mobile.
- Tải ảnh đại diện.
- Push notification qua Firebase trong phiên bản đầu.
- Offline-first hoặc đồng bộ dữ liệu bằng Room trong phiên bản đầu.
- Dark mode trong giai đoạn nền móng.

## 3. Thông tin project

- Tên ứng dụng: `License Manager`.
- Package name: `com.nartueih.licensemanager`.
- Thư mục repository: `android/`.
- Ngôn ngữ: Kotlin.
- Giao diện: Jetpack Compose với Material 3.
- Build scripts: Kotlin DSL.
- `minSdk`: API 26, Android 8.0.
- `compileSdk`: API 36.
- `targetSdk`: API 36.
- Thiết bị phát triển chính: Android Emulator.
- Backend debug: `http://10.0.2.2:8080/api/v1/`.

## 4. Kiến trúc

Ứng dụng dùng Single Activity, MVVM phân lớp và luồng dữ liệu một chiều. Phiên bản đầu dùng một Gradle module `app`; ranh giới được duy trì bằng package và interface thay vì tách nhiều module quá sớm.

```text
MainActivity
  -> LicenseManagerApp
      -> Navigation
          -> Screen composable
              -> ViewModel
                  -> Repository
                      -> Remote API / Session storage
```

### UI layer

- Compose hiển thị trạng thái bất biến từ ViewModel.
- Mỗi màn hình có ViewModel ở cấp navigation destination.
- ViewModel công khai một `StateFlow<UiState>` và nhận hành động qua hàm rõ nghĩa.
- UI không gọi Retrofit, DataStore hoặc Android Keystore trực tiếp.
- Navigation dùng một Activity và các destination Compose.

### Data layer

- Repository là ranh giới duy nhất giữa ViewModel và nguồn dữ liệu.
- Remote data source gọi Go REST API bằng Retrofit và OkHttp.
- Session data source mã hóa và lưu phiên đăng nhập.
- DTO phản ánh JSON của backend; repository ánh xạ DTO sang model mà UI cần.
- Domain layer riêng chỉ được thêm khi logic thực sự được dùng lại giữa nhiều ViewModel.

### Dependency injection

Hilt cung cấp singleton cho HTTP client, API service, session storage và repository; đồng thời cung cấp ViewModel theo từng màn hình. Không dùng service locator thủ công.

## 5. Cấu trúc package dự kiến

```text
com.nartueih.licensemanager
  app/
    LicenseManagerApplication
    LicenseManagerApp
    navigation/
  core/
    designsystem/
    network/
    session/
    common/
  data/
    auth/
    employee/
    notification/
  feature/
    auth/
    home/
    licenses/
    devices/
    requests/
    account/
```

Package chỉ được tạo khi giai đoạn tương ứng bắt đầu. Không tạo hàng loạt lớp rỗng cho chức năng tương lai.

## 6. Điều hướng và giao diện

Sau khi đăng nhập, ứng dụng dùng thanh điều hướng dưới với năm mục:

1. Tổng quan.
2. License.
3. Thiết bị.
4. Yêu cầu, bên trong chia yêu cầu License và Bảo trì.
5. Tài khoản.

Thông báo được mở từ biểu tượng chuông trên thanh đầu trang. Giao diện dùng Material 3 nhưng giữ nhận diện của website: xanh dương, trắng, card bo góc, badge trạng thái và icon người dùng mặc định. Typography phải đủ lớn, dễ đọc trên điện thoại. Phiên bản đầu chỉ hỗ trợ light theme.

Mỗi màn hình dữ liệu phải có các trạng thái loading, content, empty và error. Lỗi kết nối phải có nút `Thử lại`; tác vụ đang gửi phải ngăn nhấn lặp.

## 7. Đăng nhập và phiên người dùng

### Đăng nhập

1. Người dùng nhập email và mật khẩu.
2. App kiểm tra hai trường không trống và email đúng định dạng.
3. App gọi `POST /api/v1/auth/login`.
4. App chỉ chấp nhận phản hồi có `user.role == "employee"` và `user.status == "active"`.
5. Nếu tài khoản là Admin hoặc IT Manager, app gọi logout theo best effort, không lưu token và thông báo ứng dụng chỉ dành cho nhân viên.
6. Nếu hợp lệ, app mã hóa phiên, lưu trên thiết bị và điều hướng tới Tổng quan.

### Khôi phục phiên

Khi mở app, splash destination đọc session storage:

- Không có phiên: mở Login.
- Có access token còn hạn: mở khu vực nhân viên và xác thực người dùng khi cần.
- Access token hết hạn nhưng còn refresh token: gọi refresh.
- Refresh thành công: thay toàn bộ token pair và tiếp tục.
- Refresh thất bại do `401`: xóa phiên và quay về Login.
- Mất mạng tạm thời: giữ phiên, hiển thị lỗi kết nối và cho phép thử lại.

### Tự làm mới token

OkHttp gắn `Authorization: Bearer <access_token>` cho API được bảo vệ. Khi server trả `401`, bộ refresh chỉ thử một lần cho request đó. Các request đồng thời phải dùng chung một thao tác refresh để tránh luân chuyển refresh token nhiều lần. Request được thử lại bằng access token mới; nếu refresh thất bại, session bị xóa.

### Đăng xuất

App gọi `POST /api/v1/auth/logout` bằng refresh token theo best effort, sau đó luôn xóa phiên cục bộ và trở về Login. Lỗi mạng không được ngăn người dùng đăng xuất khỏi thiết bị.

## 8. Lưu trữ và bảo mật

- Tạo khóa AES trong Android Keystore.
- Mã hóa session payload bằng AES-GCM.
- Lưu ciphertext, IV và metadata cần thiết bằng DataStore.
- Không ghi access token, refresh token, mật khẩu hoặc license key vào log.
- Không commit mật khẩu, JWT secret, encryption key hoặc URL production bí mật.
- URL debug dùng `BuildConfig`; URL release phải là HTTPS.
- Cleartext HTTP chỉ được cho phép trong debug build để Emulator gọi backend local.
- Dữ liệu người dùng không nhạy cảm có thể được giữ trong session đã mã hóa để hiển thị nhanh khi khởi động.

## 9. Kết nối API và xử lý lỗi

Retrofit và Kotlin Serialization ánh xạ JSON `snake_case` của Go backend. OkHttp quản lý timeout, bearer token và refresh token. Mọi lỗi được chuẩn hóa thành nhóm:

- Validation: hiển thị cạnh trường nhập liệu.
- `401`: sai thông tin đăng nhập hoặc phiên hết hạn.
- `403`: tài khoản bị khóa, sai vai trò hoặc không có quyền.
- `404`: dữ liệu không còn tồn tại.
- `409`/`422`: xung đột quy tắc nghiệp vụ.
- `5xx`: lỗi máy chủ.
- IOException/timeout: không thể kết nối tới backend.
- JSON không hợp lệ: máy chủ trả về dữ liệu không hợp lệ.

UI hiển thị thông báo tiếng Việt phù hợp với ngữ cảnh. Repository giữ status code và mã lỗi nghiệp vụ khi backend cung cấp để các màn hình có thể xử lý chính xác.

## 10. Giai đoạn triển khai

### Giai đoạn 1: Nền móng, đăng nhập và Tổng quan

- Tạo project Android và cấu hình Gradle.
- Thiết lập Compose, Material 3, Navigation, Hilt, Retrofit, OkHttp, Serialization và DataStore.
- Tạo theme và component nền tảng.
- Triển khai session encryption và token refresh.
- Triển khai Login, splash/session restore và logout.
- Tạo app shell với bottom navigation.
- Tổng quan hiển thị dữ liệu thật:
  - Họ tên, mã nhân viên và phòng ban.
  - Tổng license từ `GET /api/v1/me/licenses`.
  - Tổng thiết bị từ `GET /api/v1/me/devices`.
  - Số thông báo chưa đọc từ `GET /api/v1/me/notifications`.
- Các mục chưa triển khai hiển thị trạng thái `Đang phát triển`.

### Giai đoạn 2: License của tôi

- Danh sách license được cấp.
- Trạng thái vòng đời và thông tin gán.
- Xem key khi `can_view_key == true`, có xác nhận và tự che lại.

### Giai đoạn 3: Thiết bị của tôi

- Danh sách và chi tiết thiết bị.
- Serial, asset code, loại, hãng, model, trạng thái, ngày mua và bảo hành.

### Giai đoạn 4: Yêu cầu

- Tạo, xem và hủy yêu cầu cấp license.
- Tạo, xem và hủy yêu cầu bảo trì cho thiết bị hợp lệ.
- Hiển thị phản hồi và toàn bộ vòng đời trạng thái.

### Giai đoạn 5: Thông báo và hoàn thiện

- Danh sách thông báo trong app.
- Đánh dấu một hoặc tất cả đã đọc.
- Điều hướng từ thông báo tới yêu cầu liên quan khi dữ liệu hỗ trợ.
- Hoàn thiện accessibility, responsive layout, lỗi và trạng thái rỗng.

## 11. Kiểm thử

### Unit test

- Validate email và mật khẩu.
- Auth repository với phản hồi thành công và các mã lỗi.
- Login ViewModel và StateFlow.
- Session storage mã hóa/giải mã.
- Refresh token thành công, thất bại và nhiều request đồng thời.
- Mapping DTO sang model UI.

### Network test

MockWebServer mô phỏng login, refresh, logout, timeout, JSON lỗi và API nghiệp vụ. Test phải xác nhận header Authorization, request body và giới hạn retry.

### Compose UI test

- Hiển thị và validate Login.
- Login thành công điều hướng tới Tổng quan.
- Phiên hợp lệ bỏ qua Login.
- Đăng xuất quay về Login và xóa back stack.
- Loading, error và retry hiển thị đúng.

### Kiểm thử thủ công

- Chạy Go backend trên máy ở cổng 8080.
- Chạy app trên Android Emulator.
- Đăng nhập bằng tài khoản employee trong PostgreSQL hoặc development seed.
- Tắt backend để kiểm tra lỗi kết nối.
- Khởi động lại app để kiểm tra khôi phục phiên.
- Làm access token hết hạn để kiểm tra refresh.
- Thử tài khoản admin để kiểm tra chặn sai vai trò.

## 12. Tiêu chí hoàn thành giai đoạn 1

- Project mở, sync và build thành công trong Android Studio.
- App chạy trên Emulator API 26 trở lên.
- Login với employee thật thành công.
- Admin và IT Manager không vào được khu vực nhân viên.
- Phiên được khôi phục sau khi đóng và mở app.
- Refresh token hoạt động mà người dùng không phải đăng nhập lại.
- Tổng quan hiển thị đúng số license, thiết bị và thông báo từ backend.
- Đăng xuất thu hồi phiên theo best effort và luôn xóa dữ liệu cục bộ.
- Các lỗi thường gặp có thông báo rõ ràng và app không crash.
- Unit test, network test, Compose UI test, lint và build đều đạt.

