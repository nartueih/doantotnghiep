package com.nartueih.licensemanager.data.devices

import java.io.IOException
import okhttp3.OkHttpClient
import retrofit2.HttpException

internal class RemoteEmployeeDeviceRepository(
    private val api: EmployeeDeviceApi,
) : EmployeeDeviceRepository {
    override suspend fun list(): DeviceListOutcome = try {
        DeviceListOutcome.Success(api.list().items.map(EmployeeDeviceDto::toDomain))
    } catch (_: IOException) {
        DeviceListOutcome.ConnectionError
    } catch (_: HttpException) {
        DeviceListOutcome.ServerError
    }
}

fun createEmployeeDeviceRepository(
    baseUrl: String,
    client: OkHttpClient,
): EmployeeDeviceRepository = RemoteEmployeeDeviceRepository(
    api = createEmployeeDeviceApi(baseUrl, client),
)

private fun EmployeeDeviceDto.toDomain() = EmployeeDevice(
    id = id,
    assetCode = assetCode,
    serialNumber = serialNumber,
    name = name,
    deviceType = deviceType,
    manufacturer = manufacturer,
    model = model,
    status = status,
    purchasedAt = purchasedAt,
    warrantyExpiresAt = warrantyExpiresAt,
)
