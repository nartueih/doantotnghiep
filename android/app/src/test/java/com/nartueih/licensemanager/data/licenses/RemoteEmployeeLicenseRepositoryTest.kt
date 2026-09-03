package com.nartueih.licensemanager.data.licenses

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test

class RemoteEmployeeLicenseRepositoryTest {
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
    fun listMapsAssignedLicenseFromPersonalEndpoint() = runTest {
        server.enqueue(jsonResponse(LICENSE_LIST_JSON))
        val repository = repository()

        val outcome = repository.list()

        assertEquals(
            LicenseListOutcome.Success(
                listOf(
                    EmployeeLicense(
                        assignmentId = "assignment-1",
                        licenseId = "license-1",
                        licenseName = "Adobe Creative Cloud",
                        licenseType = "subscription",
                        assignmentSource = "device",
                        deviceAssetCode = "LAP-001",
                        assignedAt = "2026-08-20T08:30:00Z",
                        expiresAt = "2026-12-31",
                        lifecycleStatus = "active",
                        notes = "Dùng cho thiết kế",
                        canViewKey = true,
                    ),
                ),
            ),
            outcome,
        )
        val request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/me/licenses", request.path)
    }

    @Test
    fun revealKeyMapsValueFromAssignmentEndpoint() = runTest {
        server.enqueue(
            jsonResponse(
                """
                {
                  "assignment_id":"assignment-1",
                  "license_id":"license-1",
                  "license_name":"Adobe Creative Cloud",
                  "license_key":"AAAA-BBBB-CCCC"
                }
                """.trimIndent(),
            ),
        )
        val repository = repository()

        val outcome = repository.revealKey("assignment-1")

        assertEquals(
            LicenseKeyOutcome.Success(
                RevealedLicenseKey(
                    assignmentId = "assignment-1",
                    licenseName = "Adobe Creative Cloud",
                    licenseKey = "AAAA-BBBB-CCCC",
                ),
            ),
            outcome,
        )
        assertEquals("/api/v1/me/licenses/assignment-1/key", server.takeRequest().path)
    }

    @Test
    fun revealKeyMapsForbiddenAndUnavailableResponses() = runTest {
        server.enqueue(MockResponse().setResponseCode(403).setBody("""{"error":"forbidden"}"""))
        server.enqueue(MockResponse().setResponseCode(409).setBody("""{"error":"unavailable"}"""))
        val repository = repository()

        assertEquals(LicenseKeyOutcome.NotAllowed, repository.revealKey("assignment-1"))
        assertEquals(LicenseKeyOutcome.Unavailable, repository.revealKey("assignment-2"))
    }

    @Test
    fun listMapsServerFailure() = runTest {
        server.enqueue(MockResponse().setResponseCode(500).setBody("""{"error":"internal"}"""))

        assertEquals(LicenseListOutcome.ServerError, repository().list())
    }

    private fun repository() = RemoteEmployeeLicenseRepository(
        api = createEmployeeLicenseApi(
            baseUrl = server.url("/api/v1/").toString(),
            client = OkHttpClient(),
        ),
    )

    private fun jsonResponse(body: String) = MockResponse()
        .setResponseCode(200)
        .setHeader("Content-Type", "application/json")
        .setBody(body)

    private companion object {
        val LICENSE_LIST_JSON =
            """
            {
              "items":[
                {
                  "assignment_id":"assignment-1",
                  "license_id":"license-1",
                  "license_name":"Adobe Creative Cloud",
                  "software_product_id":"product-1",
                  "license_type":"subscription",
                  "assignment_source":"device",
                  "device_id":"device-1",
                  "device_asset_code":"LAP-001",
                  "assigned_at":"2026-08-20T08:30:00Z",
                  "expires_at":"2026-12-31",
                  "lifecycle_status":"active",
                  "notes":"Dùng cho thiết kế",
                  "can_view_key":true
                }
              ],
              "total":1
            }
            """.trimIndent()
    }
}
