package com.nartueih.licensemanager.data.licenserequests

data class RequestableSoftware(
    val id: String,
    val name: String,
    val publisher: String,
    val version: String,
    val description: String,
)

data class LicenseRequest(
    val id: String,
    val softwareProductId: String,
    val softwareProductName: String,
    val priority: String,
    val reason: String,
    val status: String,
    val selectedLicenseName: String?,
    val assignmentId: String?,
    val reviewedByName: String?,
    val decisionReason: String?,
    val responseNote: String?,
    val createdAt: String,
    val updatedAt: String,
    val reviewedAt: String?,
    val cancelledAt: String?,
)

data class CreateLicenseRequestInput(
    val softwareProductId: String,
    val priority: String,
    val reason: String,
)

sealed interface LicenseRequestLoadOutcome {
    data class Success(
        val items: List<LicenseRequest>,
        val software: List<RequestableSoftware>,
    ) : LicenseRequestLoadOutcome

    data object ConnectionError : LicenseRequestLoadOutcome
    data object ServerError : LicenseRequestLoadOutcome
}

sealed interface LicenseRequestMutationOutcome {
    data class Success(val item: LicenseRequest) : LicenseRequestMutationOutcome
    data object PendingRequestExists : LicenseRequestMutationOutcome
    data object InvalidState : LicenseRequestMutationOutcome
    data object ConnectionError : LicenseRequestMutationOutcome
    data object ServerError : LicenseRequestMutationOutcome
}

interface LicenseRequestRepository {
    suspend fun load(): LicenseRequestLoadOutcome
    suspend fun create(input: CreateLicenseRequestInput): LicenseRequestMutationOutcome
    suspend fun cancel(requestId: String): LicenseRequestMutationOutcome
}
