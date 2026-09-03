package com.nartueih.licensemanager.data.dashboard

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import retrofit2.http.GET

@Serializable
internal data class PersonalListSummaryDto(
    val total: Int,
)

@Serializable
internal data class NotificationSummaryDto(
    val total: Int,
    @SerialName("unread_count") val unreadCount: Int,
)

internal interface EmployeeDashboardApi {
    @GET("me/licenses")
    suspend fun licenses(): PersonalListSummaryDto

    @GET("me/devices")
    suspend fun devices(): PersonalListSummaryDto

    @GET("me/notifications")
    suspend fun notifications(): NotificationSummaryDto
}

internal fun createEmployeeDashboardApi(
    baseUrl: String,
    client: OkHttpClient,
): EmployeeDashboardApi {
    val json = Json { ignoreUnknownKeys = true }
    return Retrofit.Builder()
        .baseUrl(baseUrl)
        .client(client)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()
        .create(EmployeeDashboardApi::class.java)
}
