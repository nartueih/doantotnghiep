package com.nartueih.licensemanager.feature.auth

import com.nartueih.licensemanager.MainDispatcherRule
import com.nartueih.licensemanager.core.session.SessionStore
import com.nartueih.licensemanager.data.auth.AuthRepository
import com.nartueih.licensemanager.data.auth.EmployeeSession
import com.nartueih.licensemanager.data.auth.EmployeeUser
import com.nartueih.licensemanager.data.auth.LoginOutcome
import com.nartueih.licensemanager.data.auth.RefreshOutcome
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class LoginViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    @Test
    fun loginClickWithBlankCredentialsExposesFieldErrors() {
        val viewModel = LoginViewModel()

        viewModel.onLoginClicked()

        val state = viewModel.uiState.value
        assertFalse(state.canSubmit)
        assertEquals("Email không được bỏ trống.", state.emailError)
        assertEquals("Mật khẩu không được bỏ trống.", state.passwordError)
    }

    @Test
    fun editingEmailUpdatesItsValueAndClearsOnlyItsError() {
        val viewModel = LoginViewModel()
        viewModel.onLoginClicked()

        viewModel.onEmailChanged("employee@local.test")

        val state = viewModel.uiState.value
        assertEquals("employee@local.test", state.email)
        assertEquals(null, state.emailError)
        assertEquals("Mật khẩu không được bỏ trống.", state.passwordError)
    }

    @Test
    fun editingPasswordUpdatesItsValueAndClearsOnlyItsError() {
        val viewModel = LoginViewModel()
        viewModel.onLoginClicked()

        viewModel.onPasswordChanged("ChangeMe123!")

        val state = viewModel.uiState.value
        assertEquals("ChangeMe123!", state.password)
        assertEquals("Email không được bỏ trống.", state.emailError)
        assertEquals(null, state.passwordError)
    }

    @Test
    fun validLoginSendsNormalizedCredentialsAndExposesAuthenticatedEmployee() = runTest {
        val session = EmployeeSession(
            accessToken = "access-token",
            refreshToken = "refresh-token",
            expiresInSeconds = 900,
            user = EmployeeUser(
                id = "user-1",
                email = "employee@local.test",
                fullName = "Nguyễn Hoàng Anh",
                employeeCode = "DEMO-002",
                departmentId = "department-1",
                departmentName = "Công nghệ thông tin",
            ),
        )
        val repository = RecordingAuthRepository(LoginOutcome.Success(session))
        val sessionStore = RecordingSessionStore()
        val viewModel = LoginViewModel(repository, sessionStore)
        viewModel.onEmailChanged("  employee@local.test  ")
        viewModel.onPasswordChanged("ChangeMe123!")

        viewModel.onLoginClicked()
        advanceUntilIdle()

        assertEquals("employee@local.test", repository.lastEmail)
        assertEquals("ChangeMe123!", repository.lastPassword)
        assertEquals(session, sessionStore.session.value)
        assertEquals("Nguyễn Hoàng Anh", viewModel.uiState.value.authenticatedUser?.fullName)
        assertFalse(viewModel.uiState.value.isSubmitting)
    }

    @Test
    fun loginFailuresExposeVietnameseMessages() = runTest {
        val cases = listOf(
            LoginOutcome.InvalidCredentials to "Email hoặc mật khẩu không chính xác.",
            LoginOutcome.AccountLocked to "Tài khoản đã bị khóa. Vui lòng liên hệ IT.",
            LoginOutcome.EmployeeOnly to "Ứng dụng này chỉ dành cho nhân viên.",
            LoginOutcome.ConnectionError to "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
            LoginOutcome.ServerError to "Máy chủ đang gặp sự cố. Vui lòng thử lại sau.",
        )

        cases.forEach { (outcome, expectedMessage) ->
            val viewModel = LoginViewModel(RecordingAuthRepository(outcome))
            viewModel.onEmailChanged("employee@local.test")
            viewModel.onPasswordChanged("ChangeMe123!")

            viewModel.onLoginClicked()
            advanceUntilIdle()

            assertEquals(expectedMessage, viewModel.uiState.value.generalError)
            assertFalse(viewModel.uiState.value.isSubmitting)
        }
    }

    @Test
    fun sessionStorageFailureKeepsUserOnLoginWithoutCrashing() = runTest {
        val session = EmployeeSession(
            accessToken = "access-token",
            refreshToken = "refresh-token",
            expiresInSeconds = 900,
            user = EmployeeUser(
                id = "user-1",
                email = "employee@local.test",
                fullName = "Nguyễn Hoàng Anh",
                employeeCode = "EMP-001",
                departmentId = null,
                departmentName = null,
            ),
        )
        val viewModel = LoginViewModel(
            RecordingAuthRepository(LoginOutcome.Success(session)),
            FailingSessionStore(),
        )
        viewModel.onEmailChanged("employee@local.test")
        viewModel.onPasswordChanged("ChangeMe123!")

        viewModel.onLoginClicked()
        advanceUntilIdle()

        assertNull(viewModel.uiState.value.authenticatedSession)
        assertEquals(
            "Không thể lưu phiên đăng nhập an toàn. Vui lòng thử lại.",
            viewModel.uiState.value.generalError,
        )
        assertFalse(viewModel.uiState.value.isSubmitting)
    }

    @Test
    fun returningToLoginAfterLogoutClearsCredentialsAndPreviousSession() = runTest {
        val session = EmployeeSession(
            accessToken = "access-token",
            refreshToken = "refresh-token",
            expiresInSeconds = 900,
            user = EmployeeUser(
                id = "user-1",
                email = "employee@local.test",
                fullName = "Nguyễn Hoàng Anh",
                employeeCode = "EMP-001",
                departmentId = null,
                departmentName = null,
            ),
        )
        val viewModel = LoginViewModel(
            RecordingAuthRepository(LoginOutcome.Success(session)),
            RecordingSessionStore(),
        )
        viewModel.onEmailChanged("employee@local.test")
        viewModel.onPasswordChanged("ChangeMe123!")
        viewModel.onLoginClicked()
        advanceUntilIdle()

        viewModel.onSignedOut()

        assertEquals(LoginUiState(), viewModel.uiState.value)
    }

    private class RecordingAuthRepository(
        private val outcome: LoginOutcome,
    ) : AuthRepository {
        var lastEmail: String? = null
        var lastPassword: String? = null

        override suspend fun login(email: String, password: String): LoginOutcome {
            lastEmail = email
            lastPassword = password
            return outcome
        }

        override suspend fun logout(refreshToken: String) = Unit

        override suspend fun refresh(refreshToken: String) = RefreshOutcome.InvalidSession
    }

    private class RecordingSessionStore : SessionStore {
        override val session = MutableStateFlow<EmployeeSession?>(null)

        override suspend fun save(session: EmployeeSession) {
            this.session.value = session
        }

        override suspend fun clear() {
            session.value = null
        }
    }

    private class FailingSessionStore : SessionStore {
        override val session = MutableStateFlow<EmployeeSession?>(null)

        override suspend fun save(session: EmployeeSession) {
            error("keystore unavailable")
        }

        override suspend fun clear() = Unit
    }

}
