package com.nartueih.licensemanager.data.maintenance

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class RemoteMaintenanceRequestRepositoryTest {
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
    fun listMapsCompleteMaintenanceRequestSnapshot() = runTest {
        server.enqueue(jsonResponse(listPayload()))

        val outcome = repository().listMine()

        assertTrue(outcome is MaintenanceListOutcome.Success)
        val success = outcome as MaintenanceListOutcome.Success
        assertEquals(1, success.openCount)
        assertEquals("DEMO-LT-001", success.items.single().deviceAssetCode)
        assertEquals("DEMO-SEED-DELL-001", success.items.single().deviceSerialNumber)
        assertEquals("Development Admin", success.items.single().assignedToName)
        assertEquals("2028-01-10", success.items.single().deviceWarrantyExpiresAt)
        assertEquals("GET", server.takeRequest().method)
    }

    @Test
    fun createSendsTrimmedInputToEmployeeEndpoint() = runTest {
        server.enqueue(jsonResponse(requestPayload(), responseCode = 201))

        val outcome = repository().create(
            CreateMaintenanceRequestInput(
                deviceId = "device-1",
                category = "hardware",
                priority = "urgent",
                title = "Không nhận bàn phím",
                description = "Một số phím không hoạt động.",
            ),
        )

        assertTrue(outcome is MaintenanceMutationOutcome.Success)
        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/me/maintenance-requests", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"device_id\":\"device-1\""))
        assertTrue(body.contains("\"priority\":\"urgent\""))
        assertTrue(body.contains("\"title\":\"Không nhận bàn phím\""))
    }

    @Test
    fun createMapsOpenDuplicateConflict() = runTest {
        server.enqueue(
            MockResponse().setResponseCode(409)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error":"duplicate","code":"open_maintenance_request_exists"}"""),
        )

        val outcome = repository().create(
            CreateMaintenanceRequestInput("device-1", "hardware", "normal", "Lỗi", "Mô tả"),
        )

        assertEquals(MaintenanceMutationOutcome.OpenRequestExists, outcome)
    }

    @Test
    fun cancelUsesRequestSpecificEndpointAndReturnsUpdatedItem() = runTest {
        server.enqueue(jsonResponse(requestPayload(status = "cancelled")))

        val outcome = repository().cancel("request-1")

        assertTrue(outcome is MaintenanceMutationOutcome.Success)
        assertEquals("cancelled", (outcome as MaintenanceMutationOutcome.Success).item.status)
        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/me/maintenance-requests/request-1/cancel", request.path)
    }

    private fun repository() = RemoteMaintenanceRequestRepository(
        api = createMaintenanceRequestApi(
            baseUrl = server.url("/api/v1/").toString(),
            client = OkHttpClient(),
        ),
    )

    private fun jsonResponse(body: String, responseCode: Int = 200) = MockResponse()
        .setResponseCode(responseCode)
        .setHeader("Content-Type", "application/json")
        .setBody(body)

    private fun listPayload() = """
        {"items":[${requestPayload()}],"total":1,"open_count":1}
    """.trimIndent()

    private fun requestPayload(status: String = "in_progress") = """
        {
          "id":"request-1","requester_id":"user-1","requester_name":"Nguyễn Hoàng Anh",
          "device_id":"device-1","device_asset_code":"DEMO-LT-001",
          "device_serial_number":"DEMO-SEED-DELL-001","device_name":"Laptop Dell Latitude",
          "device_type":"laptop","device_manufacturer":"Dell","device_model":"Latitude 7450",
          "device_purchased_at":"2026-01-10","device_warranty_expires_at":"2028-01-10",
          "category":"hardware","priority":"urgent","title":"Không nhận bàn phím",
          "description":"Một số phím không hoạt động.","status":"$status",
          "assigned_to":"admin-1","assigned_to_name":"Development Admin",
          "response_note":"Đang kiểm tra","created_at":"2026-09-01T08:00:00Z",
          "updated_at":"2026-09-01T09:00:00Z","accepted_at":"2026-09-01T09:00:00Z"
        }
    """.trimIndent()
}
