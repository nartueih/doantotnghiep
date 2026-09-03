package com.nartueih.licensemanager.data.notifications

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class RemoteNotificationRepositoryTest {
    private lateinit var server: MockWebServer

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun listMapsUnreadCountAndNotificationTarget() = runTest {
        server.enqueue(
            jsonResponse(
                """{"items":[${notificationPayload()}],"total":1,"unread_count":1}""",
            ),
        )

        val outcome = repository().list()

        assertTrue(outcome is NotificationListOutcome.Success)
        val success = outcome as NotificationListOutcome.Success
        assertEquals(1, success.unreadCount)
        assertEquals("license_request", success.items.single().entityType)
        assertNull(success.items.single().readAt)
        assertEquals("/api/v1/me/notifications", server.takeRequest().path)
    }

    @Test
    fun markReadUsesPatchAndReturnsUpdatedItem() = runTest {
        server.enqueue(jsonResponse(notificationPayload(readAt = "2026-09-01T10:00:00Z")))

        val outcome = repository().markRead("notification-1")

        assertTrue(outcome is NotificationMutationOutcome.Success)
        assertEquals("2026-09-01T10:00:00Z", (outcome as NotificationMutationOutcome.Success).item.readAt)
        val request = server.takeRequest()
        assertEquals("PATCH", request.method)
        assertEquals("/api/v1/me/notifications/notification-1/read", request.path)
    }

    @Test
    fun markAllReadUsesPatchAndMapsUpdatedCount() = runTest {
        server.enqueue(jsonResponse("""{"updated":3}"""))

        val outcome = repository().markAllRead()

        assertEquals(NotificationMarkAllOutcome.Success(updated = 3), outcome)
        val request = server.takeRequest()
        assertEquals("PATCH", request.method)
        assertEquals("/api/v1/me/notifications/read-all", request.path)
    }

    @Test
    fun listMapsServerFailure() = runTest {
        server.enqueue(MockResponse().setResponseCode(500).setBody("""{"error":"internal"}"""))

        assertEquals(NotificationListOutcome.ServerError, repository().list())
    }

    private fun repository() = RemoteNotificationRepository(
        createNotificationApi(server.url("/api/v1/").toString(), OkHttpClient()),
    )

    private fun jsonResponse(body: String) = MockResponse()
        .setResponseCode(200)
        .setHeader("Content-Type", "application/json")
        .setBody(body)

    private fun notificationPayload(readAt: String? = null): String {
        val readAtJson = readAt?.let { ",\"read_at\":\"$it\"" }.orEmpty()
        return """
            {
              "id":"notification-1","user_id":"user-1","type":"license_request_approved",
              "title":"Yêu cầu license đã được duyệt","message":"Adobe đã được cấp cho bạn.",
              "entity_type":"license_request","entity_id":"request-1",
              "created_at":"2026-09-01T09:00:00Z"$readAtJson
            }
        """.trimIndent()
    }
}
