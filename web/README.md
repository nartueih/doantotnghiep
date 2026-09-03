# License Manager Web

Web quản trị cho hệ thống Enterprise License Manager, xây dựng bằng React, TypeScript và Vite.

## Chạy ở môi trường development

Backend cần chạy tại `http://localhost:8080`. Sau đó mở terminal khác:

```powershell
Set-Location web
npm.cmd run dev
```

Mở `http://localhost:5173`. Vite chuyển tiếp các request `/api` đến backend, vì vậy trình duyệt không cần cấu hình CORS riêng trong lúc phát triển.

Muốn đổi địa chỉ backend, sao chép `.env.example` thành `.env.local` và sửa `VITE_API_PROXY_TARGET`.

## Kiểm tra

```powershell
npm.cmd run lint
npm.cmd run build
```
