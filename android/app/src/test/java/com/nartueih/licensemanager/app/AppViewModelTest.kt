package com.nartueih.licensemanager.app

import com.nartueih.licensemanager.MainDispatcherRule
import com.nartueih.licensemanager.core.session.SessionStore
import com.nartueih.licensemanager.data.auth.AuthRepository
import com.nartueih.licensemanager.data.auth.EmployeeSession
import com.nartueih.licensemanager.data.auth.EmployeeUser
import com.nartueih.licensemanager.data.auth.LoginOutcome
import com.nartueih.licensemanager.data.auth.RefreshOutcome
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class AppViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    @Test
    fun startupWithoutStoredSessionOpensLogin() = runTest {
        val viewModel = AppViewModel(FakeSessionStore(null), RecordingAuthRepository())

        assertEquals(AppUiState.Loading, viewModel.uiState.value)
        advanceUntilIdle()

        assertEquals(AppUiState.SignedOut, viewModel.uiState.value)
    }

    @Test
    fun startupWithStoredSessionOpensEmployeeArea() = runTest {
        val viewModel = AppViewModel(FakeSessionStore(sampleSession), RecordingAuthRepository())

        advanceUntilIdle()

        assertEquals(AppUiState.SignedIn(sampleSession), viewModel.uiState.value)
    }

    @Test
    fun logoutRevokesRefreshTokenAndClearsLocalSession() = runTest {
        val store = FakeSessionStore(sampleSession)
        val repository = RecordingAuthRepository()
        val viewModel = AppViewModel(store, repository)
        advanceUntilIdle()

        viewModel.onLogoutClicked()
        advanceUntilIdle()

        assertEquals("refresh-token", repository.loggedOutRefreshToken)
        assertEquals(null, store.session.value)
        assertEquals(AppUiState.SignedOut, viewModel.uiState.value)
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

    private class RecordingAuthRepository : AuthRepository {
        var loggedOutRefreshToken: String? = null

        override suspend fun login(email: String, password: String) = LoginOutcome.InvalidCredentials

        override suspend fun refresh(refreshToken: String) = RefreshOutcome.InvalidSession

        override suspend fun logout(refreshToken: String) {
            loggedOutRefreshToken = refreshToken
        }
    }

    private companion object {
        val sampleSession = EmployeeSession(
            accessToken = "access-token",
            refreshToken = "refresh-token",
            expiresInSeconds = 900,
            user = EmployeeUser(
                id = "employee-1",
                email = "employee@local.test",
                fullName = "Nguyễn Hoàng Anh",
                employeeCode = "EMP-001",
                departmentId = null,
                departmentName = null,
            ),
        )
    }
}
