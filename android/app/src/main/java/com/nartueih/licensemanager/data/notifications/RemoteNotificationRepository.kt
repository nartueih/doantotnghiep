package com.nartueih.licensemanager.data.notifications

import java.io.IOException
import kotlinx.serialization.SerializationException
import okhttp3.OkHttpClient
import retrofit2.HttpException

internal class RemoteNotificationRepository(
    private val api: NotificationApi,
) : NotificationRepository {
    override suspend fun list(): NotificationListOutcome = try {
        val result = api.list()
        NotificationListOutcome.Success(result.items.map(NotificationDto::toDomain), result.unreadCount)
    } catch (_: IOException) {
        NotificationListOutcome.ConnectionError
    } catch (_: HttpException) {
        NotificationListOutcome.ServerError
    } catch (_: SerializationException) {
        NotificationListOutcome.ServerError
    }

    override suspend fun markRead(notificationId: String): NotificationMutationOutcome = try {
        NotificationMutationOutcome.Success(api.markRead(notificationId).toDomain())
    } catch (_: IOException) {
        NotificationMutationOutcome.ConnectionError
    } catch (_: HttpException) {
        NotificationMutationOutcome.ServerError
    } catch (_: SerializationException) {
        NotificationMutationOutcome.ServerError
    }

    override suspend fun markAllRead(): NotificationMarkAllOutcome = try {
        NotificationMarkAllOutcome.Success(api.markAllRead().updated)
    } catch (_: IOException) {
        NotificationMarkAllOutcome.ConnectionError
    } catch (_: HttpException) {
        NotificationMarkAllOutcome.ServerError
    } catch (_: SerializationException) {
        NotificationMarkAllOutcome.ServerError
    }
}

fun createNotificationRepository(
    baseUrl: String,
    client: OkHttpClient,
): NotificationRepository = RemoteNotificationRepository(
    createNotificationApi(baseUrl, client),
)

private fun NotificationDto.toDomain() = EmployeeNotification(
    id = id,
    type = type,
    title = title,
    message = message,
    entityType = entityType,
    entityId = entityId,
    createdAt = createdAt,
    readAt = readAt,
)
