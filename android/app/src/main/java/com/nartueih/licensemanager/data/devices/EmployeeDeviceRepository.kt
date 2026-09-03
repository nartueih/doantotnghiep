package com.nartueih.licensemanager.data.devices

data class EmployeeDevice(
    val id: String,
    val assetCode: String,
    val serialNumber: String?,
    val name: String,
    val deviceType: String,
    val manufacturer: String?,
    val model: String?,
    val status: String,
    val purchasedAt: String?,
    val warrantyExpiresAt: String?,
)

sealed interface DeviceListOutcome {
    data class Success(val items: List<EmployeeDevice>) : DeviceListOutcome
    data object ConnectionError : DeviceListOutcome
    data object ServerError : DeviceListOutcome
}

interface EmployeeDeviceRepository {
    suspend fun list(): DeviceListOutcome
}
