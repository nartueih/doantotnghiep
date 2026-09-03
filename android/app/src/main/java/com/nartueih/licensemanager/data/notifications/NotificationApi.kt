package com.nartueih.licensemanager.data.notifications

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import retrofit2.http.GET
import retrofit2.http.PATCH
import retrofit2.http.Path

@Serializable
internal data class NotificationListDto(
    val items: List<NotificationDto>,
    val total: Int,
    @SerialName("unread_count") val unreadCount: Int,
)

@Serializable
internal data class NotificationDto(
    val id: String,
    val type: String,
    val title: String,
    val message: String,
    @SerialName("entity_type") val entityType: String,
    @SerialName("entity_id") val entityId: String,
    @SerialName("created_at") val createdAt: String,
    @SerialName("read_at") val readAt: String? = null,
)

@Serializable
internal data class MarkAllNotificationsDto(
    val updated: Int,
)

internal interface NotificationApi {
    @GET("me/notifications")
    suspend fun list(): NotificationListDto

    @PATCH("me/notifications/{id}/read")
    suspend fun markRead(@Path("id") notificationId: String): NotificationDto

    @PATCH("me/notifications/read-all")
    suspend fun markAllRead(): MarkAllNotificationsDto
}

internal fun createNotificationApi(
    baseUrl: String,
    client: OkHttpClient,
): NotificationApi {
    val json = Json { ignoreUnknownKeys = true }
    return Retrofit.Builder()
        .baseUrl(baseUrl)
        .client(client)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()
        .create(NotificationApi::class.java)
}
