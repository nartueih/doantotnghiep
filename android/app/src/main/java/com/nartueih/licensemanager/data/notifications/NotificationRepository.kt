package com.nartueih.licensemanager.data.notifications

data class EmployeeNotification(
    val id: String,
    val type: String,
    val title: String,
    val message: String,
    val entityType: String,
    val entityId: String,
    val createdAt: String,
    val readAt: String?,
)

sealed interface NotificationListOutcome {
    data class Success(
        val items: List<EmployeeNotification>,
        val unreadCount: Int,
    ) : NotificationListOutcome

    data object ConnectionError : NotificationListOutcome
    data object ServerError : NotificationListOutcome
}

sealed interface NotificationMutationOutcome {
    data class Success(val item: EmployeeNotification) : NotificationMutationOutcome
    data object ConnectionError : NotificationMutationOutcome
    data object ServerError : NotificationMutationOutcome
}

sealed interface NotificationMarkAllOutcome {
    data class Success(val updated: Int) : NotificationMarkAllOutcome
    data object ConnectionError : NotificationMarkAllOutcome
    data object ServerError : NotificationMarkAllOutcome
}

interface NotificationRepository {
    suspend fun list(): NotificationListOutcome
    suspend fun markRead(notificationId: String): NotificationMutationOutcome
    suspend fun markAllRead(): NotificationMarkAllOutcome
}
