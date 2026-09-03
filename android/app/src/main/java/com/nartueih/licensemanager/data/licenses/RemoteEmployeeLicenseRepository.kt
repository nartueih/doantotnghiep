package com.nartueih.licensemanager.data.licenses

import java.io.IOException
import okhttp3.OkHttpClient
import retrofit2.HttpException

internal class RemoteEmployeeLicenseRepository(
    private val api: EmployeeLicenseApi,
) : EmployeeLicenseRepository {
    override suspend fun list(): LicenseListOutcome = try {
        LicenseListOutcome.Success(api.list().items.map(EmployeeLicenseDto::toDomain))
    } catch (_: IOException) {
        LicenseListOutcome.ConnectionError
    } catch (_: HttpException) {
        LicenseListOutcome.ServerError
    }

    override suspend fun revealKey(assignmentId: String): LicenseKeyOutcome = try {
        LicenseKeyOutcome.Success(api.revealKey(assignmentId).toDomain())
    } catch (error: HttpException) {
        when (error.code()) {
            403 -> LicenseKeyOutcome.NotAllowed
            404, 409 -> LicenseKeyOutcome.Unavailable
            else -> LicenseKeyOutcome.ServerError
        }
    } catch (_: IOException) {
        LicenseKeyOutcome.ConnectionError
    }
}

fun createEmployeeLicenseRepository(
    baseUrl: String,
    client: OkHttpClient,
): EmployeeLicenseRepository = RemoteEmployeeLicenseRepository(
    api = createEmployeeLicenseApi(baseUrl, client),
)

private fun EmployeeLicenseDto.toDomain() = EmployeeLicense(
    assignmentId = assignmentId,
    licenseId = licenseId,
    licenseName = licenseName,
    licenseType = licenseType,
    assignmentSource = assignmentSource,
    deviceAssetCode = deviceAssetCode,
    assignedAt = assignedAt,
    expiresAt = expiresAt,
    lifecycleStatus = lifecycleStatus,
    notes = notes,
    canViewKey = canViewKey,
)

private fun RevealedLicenseKeyDto.toDomain() = RevealedLicenseKey(
    assignmentId = assignmentId,
    licenseName = licenseName,
    licenseKey = licenseKey,
)
