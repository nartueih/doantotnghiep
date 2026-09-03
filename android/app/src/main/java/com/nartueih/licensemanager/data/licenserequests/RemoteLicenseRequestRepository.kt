package com.nartueih.licensemanager.data.licenserequests

import java.io.IOException
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import retrofit2.HttpException

private val licenseRequestErrorJson = Json { ignoreUnknownKeys = true }

internal class RemoteLicenseRequestRepository(
    private val api: LicenseRequestApi,
) : LicenseRequestRepository {
    override suspend fun load(): LicenseRequestLoadOutcome = try {
        val requests = api.listMine()
        val software = api.requestableSoftware()
        LicenseRequestLoadOutcome.Success(
            items = requests.items.map(LicenseRequestDto::toDomain),
            software = software.items.map(RequestableSoftwareDto::toDomain),
        )
    } catch (_: IOException) {
        LicenseRequestLoadOutcome.ConnectionError
    } catch (_: HttpException) {
        LicenseRequestLoadOutcome.ServerError
    } catch (_: SerializationException) {
        LicenseRequestLoadOutcome.ServerError
    }

    override suspend fun create(input: CreateLicenseRequestInput): LicenseRequestMutationOutcome = mutate {
        api.create(
            CreateLicenseRequestDto(
                softwareProductId = input.softwareProductId,
                priority = input.priority,
                reason = input.reason,
            ),
        )
    }

    override suspend fun cancel(requestId: String): LicenseRequestMutationOutcome = mutate {
        api.cancel(requestId)
    }

    private suspend fun mutate(operation: suspend () -> LicenseRequestDto): LicenseRequestMutationOutcome = try {
        LicenseRequestMutationOutcome.Success(operation().toDomain())
    } catch (_: IOException) {
        LicenseRequestMutationOutcome.ConnectionError
    } catch (exception: HttpException) {
        when (exception.errorCode()) {
            "pending_request_exists" -> LicenseRequestMutationOutcome.PendingRequestExists
            "request_not_pending" -> LicenseRequestMutationOutcome.InvalidState
            else -> LicenseRequestMutationOutcome.ServerError
        }
    } catch (_: SerializationException) {
        LicenseRequestMutationOutcome.ServerError
    }
}

fun createLicenseRequestRepository(
    baseUrl: String,
    client: OkHttpClient,
): LicenseRequestRepository = RemoteLicenseRequestRepository(
    createLicenseRequestApi(baseUrl, client),
)

private fun HttpException.errorCode(): String? = runCatching {
    licenseRequestErrorJson.decodeFromString<LicenseRequestErrorDto>(response()?.errorBody()?.string().orEmpty()).code
}.getOrNull()

private fun RequestableSoftwareDto.toDomain() = RequestableSoftware(id, name, publisher, version, description)

private fun LicenseRequestDto.toDomain() = LicenseRequest(
    id = id,
    softwareProductId = softwareProductId,
    softwareProductName = softwareProductName,
    priority = priority,
    reason = reason,
    status = status,
    selectedLicenseName = selectedLicenseName,
    assignmentId = assignmentId,
    reviewedByName = reviewedByName,
    decisionReason = decisionReason,
    responseNote = responseNote,
    createdAt = createdAt,
    updatedAt = updatedAt,
    reviewedAt = reviewedAt,
    cancelledAt = cancelledAt,
)
