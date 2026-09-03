package com.nartueih.licensemanager.feature.requests

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.nartueih.licensemanager.data.licenserequests.CreateLicenseRequestInput
import com.nartueih.licensemanager.data.licenserequests.LicenseRequest
import com.nartueih.licensemanager.data.licenserequests.LicenseRequestLoadOutcome
import com.nartueih.licensemanager.data.licenserequests.LicenseRequestMutationOutcome
import com.nartueih.licensemanager.data.licenserequests.LicenseRequestRepository
import com.nartueih.licensemanager.data.licenserequests.RequestableSoftware
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class LicenseRequestUiState(
    val isLoading: Boolean = true,
    val items: List<LicenseRequest> = emptyList(),
    val software: List<RequestableSoftware> = emptyList(),
    val pendingCount: Int = 0,
    val loadError: String? = null,
    val isCreateOpen: Boolean = false,
    val selectedSoftwareId: String = "",
    val priority: String = "normal",
    val reason: String = "",
    val isSubmitting: Boolean = false,
    val formError: String? = null,
    val cancellingRequestId: String? = null,
    val message: String? = null,
)

class LicenseRequestViewModel(
    private val repository: LicenseRequestRepository,
) : ViewModel() {
    private val _uiState = MutableStateFlow(LicenseRequestUiState())
    val uiState: StateFlow<LicenseRequestUiState> = _uiState.asStateFlow()

    init {
        load()
    }

    fun retry() = load()

    fun sync() = load(showLoading = false)

    fun openCreate(softwareProductId: String?) {
        _uiState.update {
            it.copy(
                isCreateOpen = true,
                selectedSoftwareId = softwareProductId.orEmpty(),
                priority = "normal",
                reason = "",
                formError = null,
                message = null,
            )
        }
    }

    fun dismissCreate() {
        if (_uiState.value.isSubmitting) return
        _uiState.update { it.copy(isCreateOpen = false, formError = null) }
    }

    fun updateSoftware(value: String) = updateForm { copy(selectedSoftwareId = value, formError = null) }
    fun updatePriority(value: String) = updateForm { copy(priority = value, formError = null) }
    fun updateReason(value: String) = updateForm { copy(reason = value, formError = null) }
    fun dismissMessage() = _uiState.update { it.copy(message = null) }

    fun submit() {
        val state = _uiState.value
        val reason = state.reason.trim()
        if (state.selectedSoftwareId.isBlank() || reason.isBlank()) {
            _uiState.update { it.copy(formError = "Vui lòng chọn phần mềm và nhập lý do sử dụng.") }
            return
        }
        _uiState.update { it.copy(isSubmitting = true, formError = null) }
        viewModelScope.launch {
            val outcome = repository.create(
                CreateLicenseRequestInput(state.selectedSoftwareId, state.priority, reason),
            )
            _uiState.update { current ->
                when (outcome) {
                    is LicenseRequestMutationOutcome.Success -> {
                        val updated = listOf(outcome.item) + current.items.filterNot { it.id == outcome.item.id }
                        current.copy(
                            items = updated,
                            pendingCount = updated.count { it.status == "pending" },
                            isCreateOpen = false,
                            isSubmitting = false,
                            formError = null,
                            message = "Đã gửi yêu cầu cấp license.",
                        )
                    }
                    LicenseRequestMutationOutcome.PendingRequestExists -> current.copy(
                        isSubmitting = false,
                        formError = "Bạn đã có một yêu cầu đang chờ cho phần mềm này.",
                    )
                    LicenseRequestMutationOutcome.InvalidState -> current.copy(
                        isSubmitting = false,
                        formError = "Trạng thái yêu cầu không còn cho phép thao tác này.",
                    )
                    LicenseRequestMutationOutcome.ConnectionError -> current.copy(
                        isSubmitting = false,
                        formError = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    )
                    LicenseRequestMutationOutcome.ServerError -> current.copy(
                        isSubmitting = false,
                        formError = "Không thể gửi yêu cầu cấp license. Vui lòng thử lại sau.",
                    )
                }
            }
        }
    }

    fun cancel(requestId: String) {
        if (_uiState.value.cancellingRequestId != null) return
        _uiState.update { it.copy(cancellingRequestId = requestId, message = null) }
        viewModelScope.launch {
            val outcome = repository.cancel(requestId)
            _uiState.update { current ->
                when (outcome) {
                    is LicenseRequestMutationOutcome.Success -> {
                        val updated = current.items.map { if (it.id == outcome.item.id) outcome.item else it }
                        current.copy(
                            items = updated,
                            pendingCount = updated.count { it.status == "pending" },
                            cancellingRequestId = null,
                            message = "Đã hủy yêu cầu cấp license.",
                        )
                    }
                    LicenseRequestMutationOutcome.InvalidState -> current.copy(
                        cancellingRequestId = null,
                        message = "Yêu cầu này không còn có thể hủy.",
                    )
                    LicenseRequestMutationOutcome.ConnectionError -> current.copy(
                        cancellingRequestId = null,
                        message = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    )
                    else -> current.copy(
                        cancellingRequestId = null,
                        message = "Không thể hủy yêu cầu cấp license.",
                    )
                }
            }
        }
    }

    private fun load(showLoading: Boolean = true) {
        if (showLoading) _uiState.update { it.copy(isLoading = true, loadError = null) }
        viewModelScope.launch {
            val outcome = repository.load()
            if (!showLoading && outcome !is LicenseRequestLoadOutcome.Success) return@launch
            _uiState.update { current ->
                when (outcome) {
                    is LicenseRequestLoadOutcome.Success -> current.copy(
                        isLoading = false,
                        items = outcome.items,
                        software = outcome.software,
                        pendingCount = outcome.items.count { it.status == "pending" },
                    )
                    LicenseRequestLoadOutcome.ConnectionError -> current.copy(
                        isLoading = false,
                        loadError = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    )
                    LicenseRequestLoadOutcome.ServerError -> current.copy(
                        isLoading = false,
                        loadError = "Không thể tải yêu cầu cấp license. Vui lòng thử lại sau.",
                    )
                }
            }
        }
    }

    private inline fun updateForm(transform: LicenseRequestUiState.() -> LicenseRequestUiState) {
        _uiState.update(transform)
    }

    class Factory(
        private val repository: LicenseRequestRepository,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T = LicenseRequestViewModel(repository) as T
    }
}
