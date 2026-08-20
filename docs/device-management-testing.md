# Kiểm tra module quản lý thiết bị

Sau khi kiểm tra API, có thể tiếp tục với [kịch bản kiểm thử giao diện quản lý thiết bị](web-device-management-testing.md).

Admin và IT Manager được quản lý thiết bị. Employee không được truy cập các API quản trị thiết bị.

## API

| Method | Endpoint | Chức năng |
| --- | --- | --- |
| GET | `/api/v1/devices` | Xem danh sách thiết bị |
| POST | `/api/v1/devices` | Tạo thiết bị |
| PUT | `/api/v1/devices/:id` | Cập nhật thông tin thiết bị |
| PATCH | `/api/v1/devices/:id/status` | Đổi trạng thái thiết bị |
| PATCH | `/api/v1/devices/:id/assignment` | Bàn giao hoặc thu hồi thiết bị |

## 1. Đăng nhập Admin

```powershell
$loginBody = @{ email = "admin@local.test"; password = "ChangeMe123!" } | ConvertTo-Json
$login = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/auth/login -ContentType "application/json" -Body $loginBody
$adminHeaders = @{ Authorization = "Bearer $($login.tokens.access_token)" }
```

## 2. Tạo Employee

```powershell
$employeeBody = @{
    email = "employee@local.test"
    password = "Employee123!"
    full_name = "Test Employee"
    employee_code = "EMP-001"
    role = "employee"
} | ConvertTo-Json

$employee = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/users -Headers $adminHeaders -ContentType "application/json" -Body $employeeBody
```

## 3. Tạo laptop

```powershell
$deviceBody = @{
    asset_code = "LAP-001"
    serial_number = "SN-001"
    name = "Developer Laptop"
    device_type = "laptop"
    manufacturer = "Dell"
    model = "Latitude 7450"
    purchased_at = "2026-01-10"
    warranty_expires_at = "2029-01-09"
} | ConvertTo-Json

$device = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/devices -Headers $adminHeaders -ContentType "application/json" -Body $deviceBody
$device | Format-List *
```

Thiết bị mới phải có status `available` và một UUID.

## 4. Chống trùng mã tài sản

Chạy lại request POST ở bước 3. API phải trả `409 Conflict` với lỗi `asset code already exists`. Kiểm tra trùng không phân biệt chữ hoa/thường.

## 5. Bàn giao cho Employee

```powershell
$assignmentBody = @{ user_id = $employee.id } | ConvertTo-Json

$assignedDevice = Invoke-RestMethod -Method Patch -Uri "http://localhost:8080/api/v1/devices/$($device.id)/assignment" -Headers $adminHeaders -ContentType "application/json" -Body $assignmentBody
$assignedDevice | Format-List asset_code, status, assigned_user_id, assigned_user_name
```

Status phải là `assigned` và tên người nhận là `Test Employee`.

## 6. Không đổi trạng thái khi đang bàn giao

```powershell
$maintenanceBody = @{ status = "maintenance" } | ConvertTo-Json
Invoke-RestMethod -Method Patch -Uri "http://localhost:8080/api/v1/devices/$($device.id)/status" -Headers $adminHeaders -ContentType "application/json" -Body $maintenanceBody
```

Kết quả đúng là `409 Conflict`. Thiết bị phải được thu hồi trước.

## 7. Thu hồi thiết bị

```powershell
$unassignBody = @{ user_id = "" } | ConvertTo-Json
$unassignedDevice = Invoke-RestMethod -Method Patch -Uri "http://localhost:8080/api/v1/devices/$($device.id)/assignment" -Headers $adminHeaders -ContentType "application/json" -Body $unassignBody
$unassignedDevice | Format-List asset_code, status, assigned_user_id
```

Status phải trở lại `available` và không còn `assigned_user_id`.

## 8. Chuyển sang bảo trì

```powershell
$maintenanceDevice = Invoke-RestMethod -Method Patch -Uri "http://localhost:8080/api/v1/devices/$($device.id)/status" -Headers $adminHeaders -ContentType "application/json" -Body $maintenanceBody
```

Status phải là `maintenance`.

## 9. Không bàn giao thiết bị bảo trì

Chạy lại request bàn giao ở bước 5. Kết quả đúng là `409 Conflict` với lỗi `device is not available for assignment`.

## 10. Đưa thiết bị về available

```powershell
$availableBody = @{ status = "available" } | ConvertTo-Json
$availableDevice = Invoke-RestMethod -Method Patch -Uri "http://localhost:8080/api/v1/devices/$($device.id)/status" -Headers $adminHeaders -ContentType "application/json" -Body $availableBody
```

## 11. Xem danh sách

```powershell
$devices = Invoke-RestMethod -Method Get -Uri http://localhost:8080/api/v1/devices -Headers $adminHeaders
$devices.total
$devices.items | Format-Table asset_code, name, device_type, manufacturer, model, status, assigned_user_name
```

