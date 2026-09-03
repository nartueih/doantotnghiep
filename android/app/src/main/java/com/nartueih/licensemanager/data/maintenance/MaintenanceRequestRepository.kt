package com.nartueih.licensemanager.data.maintenance

data class MaintenanceRequest(
    val id: String,
    val requesterName: String,
    val deviceId: String,
    val deviceAssetCode: String,
    val deviceSerialNumber: String?,
    val deviceName: String,
    val deviceType: String,
    val deviceManufacturer: String?,
    val deviceModel: String?,
    val devicePurchasedAt: String?,
    val deviceWarrantyExpiresAt: String?,
    val category: String,
    val priority: String,
    val title: String,
    val description: String,
    val status: String,
    val assignedToName: String?,
    val responseNote: String?,
    val createdAt: String,
    val updatedAt: String,
    val acceptedAt: String?,
    val completedAt: String?,
    val rejectedAt: String?,
    val cancelledAt: String?,
)

data class CreateMaintenanceRequestInput(
    val deviceId: String,
    val category: String,
    val priority: String,
    val title: String,
    val description: String,
)

sealed interface MaintenanceListOutcome {
    data class Success(
        val items: List<MaintenanceRequest>,
        val openCount: Int,
    ) : MaintenanceListOutcome

    data object ConnectionError : MaintenanceListOutcome
    data object ServerError : MaintenanceListOutcome
}

sealed interface MaintenanceMutationOutcome {
    data class Success(val item: MaintenanceRequest) : MaintenanceMutationOutcome
    data object OpenRequestExists : MaintenanceMutationOutcome
    data object InvalidState : MaintenanceMutationOutcome
    data object ConnectionError : MaintenanceMutationOutcome
    data object ServerError : MaintenanceMutationOutcome
}

interface MaintenanceRequestRepository {
    suspend fun listMine(): MaintenanceListOutcome
    suspend fun create(input: CreateMaintenanceRequestInput): MaintenanceMutationOutcome
    suspend fun cancel(requestId: String): MaintenanceMutationOutcome
}
