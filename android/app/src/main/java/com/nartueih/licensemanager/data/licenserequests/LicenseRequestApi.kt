package com.nartueih.licensemanager.data.licenserequests

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.Path

@Serializable
internal data class LicenseRequestListDto(
    val items: List<LicenseRequestDto>,
    val total: Int,
)

@Serializable
internal data class RequestableSoftwareListDto(
    val items: List<RequestableSoftwareDto>,
    val total: Int,
)

@Serializable
internal data class RequestableSoftwareDto(
    val id: String,
    val name: String,
    val publisher: String,
    val version: String = "",
    val description: String = "",
)

@Serializable
internal data class LicenseRequestDto(
    val id: String,
    @SerialName("software_product_id") val softwareProductId: String,
    @SerialName("software_product_name") val softwareProductName: String,
    val priority: String,
    val reason: String,
    val status: String,
    @SerialName("selected_license_name") val selectedLicenseName: String? = null,
    @SerialName("assignment_id") val assignmentId: String? = null,
    @SerialName("reviewed_by_name") val reviewedByName: String? = null,
    @SerialName("decision_reason") val decisionReason: String? = null,
    @SerialName("response_note") val responseNote: String? = null,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    @SerialName("reviewed_at") val reviewedAt: String? = null,
    @SerialName("cancelled_at") val cancelledAt: String? = null,
)

@Serializable
internal data class CreateLicenseRequestDto(
    @SerialName("software_product_id") val softwareProductId: String,
    val priority: String,
    val reason: String,
)

@Serializable
internal data class LicenseRequestErrorDto(
    val error: String? = null,
    val code: String? = null,
)

internal interface LicenseRequestApi {
    @GET("me/license-requests")
    suspend fun listMine(): LicenseRequestListDto

    @GET("me/requestable-software")
    suspend fun requestableSoftware(): RequestableSoftwareListDto

    @POST("me/license-requests")
    suspend fun create(@Body input: CreateLicenseRequestDto): LicenseRequestDto

    @PATCH("me/license-requests/{id}/cancel")
    suspend fun cancel(@Path("id") requestId: String): LicenseRequestDto
}

internal fun createLicenseRequestApi(
    baseUrl: String,
    client: OkHttpClient,
): LicenseRequestApi {
    val json = Json { ignoreUnknownKeys = true }
    return Retrofit.Builder()
        .baseUrl(baseUrl)
        .client(client)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()
        .create(LicenseRequestApi::class.java)
}
