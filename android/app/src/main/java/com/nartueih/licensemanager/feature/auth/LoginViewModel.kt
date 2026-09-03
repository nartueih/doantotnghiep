package com.nartueih.licensemanager.feature.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.nartueih.licensemanager.core.session.SessionStore
import com.nartueih.licensemanager.data.auth.AuthRepository
import com.nartueih.licensemanager.data.auth.EmployeeSession
import com.nartueih.licensemanager.data.auth.EmployeeUser
import com.nartueih.licensemanager.data.auth.LoginOutcome
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

data class LoginUiState(
    val email: String = "",
    val password: String = "",
    val emailError: String? = null,
    val passwordError: String? = null,
    val isSubmitting: Boolean = false,
    val generalError: String? = null,
    val authenticatedSession: EmployeeSession? = null,
) {
    val authenticatedUser: EmployeeUser?
        get() = authenticatedSession?.user

    val canSubmit: Boolean
        get() = !isSubmitting && email.isNotBlank() && password.isNotBlank() &&
            emailError == null && passwordError == null
}

class LoginViewModel(
    private val authRepository: AuthRepository? = null,
    private val sessionStore: SessionStore? = null,
) : ViewModel() {
    private val _uiState = MutableStateFlow(LoginUiState())
    val uiState: StateFlow<LoginUiState> = _uiState.asStateFlow()

    fun onEmailChanged(email: String) {
        _uiState.value = _uiState.value.copy(
            email = email,
            emailError = null,
            generalError = null,
        )
    }

    fun onPasswordChanged(password: String) {
        _uiState.value = _uiState.value.copy(
            password = password,
            passwordError = null,
            generalError = null,
        )
    }

    fun onSignedOut() {
        _uiState.value = LoginUiState()
    }

    fun onLoginClicked() {
        if (_uiState.value.isSubmitting) return

        val validation = LoginFormValidator.validate(
            email = _uiState.value.email,
            password = _uiState.value.password,
        )

        _uiState.value = _uiState.value.copy(
            email = validation.normalizedEmail,
            emailError = validation.emailError,
            passwordError = validation.passwordError,
            generalError = null,
        )
        if (!validation.isValid) return

        val repository = authRepository ?: return
        val password = _uiState.value.password
        _uiState.value = _uiState.value.copy(isSubmitting = true)
        viewModelScope.launch {
            when (val outcome = repository.login(validation.normalizedEmail, password)) {
                is LoginOutcome.Success -> {
                    try {
                        sessionStore?.save(outcome.session)
                    } catch (error: Exception) {
                        if (error is CancellationException) throw error
                        _uiState.value = _uiState.value.copy(
                            isSubmitting = false,
                            generalError = "Không thể lưu phiên đăng nhập an toàn. Vui lòng thử lại.",
                        )
                        return@launch
                    }
                    _uiState.value = _uiState.value.copy(
                        isSubmitting = false,
                        authenticatedSession = outcome.session,
                    )
                }
                else -> {
                    val message = when (outcome) {
                        LoginOutcome.InvalidCredentials -> "Email hoặc mật khẩu không chính xác."
                        LoginOutcome.AccountLocked -> "Tài khoản đã bị khóa. Vui lòng liên hệ IT."
                        LoginOutcome.EmployeeOnly -> "Ứng dụng này chỉ dành cho nhân viên."
                        LoginOutcome.ConnectionError -> "Không thể kết nối tới máy chủ. Vui lòng thử lại."
                        LoginOutcome.ServerError -> "Máy chủ đang gặp sự cố. Vui lòng thử lại sau."
                        is LoginOutcome.Success -> error("Handled above")
                    }
                    _uiState.value = _uiState.value.copy(
                        isSubmitting = false,
                        generalError = message,
                    )
                }
            }
        }
    }

    class Factory(
        private val authRepository: AuthRepository,
        private val sessionStore: SessionStore,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            require(modelClass.isAssignableFrom(LoginViewModel::class.java))
            return LoginViewModel(authRepository, sessionStore) as T
        }
    }
}
