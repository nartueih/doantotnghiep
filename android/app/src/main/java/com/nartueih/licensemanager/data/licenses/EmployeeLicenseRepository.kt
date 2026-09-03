package com.nartueih.licensemanager.data.licenses

data class EmployeeLicense(
    val assignmentId: String,
    val licenseId: String,
    val licenseName: String,
    val licenseType: String,
    val assignmentSource: String,
    val deviceAssetCode: String?,
    val assignedAt: String,
    val expiresAt: String?,
    val lifecycleStatus: String,
    val notes: String?,
    val canViewKey: Boolean,
)

data class RevealedLicenseKey(
    val assignmentId: String,
    val licenseName: String,
    val licenseKey: String,
)

sealed interface LicenseListOutcome {
    data class Success(val items: List<EmployeeLicense>) : LicenseListOutcome
    data object ConnectionError : LicenseListOutcome
    data object ServerError : LicenseListOutcome
}

sealed interface LicenseKeyOutcome {
    data class Success(val result: RevealedLicenseKey) : LicenseKeyOutcome
    data object NotAllowed : LicenseKeyOutcome
    data object Unavailable : LicenseKeyOutcome
    data object ConnectionError : LicenseKeyOutcome
    data object ServerError : LicenseKeyOutcome
}

interface EmployeeLicenseRepository {
    suspend fun list(): LicenseListOutcome
    suspend fun revealKey(assignmentId: String): LicenseKeyOutcome
}
