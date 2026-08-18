package developmentseed

import (
	"context"
	"fmt"
	"time"

	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/departments"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/modules/users"
)

const DemoUserPassword = "ChangeMe123!"

type Services struct {
	Departments *departments.Service
	Users       *users.Service
	Software    *software.Service
	Licenses    *licenses.Service
	Devices     *devices.Service
	Assignments *assignments.Service
}

type Result struct {
	Departments int
	Users       int
	Software    int
	Licenses    int
	Devices     int
	Assignments int
}

func Seed(ctx context.Context, services Services, actorID string, now time.Time) (Result, error) {
	departmentItems, err := seedDepartments(ctx, services.Departments)
	if err != nil {
		return Result{}, err
	}
	userItems, err := seedUsers(ctx, services.Users, departmentItems)
	if err != nil {
		return Result{}, err
	}
	softwareItems, err := seedSoftware(ctx, services.Software)
	if err != nil {
		return Result{}, err
	}
	licenseItems, err := seedLicenses(ctx, services.Licenses, softwareItems, now)
	if err != nil {
		return Result{}, err
	}
	deviceItems, err := seedDevices(ctx, services.Devices, userItems, now)
	if err != nil {
		return Result{}, err
	}
	assignmentCount, err := seedAssignments(ctx, services.Assignments, actorID, userItems, deviceItems, licenseItems)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Departments: len(departmentItems),
		Users:       len(userItems),
		Software:    len(softwareItems),
		Licenses:    len(licenseItems),
		Devices:     len(deviceItems),
		Assignments: assignmentCount,
	}, nil
}

func seedDepartments(ctx context.Context, service *departments.Service) ([]departments.Department, error) {
	inputs := []departments.Input{
		{Name: "Công nghệ thông tin", Code: "IT"},
		{Name: "Thiết kế", Code: "DESIGN"},
		{Name: "Vận hành", Code: "OPS"},
	}
	items := make([]departments.Department, 0, len(inputs))
	for _, input := range inputs {
		item, err := service.Create(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("seed department %q: %w", input.Code, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func seedUsers(ctx context.Context, service *users.Service, departmentItems []departments.Department) ([]auth.User, error) {
	inputs := []users.CreateInput{
		{Email: "it.manager@local.test", Password: DemoUserPassword, FullName: "Trần Minh Quân", EmployeeCode: "DEMO-001", DepartmentID: departmentItems[0].ID, Role: auth.RoleITManager},
		{Email: "anh.nguyen@local.test", Password: DemoUserPassword, FullName: "Nguyễn Hoàng Anh", EmployeeCode: "DEMO-002", DepartmentID: departmentItems[0].ID, Role: auth.RoleEmployee},
		{Email: "linh.tran@local.test", Password: DemoUserPassword, FullName: "Trần Mỹ Linh", EmployeeCode: "DEMO-003", DepartmentID: departmentItems[1].ID, Role: auth.RoleEmployee},
		{Email: "minh.le@local.test", Password: DemoUserPassword, FullName: "Lê Quang Minh", EmployeeCode: "DEMO-004", DepartmentID: departmentItems[1].ID, Role: auth.RoleEmployee},
		{Email: "hoa.pham@local.test", Password: DemoUserPassword, FullName: "Phạm Thu Hoa", EmployeeCode: "DEMO-005", DepartmentID: departmentItems[2].ID, Role: auth.RoleEmployee},
	}
	items := make([]auth.User, 0, len(inputs))
	for _, input := range inputs {
		item, err := service.Create(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("seed user %q: %w", input.Email, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func seedSoftware(ctx context.Context, service *software.Service) ([]software.Product, error) {
	inputs := []software.Input{
		{Name: "Microsoft 365", Publisher: "Microsoft", Version: "Business Premium", Description: "Bộ công cụ cộng tác và văn phòng cho doanh nghiệp."},
		{Name: "Adobe Creative Cloud", Publisher: "Adobe", Version: "All Apps", Description: "Bộ ứng dụng thiết kế dành cho đội ngũ sáng tạo."},
		{Name: "Figma", Publisher: "Figma", Version: "Organization", Description: "Nền tảng thiết kế và cộng tác giao diện."},
		{Name: "JetBrains All Products Pack", Publisher: "JetBrains", Version: "2026", Description: "Bộ IDE dành cho đội ngũ phát triển phần mềm."},
		{Name: "Windows 11 Pro", Publisher: "Microsoft", Version: "24H2", Description: "Hệ điều hành cho máy trạm doanh nghiệp."},
		{Name: "Zoom Workplace", Publisher: "Zoom", Version: "Business", Description: "Nền tảng họp và cộng tác trực tuyến."},
	}
	items := make([]software.Product, 0, len(inputs))
	for _, input := range inputs {
		item, err := service.Create(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("seed software %q: %w", input.Name, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func seedLicenses(ctx context.Context, service *licenses.Service, products []software.Product, now time.Time) ([]licenses.License, error) {
	date := func(days int) string { return now.AddDate(0, 0, days).Format("2006-01-02") }
	inputs := []licenses.Input{
		{SoftwareProductID: products[0].ID, Name: "Microsoft 365 Business Premium", LicenseType: licenses.TypeSubscription, AssignmentType: licenses.AssignmentUser, SeatCount: 5, LicenseKey: "DEMO-M365-2026-PREM", Vendor: "Microsoft Vietnam", PurchasedAt: date(-340), StartsAt: date(-330), ExpiresAt: date(24), Cost: 75000000, Currency: "VND", Notes: "Gia hạn hàng năm cho khối văn phòng."},
		{SoftwareProductID: products[1].ID, Name: "Adobe Creative Cloud All Apps", LicenseType: licenses.TypeSubscription, AssignmentType: licenses.AssignmentUser, SeatCount: 3, LicenseKey: "DEMO-ADOBE-ALL-APPS", Vendor: "Adobe Partner", PurchasedAt: date(-300), StartsAt: date(-295), ExpiresAt: date(55), Cost: 42000000, Currency: "VND", Notes: "Dành cho phòng Thiết kế."},
		{SoftwareProductID: products[2].ID, Name: "Figma Organization", LicenseType: licenses.TypeSubscription, AssignmentType: licenses.AssignmentUser, SeatCount: 4, LicenseKey: "DEMO-FIGMA-ORG-2026", Vendor: "Figma", PurchasedAt: date(-350), StartsAt: date(-350), ExpiresAt: date(12), Cost: 25000000, Currency: "VND", Notes: "License thiết kế sản phẩm."},
		{SoftwareProductID: products[3].ID, Name: "JetBrains All Products", LicenseType: licenses.TypeSubscription, AssignmentType: licenses.AssignmentUser, SeatCount: 4, LicenseKey: "DEMO-JB-ALL-PRODUCTS", Vendor: "JetBrains", PurchasedAt: date(-280), StartsAt: date(-275), ExpiresAt: date(82), Cost: 30000000, Currency: "VND", Notes: "Công cụ phát triển cho đội IT."},
		{SoftwareProductID: products[4].ID, Name: "Windows 11 Pro Volume", LicenseType: licenses.TypePerpetual, AssignmentType: licenses.AssignmentDevice, SeatCount: 6, LicenseKey: "DEMO-WIN11-VOLUME-KEY", Vendor: "Microsoft CSP", PurchasedAt: date(-500), StartsAt: date(-500), Cost: 36000000, Currency: "VND", Notes: "License vĩnh viễn theo thiết bị."},
		{SoftwareProductID: products[5].ID, Name: "Zoom Workplace Business", LicenseType: licenses.TypeSubscription, AssignmentType: licenses.AssignmentUser, SeatCount: 10, LicenseKey: "DEMO-ZOOM-BUSINESS", Vendor: "Zoom", PurchasedAt: date(-380), StartsAt: date(-370), ExpiresAt: date(-7), Cost: 18000000, Currency: "VND", Notes: "Đã hết hạn, cần xem xét gia hạn."},
	}
	items := make([]licenses.License, 0, len(inputs))
	for _, input := range inputs {
		item, err := service.Create(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("seed license %q: %w", input.Name, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func seedDevices(ctx context.Context, service *devices.Service, userItems []auth.User, now time.Time) ([]devices.Device, error) {
	date := func(days int) string { return now.AddDate(0, 0, days).Format("2006-01-02") }
	inputs := []devices.Input{
		{AssetCode: "LT-001", SerialNumber: "DEMO-DELL-001", Name: "Laptop Dell Latitude", DeviceType: "laptop", Manufacturer: "Dell", Model: "Latitude 7450", PurchasedAt: date(-260), WarrantyExpiresAt: date(835)},
		{AssetCode: "LT-002", SerialNumber: "DEMO-HP-002", Name: "Laptop HP EliteBook", DeviceType: "laptop", Manufacturer: "HP", Model: "EliteBook 840", PurchasedAt: date(-210), WarrantyExpiresAt: date(885)},
		{AssetCode: "WS-001", SerialNumber: "DEMO-DELL-WS-001", Name: "Workstation Thiết kế", DeviceType: "workstation", Manufacturer: "Dell", Model: "Precision 3680", PurchasedAt: date(-180), WarrantyExpiresAt: date(915)},
		{AssetCode: "MB-001", SerialNumber: "DEMO-APPLE-001", Name: "MacBook Pro Thiết kế", DeviceType: "laptop", Manufacturer: "Apple", Model: "MacBook Pro 14", PurchasedAt: date(-120), WarrantyExpiresAt: date(245)},
		{AssetCode: "SV-001", SerialNumber: "DEMO-SERVER-001", Name: "Máy chủ nội bộ", DeviceType: "server", Manufacturer: "Dell", Model: "PowerEdge T350", PurchasedAt: date(-700), WarrantyExpiresAt: date(395)},
		{AssetCode: "DT-OLD-001", SerialNumber: "DEMO-OLD-001", Name: "Máy bàn văn phòng cũ", DeviceType: "desktop", Manufacturer: "Lenovo", Model: "ThinkCentre M720", PurchasedAt: date(-1700), WarrantyExpiresAt: date(-605)},
	}
	items := make([]devices.Device, 0, len(inputs))
	for _, input := range inputs {
		item, err := service.Create(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("seed device %q: %w", input.AssetCode, err)
		}
		items = append(items, item)
	}

	var err error
	items[0], err = service.Assign(ctx, items[0].ID, userItems[1].ID)
	if err != nil {
		return nil, fmt.Errorf("assign demo device %q: %w", items[0].AssetCode, err)
	}
	items[1], err = service.Assign(ctx, items[1].ID, userItems[2].ID)
	if err != nil {
		return nil, fmt.Errorf("assign demo device %q: %w", items[1].AssetCode, err)
	}
	items[4], err = service.ChangeStatus(ctx, items[4].ID, devices.StatusMaintenance)
	if err != nil {
		return nil, fmt.Errorf("mark demo device %q as maintenance: %w", items[4].AssetCode, err)
	}
	items[5], err = service.ChangeStatus(ctx, items[5].ID, devices.StatusRetired)
	if err != nil {
		return nil, fmt.Errorf("retire demo device %q: %w", items[5].AssetCode, err)
	}
	return items, nil
}

func seedAssignments(
	ctx context.Context,
	service *assignments.Service,
	actorID string,
	userItems []auth.User,
	deviceItems []devices.Device,
	licenseItems []licenses.License,
) (int, error) {
	inputs := []assignments.CreateInput{
		{LicenseID: licenseItems[0].ID, UserID: userItems[0].ID, Notes: "Demo Microsoft 365"},
		{LicenseID: licenseItems[0].ID, UserID: userItems[1].ID, Notes: "Demo Microsoft 365"},
		{LicenseID: licenseItems[0].ID, UserID: userItems[2].ID, Notes: "Demo Microsoft 365"},
		{LicenseID: licenseItems[0].ID, UserID: userItems[3].ID, Notes: "Demo Microsoft 365"},
		{LicenseID: licenseItems[1].ID, UserID: userItems[2].ID, Notes: "Demo Adobe"},
		{LicenseID: licenseItems[1].ID, UserID: userItems[3].ID, Notes: "Demo Adobe"},
		{LicenseID: licenseItems[1].ID, UserID: userItems[4].ID, Notes: "Demo Adobe"},
		{LicenseID: licenseItems[2].ID, UserID: userItems[2].ID, Notes: "Demo Figma"},
		{LicenseID: licenseItems[2].ID, UserID: userItems[3].ID, Notes: "Demo Figma"},
		{LicenseID: licenseItems[3].ID, UserID: userItems[0].ID, Notes: "Demo JetBrains"},
		{LicenseID: licenseItems[4].ID, DeviceID: deviceItems[0].ID, Notes: "Demo Windows"},
		{LicenseID: licenseItems[4].ID, DeviceID: deviceItems[1].ID, Notes: "Demo Windows"},
		{LicenseID: licenseItems[4].ID, DeviceID: deviceItems[2].ID, Notes: "Demo Windows"},
		{LicenseID: licenseItems[4].ID, DeviceID: deviceItems[3].ID, Notes: "Demo Windows"},
	}
	for _, input := range inputs {
		if _, err := service.Create(ctx, actorID, input); err != nil {
			return 0, fmt.Errorf("seed assignment for license %q: %w", input.LicenseID, err)
		}
	}
	return len(inputs), nil
}
