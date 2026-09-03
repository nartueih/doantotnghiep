package com.nartueih.licensemanager.data.devices

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test

class RemoteEmployeeDeviceRepositoryTest {
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
    fun listMapsCompleteAssignedDeviceFromPersonalEndpoint() = runTest {
        server.enqueue(
            jsonResponse(
                """
                {
                  "items":[
                    {
                      "id":"device-1",
                      "assigned_user_id":"user-1",
                      "assigned_user_name":"Nguyễn Hoàng Anh",
                      "asset_code":"DEMO-LT-001",
                      "serial_number":"DEMO-SEED-DELL-001",
                      "name":"Laptop Dell Latitude",
                      "device_type":"laptop",
                      "manufacturer":"Dell",
                      "model":"Latitude 7450",
                      "status":"assigned",
                      "purchased_at":"2026-01-10",
                      "warranty_expires_at":"2028-01-10",
                      "created_at":"2026-01-10T08:00:00Z",
                      "updated_at":"2026-01-10T08:00:00Z"
                    }
                  ],
                  "total":1
                }
                """.trimIndent(),
            ),
        )
        val repository = repository()

        val outcome = repository.list()

        assertEquals(
            DeviceListOutcome.Success(
                listOf(
                    EmployeeDevice(
                        id = "device-1",
                        assetCode = "DEMO-LT-001",
                        serialNumber = "DEMO-SEED-DELL-001",
                        name = "Laptop Dell Latitude",
                        deviceType = "laptop",
                        manufacturer = "Dell",
                        model = "Latitude 7450",
                        status = "assigned",
                        purchasedAt = "2026-01-10",
                        warrantyExpiresAt = "2028-01-10",
                    ),
                ),
            ),
            outcome,
        )
        val request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/me/devices", request.path)
    }

    @Test
    fun listMapsServerFailure() = runTest {
        server.enqueue(MockResponse().setResponseCode(500).setBody("""{"error":"internal"}"""))

        assertEquals(DeviceListOutcome.ServerError, repository().list())
    }

    private fun repository() = RemoteEmployeeDeviceRepository(
        api = createEmployeeDeviceApi(
            baseUrl = server.url("/api/v1/").toString(),
            client = OkHttpClient(),
        ),
    )

    private fun jsonResponse(body: String) = MockResponse()
        .setResponseCode(200)
        .setHeader("Content-Type", "application/json")
        .setBody(body)
}
