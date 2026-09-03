package com.nartueih.licensemanager.core.session

import com.nartueih.licensemanager.data.auth.AuthRepository
import com.nartueih.licensemanager.data.auth.EmployeeSession
import com.nartueih.licensemanager.data.auth.EmployeeUser
import com.nartueih.licensemanager.data.auth.LoginOutcome
import com.nartueih.licensemanager.data.auth.RefreshOutcome
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class SessionRefresherTest {
    @Test
    fun concurrentFailuresForTheSameAccessTokenPerformOneRefresh() = runTest {
        val store = FakeSessionStore(oldSession)
        val repository = RecordingAuthRepository(RefreshOutcome.Success(newSession))
        val refresher = SessionRefresher(store, repository)

        val results = List(8) {
            async { refresher.refreshAfterUnauthorized("old-access") }
        }.awaitAll()

        assertEquals(1, repository.refreshCalls)
        assertEquals(newSession, store.session.value)
        assertTrue(results.all { it == SessionRefreshResult.Refreshed("new-access") })
    }

    @Test
    fun invalidRefreshTokenClearsStoredSession() = runTest {
        val store = FakeSessionStore(oldSession)
        val refresher = SessionRefresher(
            store,
            RecordingAuthRepository(RefreshOutcome.InvalidSession),
        )

        val result = refresher.refreshAfterUnauthorized("old-access")

        assertEquals(SessionRefreshResult.SessionExpired, result)
        assertEquals(null, store.session.value)
    }

    @Test
    fun connectionFailureKeepsStoredSessionForRetry() = runTest {
        val store = FakeSessionStore(oldSession)
        val refresher = SessionRefresher(
            store,
            RecordingAuthRepository(RefreshOutcome.ConnectionError),
        )

        val result = refresher.refreshAfterUnauthorized("old-access")

        assertEquals(SessionRefreshResult.TemporarilyUnavailable, result)
        assertEquals(oldSession, store.session.value)
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

    private class RecordingAuthRepository(
        private val refreshOutcome: RefreshOutcome,
    ) : AuthRepository {
        var refreshCalls = 0

        override suspend fun login(email: String, password: String) = LoginOutcome.InvalidCredentials

        override suspend fun refresh(refreshToken: String): RefreshOutcome {
            refreshCalls += 1
            delay(10)
            return refreshOutcome
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
