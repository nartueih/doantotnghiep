package com.nartueih.licensemanager.app

import com.nartueih.licensemanager.feature.requests.RequestSection
import org.junit.Assert.assertEquals
import org.junit.Test

class EmployeeDataSynchronizerTest {
    @Test
    fun syncRefreshesNotificationsAndOnlyTheVisibleDestination() {
        val calls = mutableListOf<String>()
        val synchronizer = EmployeeDataSynchronizer(
            syncOverview = { calls += "overview" },
            syncLicenses = { calls += "licenses" },
            syncDevices = { calls += "devices" },
            syncLicenseRequests = { calls += "license-requests" },
            syncMaintenanceRequests = { calls += "maintenance-requests" },
            syncNotifications = { calls += "notifications" },
        )

        synchronizer.sync(EmployeeSyncDestination.REQUESTS, RequestSection.MAINTENANCE)

        assertEquals(listOf("notifications", "maintenance-requests"), calls)
    }

    @Test
    fun syncUsesLicenseRequestSectionWhenItIsVisible() {
        val calls = mutableListOf<String>()
        val synchronizer = EmployeeDataSynchronizer(
            syncOverview = { calls += "overview" },
            syncLicenses = { calls += "licenses" },
            syncDevices = { calls += "devices" },
            syncLicenseRequests = { calls += "license-requests" },
            syncMaintenanceRequests = { calls += "maintenance-requests" },
            syncNotifications = { calls += "notifications" },
        )

        synchronizer.sync(EmployeeSyncDestination.REQUESTS, RequestSection.LICENSE)

        assertEquals(listOf("notifications", "license-requests"), calls)
    }
}
