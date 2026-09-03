package com.nartueih.licensemanager.feature.licenses

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.nartueih.licensemanager.data.licenses.EmployeeLicense
import com.nartueih.licensemanager.data.licenses.EmployeeLicenseRepository
import com.nartueih.licensemanager.data.licenses.LicenseKeyOutcome
import com.nartueih.licensemanager.data.licenses.LicenseListOutcome
import com.nartueih.licensemanager.data.licenses.RevealedLicenseKey
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class LicenseUiState(
    val isLoading: Boolean = true,
    val items: List<EmployeeLicense> = emptyList(),
    val loadError: String? = null,
    val revealingAssignmentId: String? = null,
    val revealedKey: RevealedLicenseKey? = null,
    val keyError: String? = null,
)

class LicenseViewModel(
    private val repository: EmployeeLicenseRepository,
) : ViewModel() {
    private val _uiState = MutableStateFlow(LicenseUiState())
    val uiState: StateFlow<LicenseUiState> = _uiState.asStateFlow()

    init {
        load()
    }

    fun retry() {
        load()
    }

    fun sync() {
        load(showLoading = false)
    }

    fun revealKey(assignmentId: String) {
        if (_uiState.value.revealingAssignmentId != null) return
        _uiState.update {
            it.copy(
                revealingAssignmentId = assignmentId,
                revealedKey = null,
                keyError = null,
            )
        }
        viewModelScope.launch {
            val outcome = repository.revealKey(assignmentId)
            _uiState.update { current ->
                when (outcome) {
                    is LicenseKeyOutcome.Success -> current.copy(
                        revealingAssignmentId = null,
                        revealedKey = outcome.result,
                    )
                    LicenseKeyOutcome.NotAllowed -> current.withKeyError(
                        "Bạn không có quyền xem key license này.",
                    )
                    LicenseKeyOutcome.Unavailable -> current.withKeyError(
                        "Key license hiện không khả dụng. Vui lòng liên hệ IT.",
                    )
                    LicenseKeyOutcome.ConnectionError -> current.withKeyError(
                        "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    )
                    LicenseKeyOutcome.ServerError -> current.withKeyError(
                        "Không thể tải key license. Vui lòng thử lại sau.",
                    )
                }
            }
        }
    }

    fun dismissKeyResult() {
        _uiState.update { it.copy(revealedKey = null, keyError = null) }
    }

    private fun load(showLoading: Boolean = true) {
        if (showLoading) _uiState.update { it.copy(isLoading = true, loadError = null) }
        viewModelScope.launch {
            val outcome = repository.list()
            if (!showLoading && outcome !is LicenseListOutcome.Success) return@launch
            _uiState.value = when (outcome) {
                is LicenseListOutcome.Success -> _uiState.value.copy(
                    isLoading = false,
                    items = outcome.items,
                    loadError = null,
                )
                LicenseListOutcome.ConnectionError -> _uiState.value.copy(
                    isLoading = false,
                    items = emptyList(),
                    loadError = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                )
                LicenseListOutcome.ServerError -> _uiState.value.copy(
                    isLoading = false,
                    items = emptyList(),
                    loadError = "Không thể tải danh sách license. Vui lòng thử lại sau.",
                )
            }
        }
    }

    private fun LicenseUiState.withKeyError(message: String) = copy(
        revealingAssignmentId = null,
        keyError = message,
    )

    class Factory(
        private val repository: EmployeeLicenseRepository,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            return LicenseViewModel(repository) as T
        }
    }
}
