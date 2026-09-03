package com.nartueih.licensemanager.data.devices

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import retrofit2.http.GET

@Serializable
internal data class EmployeeDeviceListDto(
    val items: List<EmployeeDeviceDto>,
    val total: Int,
)

@Serializable
internal data class EmployeeDeviceDto(
    val id: String,
    @SerialName("asset_code") val assetCode: String,
    @SerialName("serial_number") val serialNumber: String? = null,
    val name: String,
    @SerialName("device_type") val deviceType: String,
    val manufacturer: String? = null,
    val model: String? = null,
    val status: String,
    @SerialName("purchased_at") val purchasedAt: String? = null,
    @SerialName("warranty_expires_at") val warrantyExpiresAt: String? = null,
)

internal interface EmployeeDeviceApi {
    @GET("me/devices")
    suspend fun list(): EmployeeDeviceListDto
}

internal fun createEmployeeDeviceApi(
    baseUrl: String,
    client: OkHttpClient,
): EmployeeDeviceApi {
    val json = Json { ignoreUnknownKeys = true }
    return Retrofit.Builder()
        .baseUrl(baseUrl)
        .client(client)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()
        .create(EmployeeDeviceApi::class.java)
}
