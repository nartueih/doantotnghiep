package com.nartueih.licensemanager.data.dashboard

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class RemoteEmployeeDashboardRepositoryTest {
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
    fun loadFetchesAllPersonalSummariesAndMapsCounts() = runTest {
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse = when (request.path) {
                "/api/v1/me/licenses" -> jsonResponse("""{"items":[],"total":4}""")
                "/api/v1/me/devices" -> jsonResponse("""{"items":[],"total":2}""")
                "/api/v1/me/notifications" -> jsonResponse(
                    """{"items":[],"total":7,"unread_count":3}""",
                )
                else -> MockResponse().setResponseCode(404)
            }
        }
        val repository = RemoteEmployeeDashboardRepository(
            api = createEmployeeDashboardApi(
                baseUrl = server.url("/api/v1/").toString(),
                client = OkHttpClient(),
            ),
        )

        val outcome = repository.load()

        assertEquals(
            DashboardLoadOutcome.Success(
                EmployeeDashboardSummary(
                    licenseCount = 4,
                    deviceCount = 2,
                    unreadNotificationCount = 3,
                ),
            ),
            outcome,
        )
        assertEquals(3, server.requestCount)
    }

    @Test
    fun loadMapsServerFailureWithoutCrashing() = runTest {
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest) = MockResponse()
                .setResponseCode(500)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error":"internal server error"}""")
        }
        val repository = RemoteEmployeeDashboardRepository(
            api = createEmployeeDashboardApi(
                baseUrl = server.url("/api/v1/").toString(),
                client = OkHttpClient(),
            ),
        )

        assertEquals(DashboardLoadOutcome.ServerError, repository.load())
    }

    @Test
    fun loadMapsConnectionFailureWithoutCrashing() = runTest {
        val baseUrl = server.url("/api/v1/").toString()
        server.shutdown()
        val repository = RemoteEmployeeDashboardRepository(
            api = createEmployeeDashboardApi(baseUrl, OkHttpClient()),
        )

        assertTrue(repository.load() is DashboardLoadOutcome.ConnectionError)
    }

    private fun jsonResponse(body: String) = MockResponse()
        .setResponseCode(200)
        .setHeader("Content-Type", "application/json")
        .setBody(body)
}
