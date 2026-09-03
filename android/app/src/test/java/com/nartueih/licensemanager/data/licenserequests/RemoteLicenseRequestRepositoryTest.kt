package com.nartueih.licensemanager.data.licenserequests

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class RemoteLicenseRequestRepositoryTest {
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
    fun loadMapsPersonalRequestsAndRequestableSoftware() = runTest {
        server.enqueue(jsonResponse("""{"items":[${requestPayload()}],"total":1}"""))
        server.enqueue(
            jsonResponse(
                """{"items":[{"id":"product-1","name":"Adobe Creative Cloud","publisher":"Adobe","version":"2026","description":"Thiết kế"}],"total":1}""",
            ),
        )

        val outcome = repository().load()

        assertTrue(outcome is LicenseRequestLoadOutcome.Success)
        val success = outcome as LicenseRequestLoadOutcome.Success
        assertEquals("Adobe Creative Cloud", success.items.single().softwareProductName)
        assertEquals("Development Admin", success.items.single().reviewedByName)
        assertEquals("Adobe", success.software.single().publisher)
        assertEquals("/api/v1/me/license-requests", server.takeRequest().path)
        assertEquals("/api/v1/me/requestable-software", server.takeRequest().path)
    }

    @Test
    fun createSendsExpectedPayload() = runTest {
        server.enqueue(jsonResponse(requestPayload(status = "pending"), responseCode = 201))

        val outcome = repository().create(
            CreateLicenseRequestInput("product-1", "urgent", "Cần phục vụ thiết kế."),
        )

        assertTrue(outcome is LicenseRequestMutationOutcome.Success)
        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/me/license-requests", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"software_product_id\":\"product-1\""))
        assertTrue(body.contains("\"priority\":\"urgent\""))
    }

    @Test
    fun createMapsPendingDuplicateConflict() = runTest {
        server.enqueue(
            MockResponse().setResponseCode(409)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error":"duplicate","code":"pending_request_exists"}"""),
        )

        val outcome = repository().create(CreateLicenseRequestInput("product-1", "normal", "Cần dùng"))

        assertEquals(LicenseRequestMutationOutcome.PendingRequestExists, outcome)
    }

    @Test
    fun cancelUsesPatchAndMapsUpdatedRequest() = runTest {
        server.enqueue(jsonResponse(requestPayload(status = "cancelled")))

        val outcome = repository().cancel("request-1")

        assertTrue(outcome is LicenseRequestMutationOutcome.Success)
        assertEquals("cancelled", (outcome as LicenseRequestMutationOutcome.Success).item.status)
        val request = server.takeRequest()
        assertEquals("PATCH", request.method)
        assertEquals("/api/v1/me/license-requests/request-1/cancel", request.path)
    }

    private fun repository() = RemoteLicenseRequestRepository(
        createLicenseRequestApi(server.url("/api/v1/").toString(), OkHttpClient()),
    )

    private fun jsonResponse(body: String, responseCode: Int = 200) = MockResponse()
        .setResponseCode(responseCode)
        .setHeader("Content-Type", "application/json")
        .setBody(body)

    private fun requestPayload(status: String = "approved") = """
        {
          "id":"request-1","requester_id":"user-1","requester_name":"Nguyễn Hoàng Anh",
          "software_product_id":"product-1","software_product_name":"Adobe Creative Cloud",
          "priority":"urgent","reason":"Cần phục vụ thiết kế.","status":"$status",
          "selected_license_id":"license-1","selected_license_name":"Adobe Creative Cloud All Apps",
          "assignment_id":"assignment-1","reviewed_by":"admin-1","reviewed_by_name":"Development Admin",
          "response_note":"Đã cấp license","created_at":"2026-09-01T08:00:00Z",
          "updated_at":"2026-09-01T09:00:00Z","reviewed_at":"2026-09-01T09:00:00Z"
        }
    """.trimIndent()
}
