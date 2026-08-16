# Kiểm tra Dashboard và cảnh báo license

Dashboard tổng hợp dữ liệu trực tiếp từ software, license và device hiện có. Admin và IT Manager được truy cập; Employee không được phép xem.

## API

| Method | Endpoint | Chức năng |
| --- | --- | --- |
| GET | `/api/v1/dashboard/summary` | Xem số liệu tổng quan |
| GET | `/api/v1/dashboard/license-alerts?days=30` | Xem cảnh báo license |

Query parameter `days` chỉ nhận `30`, `60` hoặc `90`; nếu bỏ trống thì mặc định là `30`.

## Dữ liệu tổng quan

- Tổng thiết bị và số lượng theo từng trạng thái.
- Tổng sản phẩm phần mềm và tổng license.
- Tổng seat, seat đã dùng và seat còn lại.
- License đã hết hạn và sắp hết hạn trong 30/60/90 ngày.
- License hết seat và license sử dụng từ 80% trở lên.
- Tổng chi phí license được nhóm theo từng loại tiền tệ.

Các mốc hết hạn là cộng dồn: license hết hạn trong 30 ngày cũng được tính trong nhóm 60 và 90 ngày.

## Mức độ cảnh báo

| Severity | Điều kiện |
| --- | --- |
| `critical` | License đã hết hạn hoặc đã dùng hết seat |
| `warning` | Sắp hết hạn trong 30 ngày hoặc mức sử dụng từ 80% |
| `info` | Sắp hết hạn trong khoảng được chọn nhưng còn trên 30 ngày |

Một license có thể có nhiều `alert_types`, ví dụ vừa `expired` vừa `exhausted`.

## Ví dụ kiểm tra

```powershell
$summary = Invoke-RestMethod `
    -Method Get `
    -Uri "http://localhost:8080/api/v1/dashboard/summary" `
    -Headers $adminHeaders

$alerts = Invoke-RestMethod `
    -Method Get `
    -Uri "http://localhost:8080/api/v1/dashboard/license-alerts?days=90" `
    -Headers $adminHeaders

$summary
$alerts.items | Format-Table license_name, severity, expires_at, utilization_percent
```

Chạy kiểm thử tự động từ thư mục `backend`:

```powershell
go test -count=1 ./...
go vet ./...
```
