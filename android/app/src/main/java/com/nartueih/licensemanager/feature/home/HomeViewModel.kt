package com.nartueih.licensemanager.feature.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.nartueih.licensemanager.data.dashboard.DashboardLoadOutcome
import com.nartueih.licensemanager.data.dashboard.EmployeeDashboardRepository
import com.nartueih.licensemanager.data.dashboard.EmployeeDashboardSummary
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

sealed interface HomeUiState {
    data object Loading : HomeUiState
    data class Content(val summary: EmployeeDashboardSummary) : HomeUiState
    data class Error(val message: String) : HomeUiState
}

class HomeViewModel(
    private val repository: EmployeeDashboardRepository,
) : ViewModel() {
    private val _uiState = MutableStateFlow<HomeUiState>(HomeUiState.Loading)
    val uiState: StateFlow<HomeUiState> = _uiState.asStateFlow()

    init {
        load()
    }

    fun retry() {
        load()
    }

    fun sync() {
        load(showLoading = false)
    }

    private fun load(showLoading: Boolean = true) {
        if (showLoading) _uiState.value = HomeUiState.Loading
        viewModelScope.launch {
            val outcome = repository.load()
            if (!showLoading && outcome !is DashboardLoadOutcome.Success) return@launch
            _uiState.value = when (outcome) {
                is DashboardLoadOutcome.Success -> HomeUiState.Content(outcome.summary)
                DashboardLoadOutcome.ConnectionError -> HomeUiState.Error(
                    "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                )
                DashboardLoadOutcome.ServerError -> HomeUiState.Error(
                    "Không thể tải dữ liệu tổng quan. Vui lòng thử lại sau.",
                )
            }
        }
    }

    class Factory(
        private val repository: EmployeeDashboardRepository,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            return HomeViewModel(repository) as T
        }
    }
}
