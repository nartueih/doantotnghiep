package com.nartueih.licensemanager.data.licenses

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import retrofit2.http.GET
import retrofit2.http.Path

@Serializable
internal data class EmployeeLicenseListDto(
    val items: List<EmployeeLicenseDto>,
    val total: Int,
)

@Serializable
internal data class EmployeeLicenseDto(
    @SerialName("assignment_id") val assignmentId: String,
    @SerialName("license_id") val licenseId: String,
    @SerialName("license_name") val licenseName: String,
    @SerialName("software_product_id") val softwareProductId: String,
    @SerialName("license_type") val licenseType: String,
    @SerialName("assignment_source") val assignmentSource: String,
    @SerialName("device_id") val deviceId: String? = null,
    @SerialName("device_asset_code") val deviceAssetCode: String? = null,
    @SerialName("assigned_at") val assignedAt: String,
    @SerialName("expires_at") val expiresAt: String? = null,
    @SerialName("lifecycle_status") val lifecycleStatus: String,
    val notes: String? = null,
    @SerialName("can_view_key") val canViewKey: Boolean,
)

@Serializable
internal data class RevealedLicenseKeyDto(
    @SerialName("assignment_id") val assignmentId: String,
    @SerialName("license_id") val licenseId: String,
    @SerialName("license_name") val licenseName: String,
    @SerialName("license_key") val licenseKey: String,
)

internal interface EmployeeLicenseApi {
    @GET("me/licenses")
    suspend fun list(): EmployeeLicenseListDto

    @GET("me/licenses/{assignmentId}/key")
    suspend fun revealKey(@Path("assignmentId") assignmentId: String): RevealedLicenseKeyDto
}

internal fun createEmployeeLicenseApi(
    baseUrl: String,
    client: OkHttpClient,
): EmployeeLicenseApi {
    val json = Json { ignoreUnknownKeys = true }
    return Retrofit.Builder()
        .baseUrl(baseUrl)
        .client(client)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()
        .create(EmployeeLicenseApi::class.java)
}
