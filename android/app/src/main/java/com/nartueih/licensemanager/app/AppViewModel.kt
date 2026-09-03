package com.nartueih.licensemanager.app

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.nartueih.licensemanager.core.session.SessionStore
import com.nartueih.licensemanager.data.auth.AuthRepository
import com.nartueih.licensemanager.data.auth.EmployeeSession
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

sealed interface AppUiState {
    data object Loading : AppUiState
    data object SignedOut : AppUiState
    data class SignedIn(val session: EmployeeSession) : AppUiState
}

class AppViewModel(
    private val sessionStore: SessionStore,
    private val authRepository: AuthRepository,
) : ViewModel() {
    private val _uiState = MutableStateFlow<AppUiState>(AppUiState.Loading)
    val uiState: StateFlow<AppUiState> = _uiState.asStateFlow()

    init {
        viewModelScope.launch {
            sessionStore.session.collect { session ->
                _uiState.value = session?.let(AppUiState::SignedIn) ?: AppUiState.SignedOut
            }
        }
    }

    fun onLogoutClicked() {
        val session = (_uiState.value as? AppUiState.SignedIn)?.session ?: return
        viewModelScope.launch {
            authRepository.logout(session.refreshToken)
            sessionStore.clear()
        }
    }

    class Factory(
        private val sessionStore: SessionStore,
        private val authRepository: AuthRepository,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            require(modelClass.isAssignableFrom(AppViewModel::class.java))
            return AppViewModel(sessionStore, authRepository) as T
        }
    }
}
