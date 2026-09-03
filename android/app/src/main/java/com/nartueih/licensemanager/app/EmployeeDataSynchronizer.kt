package com.nartueih.licensemanager.app

import com.nartueih.licensemanager.feature.requests.RequestSection

enum class EmployeeSyncDestination {
    OVERVIEW,
    LICENSES,
    DEVICES,
    REQUESTS,
    PROFILE,
}

class EmployeeDataSynchronizer(
    private val syncOverview: () -> Unit,
    private val syncLicenses: () -> Unit,
    private val syncDevices: () -> Unit,
    private val syncLicenseRequests: () -> Unit,
    private val syncMaintenanceRequests: () -> Unit,
    private val syncNotifications: () -> Unit,
) {
    fun sync(destination: EmployeeSyncDestination, requestSection: RequestSection) {
        syncNotifications()
        when (destination) {
            EmployeeSyncDestination.OVERVIEW -> syncOverview()
            EmployeeSyncDestination.LICENSES -> syncLicenses()
            EmployeeSyncDestination.DEVICES -> syncDevices()
            EmployeeSyncDestination.REQUESTS -> when (requestSection) {
                RequestSection.LICENSE -> syncLicenseRequests()
                RequestSection.MAINTENANCE -> syncMaintenanceRequests()
            }
            EmployeeSyncDestination.PROFILE -> Unit
        }
    }
}
