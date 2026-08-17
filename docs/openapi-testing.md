# Kiểm tra OpenAPI và Swagger UI

Backend nhúng OpenAPI specification trực tiếp vào binary, vì vậy tài liệu và ứng dụng luôn được triển khai cùng nhau.

## Endpoint tài liệu

| Endpoint | Chức năng |
| --- | --- |
| `GET /openapi.json` | OpenAPI 3.0.3 ở định dạng JSON |
| `GET /docs` | Swagger UI tương tác |

Hai endpoint này không yêu cầu đăng nhập.

## Sử dụng Swagger UI

1. Khởi động backend và mở `http://localhost:8080/docs`.
2. Mở operation `POST /api/v1/auth/login`, chọn **Try it out** và đăng nhập.
3. Sao chép giá trị `access_token` trong response.
4. Chọn nút **Authorize** ở đầu trang.
5. Dán riêng access token, không thêm chữ `Bearer`.
6. Thử các API phù hợp với role của tài khoản.

Swagger UI tải giao diện từ CDN. Nếu máy không có Internet, `/openapi.json` vẫn hoạt động và trang `/docs` sẽ hiển thị liên kết tải specification.

## Kiểm tra bằng PowerShell

```powershell
$specResponse = Invoke-WebRequest http://localhost:8080/openapi.json
$docsResponse = Invoke-WebRequest http://localhost:8080/docs

$specResponse.StatusCode
$specResponse.Headers["Content-Type"]
$docsResponse.StatusCode

$spec = $specResponse.Content | ConvertFrom-Json
$spec.openapi
$spec.info.title
@($spec.paths.PSObject.Properties).Count
```

Kết quả cần có OpenAPI `3.0.3`, đúng tên API và danh sách path không rỗng.

## Kiểm tra tự động

Test backend xác nhận:

- Specification là JSON hợp lệ.
- Mọi nhóm API đều được mô tả.
- Mọi operation có `operationId` duy nhất.
- Mọi `$ref` nội bộ đều phân giải được.
- `/openapi.json` và `/docs` có thể truy cập công khai.

```powershell
Set-Location backend
go test -count=1 ./...
go vet ./...
```
