package com.nartueih.licensemanager.data.maintenance

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

@Serializable
internal data class MaintenanceRequestListDto(
    val items: List<MaintenanceRequestDto>,
    val total: Int,
    @SerialName("open_count") val openCount: Int = 0,
)

@Serializable
internal data class MaintenanceRequestDto(
    val id: String,
    @SerialName("requester_name") val requesterName: String,
    @SerialName("device_id") val deviceId: String,
    @SerialName("device_asset_code") val deviceAssetCode: String,
    @SerialName("device_serial_number") val deviceSerialNumber: String? = null,
    @SerialName("device_name") val deviceName: String,
    @SerialName("device_type") val deviceType: String,
    @SerialName("device_manufacturer") val deviceManufacturer: String? = null,
    @SerialName("device_model") val deviceModel: String? = null,
    @SerialName("device_purchased_at") val devicePurchasedAt: String? = null,
    @SerialName("device_warranty_expires_at") val deviceWarrantyExpiresAt: String? = null,
    val category: String,
    val priority: String,
    val title: String,
    val description: String,
    val status: String,
    @SerialName("assigned_to_name") val assignedToName: String? = null,
    @SerialName("response_note") val responseNote: String? = null,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    @SerialName("accepted_at") val acceptedAt: String? = null,
    @SerialName("completed_at") val completedAt: String? = null,
    @SerialName("rejected_at") val rejectedAt: String? = null,
    @SerialName("cancelled_at") val cancelledAt: String? = null,
)

@Serializable
internal data class CreateMaintenanceRequestDto(
    @SerialName("device_id") val deviceId: String,
    val category: String,
    val priority: String,
    val title: String,
    val description: String,
)

@Serializable
internal data class MaintenanceErrorDto(
    val error: String? = null,
    val code: String? = null,
)

internal interface MaintenanceRequestApi {
    @GET("me/maintenance-requests")
    suspend fun listMine(): MaintenanceRequestListDto

    @POST("me/maintenance-requests")
    suspend fun create(@Body input: CreateMaintenanceRequestDto): MaintenanceRequestDto

    @POST("me/maintenance-requests/{id}/cancel")
    suspend fun cancel(@Path("id") requestId: String): MaintenanceRequestDto
}

internal fun createMaintenanceRequestApi(
    baseUrl: String,
    client: OkHttpClient,
): MaintenanceRequestApi {
    val json = Json { ignoreUnknownKeys = true }
    return Retrofit.Builder()
        .baseUrl(baseUrl)
        .client(client)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()
        .create(MaintenanceRequestApi::class.java)
}
