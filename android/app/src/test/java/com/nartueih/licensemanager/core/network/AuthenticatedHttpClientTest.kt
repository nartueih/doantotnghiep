package com.nartueih.licensemanager.core.network

import com.nartueih.licensemanager.core.session.SessionRefresher
import com.nartueih.licensemanager.core.session.SessionStore
import com.nartueih.licensemanager.data.auth.AuthRepository
import com.nartueih.licensemanager.data.auth.EmployeeSession
import com.nartueih.licensemanager.data.auth.EmployeeUser
import com.nartueih.licensemanager.data.auth.LoginOutcome
import com.nartueih.licensemanager.data.auth.RefreshOutcome
import kotlinx.coroutines.flow.MutableStateFlow
import okhttp3.Request
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test

class AuthenticatedHttpClientTest {
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
    fun unauthorizedResponseRefreshesAndRetriesOnceWithNewAccessToken() {
        server.enqueue(MockResponse().setResponseCode(401))
        server.enqueue(MockResponse().setResponseCode(200).setBody("ok"))
        val store = FakeSessionStore(oldSession)
        val repository = FakeAuthRepository(RefreshOutcome.Success(newSession))
        val client = createAuthenticatedHttpClient(
            store,
            SessionRefresher(store, repository),
        )

        val response = client.newCall(Request.Builder().url(server.url("/protected")).build()).execute()

        response.use { assertEquals(200, it.code) }
        assertEquals("Bearer old-access", server.takeRequest().getHeader("Authorization"))
        assertEquals("Bearer new-access", server.takeRequest().getHeader("Authorization"))
        assertEquals(1, repository.refreshCalls)
    }

    @Test
    fun invalidRefreshTokenReturnsUnauthorizedAndClearsSessionWithoutLooping() {
        server.enqueue(MockResponse().setResponseCode(401))
        val store = FakeSessionStore(oldSession)
        val repository = FakeAuthRepository(RefreshOutcome.InvalidSession)
        val client = createAuthenticatedHttpClient(
            store,
            SessionRefresher(store, repository),
        )

        val response = client.newCall(Request.Builder().url(server.url("/protected")).build()).execute()

        response.use { assertEquals(401, it.code) }
        assertEquals(1, server.requestCount)
        assertEquals(null, store.session.value)
    }

    private class FakeSessionStore(initial: EmployeeSession?) : SessionStore {
        override val session = MutableStateFlow(initial)

        override suspend fun save(session: EmployeeSession) {
            this.session.value = session
        }

        override suspend fun clear() {
            session.value = null
        }
    }

    private class FakeAuthRepository(
        private val outcome: RefreshOutcome,
    ) : AuthRepository {
        var refreshCalls = 0

        override suspend fun login(email: String, password: String) = LoginOutcome.InvalidCredentials

        override suspend fun refresh(refreshToken: String): RefreshOutcome {
            refreshCalls += 1
            return outcome
        }

        override suspend fun logout(refreshToken: String) = Unit
    }

    private companion object {
        val user = EmployeeUser(
            id = "employee-1",
            email = "employee@local.test",
            fullName = "Nguyễn Hoàng Anh",
            employeeCode = "EMP-001",
            departmentId = null,
            departmentName = null,
        )
        val oldSession = EmployeeSession("old-access", "old-refresh", 900, user)
        val newSession = EmployeeSession("new-access", "new-refresh", 900, user)
    }
}
