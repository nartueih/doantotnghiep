package com.nartueih.licensemanager.data.maintenance

import java.io.IOException
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import retrofit2.HttpException

private val maintenanceErrorJson = Json { ignoreUnknownKeys = true }

internal class RemoteMaintenanceRequestRepository(
    private val api: MaintenanceRequestApi,
) : MaintenanceRequestRepository {
    override suspend fun listMine(): MaintenanceListOutcome = try {
        val result = api.listMine()
        MaintenanceListOutcome.Success(
            items = result.items.map(MaintenanceRequestDto::toDomain),
            openCount = result.openCount,
        )
    } catch (_: IOException) {
        MaintenanceListOutcome.ConnectionError
    } catch (_: HttpException) {
        MaintenanceListOutcome.ServerError
    } catch (_: SerializationException) {
        MaintenanceListOutcome.ServerError
    }

    override suspend fun create(input: CreateMaintenanceRequestInput): MaintenanceMutationOutcome = mutate {
        api.create(
            CreateMaintenanceRequestDto(
                deviceId = input.deviceId,
                category = input.category,
                priority = input.priority,
                title = input.title,
                description = input.description,
            ),
        )
    }

    override suspend fun cancel(requestId: String): MaintenanceMutationOutcome = mutate {
        api.cancel(requestId)
    }

    private suspend fun mutate(operation: suspend () -> MaintenanceRequestDto): MaintenanceMutationOutcome = try {
        MaintenanceMutationOutcome.Success(operation().toDomain())
    } catch (_: IOException) {
        MaintenanceMutationOutcome.ConnectionError
    } catch (exception: HttpException) {
        when (exception.errorCode()) {
            "open_maintenance_request_exists" -> MaintenanceMutationOutcome.OpenRequestExists
            "invalid_maintenance_state" -> MaintenanceMutationOutcome.InvalidState
            else -> MaintenanceMutationOutcome.ServerError
        }
    } catch (_: SerializationException) {
        MaintenanceMutationOutcome.ServerError
    }
}

fun createMaintenanceRequestRepository(
    baseUrl: String,
    client: OkHttpClient,
): MaintenanceRequestRepository = RemoteMaintenanceRequestRepository(
    createMaintenanceRequestApi(baseUrl, client),
)

private fun HttpException.errorCode(): String? = runCatching {
    val body = response()?.errorBody()?.string().orEmpty()
    maintenanceErrorJson.decodeFromString<MaintenanceErrorDto>(body).code
}.getOrNull()

private fun MaintenanceRequestDto.toDomain() = MaintenanceRequest(
    id = id,
    requesterName = requesterName,
    deviceId = deviceId,
    deviceAssetCode = deviceAssetCode,
    deviceSerialNumber = deviceSerialNumber,
    deviceName = deviceName,
    deviceType = deviceType,
    deviceManufacturer = deviceManufacturer,
    deviceModel = deviceModel,
    devicePurchasedAt = devicePurchasedAt,
    deviceWarrantyExpiresAt = deviceWarrantyExpiresAt,
    category = category,
    priority = priority,
    title = title,
    description = description,
    status = status,
    assignedToName = assignedToName,
    responseNote = responseNote,
    createdAt = createdAt,
    updatedAt = updatedAt,
    acceptedAt = acceptedAt,
    completedAt = completedAt,
    rejectedAt = rejectedAt,
    cancelledAt = cancelledAt,
)
