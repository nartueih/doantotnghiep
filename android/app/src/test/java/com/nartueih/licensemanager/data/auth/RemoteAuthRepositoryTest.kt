package com.nartueih.licensemanager.data.auth

import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.SocketPolicy
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class RemoteAuthRepositoryTest {
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
    fun loginPostsCredentialsAndMapsEmployeeSession() = runTest {
        server.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                      "tokens": {
                        "access_token": "access-token",
                        "refresh_token": "refresh-token",
                        "token_type": "Bearer",
                        "expires_in": 900
                      },
                      "user": {
                        "id": "user-1",
                        "email": "anh.nguyen@local.test",
                        "full_name": "Nguyễn Hoàng Anh",
                        "employee_code": "DEMO-002",
                        "department_id": "department-1",
                        "department_name": "Công nghệ thông tin",
                        "role": "employee",
                        "status": "active",
                        "created_at": "2026-08-01T08:00:00Z"
                      }
                    }
                    """.trimIndent(),
                ),
        )
        val repository = RemoteAuthRepository(
            api = createAuthApi(server.url("/api/v1/").toString()),
        )

        val outcome = repository.login(
            email = "anh.nguyen@local.test",
            password = "ChangeMe123!",
        )

        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/auth/login", request.path)
        assertEquals(
            "{\"email\":\"anh.nguyen@local.test\",\"password\":\"ChangeMe123!\"}",
            request.body.readUtf8(),
        )
        assertTrue(outcome is LoginOutcome.Success)
        val session = (outcome as LoginOutcome.Success).session
        assertEquals("access-token", session.accessToken)
        assertEquals("refresh-token", session.refreshToken)
        assertEquals(900, session.expiresInSeconds)
        assertEquals("Nguyễn Hoàng Anh", session.user.fullName)
        assertEquals("DEMO-002", session.user.employeeCode)
        assertEquals("Công nghệ thông tin", session.user.departmentName)
    }

    @Test
    fun loginMapsUnauthorizedResponseToInvalidCredentials() = runTest {
        server.enqueue(
            MockResponse()
                .setResponseCode(401)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error":"email or password is incorrect"}"""),
        )
        val repository = RemoteAuthRepository(
            api = createAuthApi(server.url("/api/v1/").toString()),
        )

        val outcome = repository.login("employee@local.test", "wrong-password")

        assertEquals(LoginOutcome.InvalidCredentials, outcome)
    }

    @Test
    fun loginRejectsAdminAndRevokesReturnedRefreshToken() = runTest {
        server.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                      "tokens": {
                        "access_token": "admin-access",
                        "refresh_token": "admin-refresh",
                        "token_type": "Bearer",
                        "expires_in": 900
                      },
                      "user": {
                        "id": "admin-1",
                        "email": "admin@local.test",
                        "full_name": "Development Admin",
                        "employee_code": "DEV-ADMIN",
                        "role": "admin",
                        "status": "active"
                      }
                    }
                    """.trimIndent(),
                ),
        )
        server.enqueue(MockResponse().setResponseCode(204))
        val repository = RemoteAuthRepository(
            api = createAuthApi(server.url("/api/v1/").toString()),
        )

        val outcome = repository.login("admin@local.test", "ChangeMe123!")

        assertEquals(LoginOutcome.EmployeeOnly, outcome)
        server.takeRequest()
        val logoutRequest = server.takeRequest()
        assertEquals("POST", logoutRequest.method)
        assertEquals("/api/v1/auth/logout", logoutRequest.path)
        assertEquals(
            "{\"refresh_token\":\"admin-refresh\"}",
            logoutRequest.body.readUtf8(),
        )
    }

    @Test
    fun loginMapsForbiddenResponseToAccountLocked() = runTest {
        server.enqueue(
            MockResponse()
                .setResponseCode(403)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error":"account is locked"}"""),
        )
        val repository = RemoteAuthRepository(
            api = createAuthApi(server.url("/api/v1/").toString()),
        )

        val outcome = repository.login("employee@local.test", "ChangeMe123!")

        assertEquals(LoginOutcome.AccountLocked, outcome)
    }

    @Test
    fun loginMapsTransportFailureToConnectionError() = runTest {
        server.enqueue(MockResponse().setSocketPolicy(SocketPolicy.DISCONNECT_AT_START))
        val repository = RemoteAuthRepository(
            api = createAuthApi(server.url("/api/v1/").toString()),
        )

        val outcome = repository.login("employee@local.test", "ChangeMe123!")

        assertEquals(LoginOutcome.ConnectionError, outcome)
    }

    @Test
    fun loginMapsServerFailureToServerError() = runTest {
        server.enqueue(
            MockResponse()
                .setResponseCode(500)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error":"internal server error"}"""),
        )
        val repository = RemoteAuthRepository(
            api = createAuthApi(server.url("/api/v1/").toString()),
        )

        val outcome = repository.login("employee@local.test", "ChangeMe123!")

        assertEquals(LoginOutcome.ServerError, outcome)
    }

    @Test
    fun refreshPostsRefreshTokenAndMapsReplacementSession() = runTest {
        server.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                      "tokens": {
                        "access_token": "new-access",
                        "refresh_token": "new-refresh",
                        "token_type": "Bearer",
                        "expires_in": 900
                      },
                      "user": {
                        "id": "user-1",
                        "email": "employee@local.test",
                        "full_name": "Nguyễn Hoàng Anh",
                        "employee_code": "EMP-001",
                        "role": "employee",
                        "status": "active"
                      }
                    }
                    """.trimIndent(),
                ),
        )
        val repository = RemoteAuthRepository(
            api = createAuthApi(server.url("/api/v1/").toString()),
        )

        val outcome = repository.refresh("old-refresh")

        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/auth/refresh", request.path)
        assertEquals("{\"refresh_token\":\"old-refresh\"}", request.body.readUtf8())
        assertTrue(outcome is RefreshOutcome.Success)
        val session = (outcome as RefreshOutcome.Success).session
        assertEquals("new-access", session.accessToken)
        assertEquals("new-refresh", session.refreshToken)
    }

    @Test
    fun refreshMapsUnauthorizedResponseToInvalidSession() = runTest {
        server.enqueue(
            MockResponse()
                .setResponseCode(401)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error":"token is invalid or expired"}"""),
        )
        val repository = RemoteAuthRepository(
            api = createAuthApi(server.url("/api/v1/").toString()),
        )

        assertEquals(RefreshOutcome.InvalidSession, repository.refresh("expired-refresh"))
    }
}
