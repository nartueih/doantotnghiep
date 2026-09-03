package com.nartueih.licensemanager.feature.notifications

import com.nartueih.licensemanager.MainDispatcherRule
import com.nartueih.licensemanager.data.notifications.EmployeeNotification
import com.nartueih.licensemanager.data.notifications.NotificationListOutcome
import com.nartueih.licensemanager.data.notifications.NotificationMarkAllOutcome
import com.nartueih.licensemanager.data.notifications.NotificationMutationOutcome
import com.nartueih.licensemanager.data.notifications.NotificationRepository
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class NotificationViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    @Test
    fun initializationLoadsNotificationsAndUnreadBadge() = runTest {
        val item = notification()
        val viewModel = NotificationViewModel(
            FakeRepository(listOutcome = NotificationListOutcome.Success(listOf(item), unreadCount = 1)),
        )

        advanceUntilIdle()

        assertFalse(viewModel.uiState.value.isLoading)
        assertEquals(listOf(item), viewModel.uiState.value.items)
        assertEquals(1, viewModel.uiState.value.unreadCount)
    }

    @Test
    fun markingOneNotificationReadReplacesItemAndDecrementsBadge() = runTest {
        val unread = notification()
        val read = unread.copy(readAt = "2026-09-01T10:00:00Z")
        val repository = FakeRepository(
            listOutcome = NotificationListOutcome.Success(listOf(unread), 1),
            mutationOutcome = NotificationMutationOutcome.Success(read),
        )
        val viewModel = NotificationViewModel(repository)
        advanceUntilIdle()

        viewModel.markRead("notification-1")
        advanceUntilIdle()

        assertEquals("notification-1", repository.lastReadId)
        assertEquals(read, viewModel.uiState.value.items.single())
        assertEquals(0, viewModel.uiState.value.unreadCount)
        assertNull(viewModel.uiState.value.error)
    }

    @Test
    fun markingAlreadyReadNotificationDoesNotCallRepository() = runTest {
        val read = notification().copy(readAt = "2026-09-01T10:00:00Z")
        val repository = FakeRepository(listOutcome = NotificationListOutcome.Success(listOf(read), 0))
        val viewModel = NotificationViewModel(repository)
        advanceUntilIdle()

        viewModel.markRead("notification-1")
        advanceUntilIdle()

        assertNull(repository.lastReadId)
    }

    @Test
    fun markAllReadUpdatesEveryItemAndClearsBadge() = runTest {
        val item = notification()
        val repository = FakeRepository(
            listOutcome = NotificationListOutcome.Success(listOf(item), 1),
            markAllOutcome = NotificationMarkAllOutcome.Success(1),
        )
        val viewModel = NotificationViewModel(repository, now = { "2026-09-01T11:00:00Z" })
        advanceUntilIdle()

        viewModel.markAllRead()
        advanceUntilIdle()

        assertEquals("2026-09-01T11:00:00Z", viewModel.uiState.value.items.single().readAt)
        assertEquals(0, viewModel.uiState.value.unreadCount)
        assertFalse(viewModel.uiState.value.isMarkingAll)
    }

    private fun notification() = EmployeeNotification(
        id = "notification-1", type = "license_request_approved",
        title = "Yêu cầu license đã được duyệt", message = "Adobe đã được cấp cho bạn.",
        entityType = "license_request", entityId = "request-1",
        createdAt = "2026-09-01T09:00:00Z", readAt = null,
    )

    private class FakeRepository(
        private val listOutcome: NotificationListOutcome = NotificationListOutcome.Success(emptyList(), 0),
        private val mutationOutcome: NotificationMutationOutcome = NotificationMutationOutcome.ServerError,
        private val markAllOutcome: NotificationMarkAllOutcome = NotificationMarkAllOutcome.ServerError,
    ) : NotificationRepository {
        var lastReadId: String? = null

        override suspend fun list() = listOutcome
        override suspend fun markRead(notificationId: String): NotificationMutationOutcome {
            lastReadId = notificationId
            return mutationOutcome
        }
        override suspend fun markAllRead() = markAllOutcome
    }
}
