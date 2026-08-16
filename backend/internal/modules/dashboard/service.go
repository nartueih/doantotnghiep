package dashboard

import (
	"context"
	"math"
	"sort"
	"time"

	"license-manager/backend/internal/modules/licenses"
)

type Service struct {
	licenses LicenseLister
	devices  DeviceLister
	software SoftwareLister
	now      func() time.Time
}

func NewService(licenseLister LicenseLister, deviceLister DeviceLister, softwareLister SoftwareLister) *Service {
	return &Service{
		licenses: licenseLister,
		devices:  deviceLister,
		software: softwareLister,
		now:      time.Now,
	}
}

func (s *Service) Summary(ctx context.Context) (Summary, error) {
	licenseItems, err := s.licenses.List(ctx)
	if err != nil {
		return Summary{}, err
	}
	deviceItems, err := s.devices.List(ctx)
	if err != nil {
		return Summary{}, err
	}
	softwareItems, err := s.software.List(ctx)
	if err != nil {
		return Summary{}, err
	}

	today := startOfDay(s.now())
	result := Summary{
		TotalDevices:          len(deviceItems),
		DevicesByStatus:       make(map[string]int),
		TotalSoftwareProducts: len(softwareItems),
		TotalLicenses:         len(licenseItems),
		CostsByCurrency:       make([]CostByCurrency, 0),
		GeneratedAt:           s.now().UTC(),
	}
	for _, item := range deviceItems {
		result.DevicesByStatus[item.Status]++
	}

	costs := make(map[string]float64)
	for _, item := range licenseItems {
		result.TotalSeats += item.SeatCount
		result.UsedSeats += item.UsedSeats
		result.AvailableSeats += max(item.SeatCount-item.UsedSeats, 0)
		if item.Cost != 0 && item.Currency != "" {
			costs[item.Currency] += item.Cost
		}
		utilization := utilizationPercent(item)
		if item.UsedSeats >= item.SeatCount {
			result.ExhaustedLicenses++
		} else if utilization >= HighUsageThreshold {
			result.HighUsageLicenses++
		}

		days, hasExpiry := daysUntilExpiry(today, item.ExpiresAt)
		if !hasExpiry {
			continue
		}
		switch {
		case days < 0:
			result.ExpiredLicenses++
		case days <= 30:
			result.ExpiringIn30Days++
			result.ExpiringIn60Days++
			result.ExpiringIn90Days++
		case days <= 60:
			result.ExpiringIn60Days++
			result.ExpiringIn90Days++
		case days <= 90:
			result.ExpiringIn90Days++
		}
	}

	for currency, amount := range costs {
		result.CostsByCurrency = append(result.CostsByCurrency, CostByCurrency{Currency: currency, Amount: amount})
	}
	sort.Slice(result.CostsByCurrency, func(i, j int) bool {
		return result.CostsByCurrency[i].Currency < result.CostsByCurrency[j].Currency
	})
	return result, nil
}

func (s *Service) LicenseAlerts(ctx context.Context, expiryWindow int) ([]LicenseAlert, error) {
	if expiryWindow != 30 && expiryWindow != 60 && expiryWindow != 90 {
		return nil, ErrInvalidExpiryWindow
	}
	items, err := s.licenses.List(ctx)
	if err != nil {
		return nil, err
	}
	today := startOfDay(s.now())
	alerts := make([]LicenseAlert, 0)
	for _, item := range items {
		alert, include := buildAlert(today, expiryWindow, item)
		if include {
			alerts = append(alerts, alert)
		}
	}
	sort.Slice(alerts, func(i, j int) bool {
		leftPriority := severityPriority(alerts[i].Severity)
		rightPriority := severityPriority(alerts[j].Severity)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if alerts[i].ExpiresAt != alerts[j].ExpiresAt {
			if alerts[i].ExpiresAt == "" {
				return false
			}
			if alerts[j].ExpiresAt == "" {
				return true
			}
			return alerts[i].ExpiresAt < alerts[j].ExpiresAt
		}
		return alerts[i].LicenseName < alerts[j].LicenseName
	})
	return alerts, nil
}

func buildAlert(today time.Time, expiryWindow int, item licenses.License) (LicenseAlert, bool) {
	alert := LicenseAlert{
		LicenseID:          item.ID,
		LicenseName:        item.Name,
		LicenseType:        item.LicenseType,
		ExpiresAt:          item.ExpiresAt,
		SeatCount:          item.SeatCount,
		UsedSeats:          item.UsedSeats,
		AvailableSeats:     max(item.SeatCount-item.UsedSeats, 0),
		UtilizationPercent: utilizationPercent(item),
		AlertTypes:         make([]string, 0),
	}
	days, hasExpiry := daysUntilExpiry(today, item.ExpiresAt)
	if hasExpiry {
		alert.DaysUntilExpiry = &days
		if days < 0 {
			alert.AlertTypes = append(alert.AlertTypes, "expired")
		} else if days <= expiryWindow {
			alert.AlertTypes = append(alert.AlertTypes, "expiring")
		}
	}
	if item.UsedSeats >= item.SeatCount {
		alert.AlertTypes = append(alert.AlertTypes, "exhausted")
	} else if alert.UtilizationPercent >= HighUsageThreshold {
		alert.AlertTypes = append(alert.AlertTypes, "high_usage")
	}
	if len(alert.AlertTypes) == 0 {
		return LicenseAlert{}, false
	}
	alert.Severity = "info"
	for _, alertType := range alert.AlertTypes {
		if alertType == "expired" || alertType == "exhausted" {
			alert.Severity = "critical"
			break
		}
		if alertType == "high_usage" || (alertType == "expiring" && days <= 30) {
			alert.Severity = "warning"
		}
	}
	return alert, true
}

func utilizationPercent(item licenses.License) float64 {
	if item.SeatCount <= 0 {
		return 0
	}
	value := float64(item.UsedSeats) / float64(item.SeatCount) * 100
	return math.Round(value*100) / 100
}

func daysUntilExpiry(today time.Time, expiresAt string) (int, bool) {
	if expiresAt == "" {
		return 0, false
	}
	expiry, err := time.ParseInLocation("2006-01-02", expiresAt, today.Location())
	if err != nil {
		return 0, false
	}
	return int(expiry.Sub(today).Hours() / 24), true
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func severityPriority(severity string) int {
	switch severity {
	case "critical":
		return 0
	case "warning":
		return 1
	default:
		return 2
	}
}
