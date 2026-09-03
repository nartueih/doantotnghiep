package com.nartueih.licensemanager.feature.requests

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.nartueih.licensemanager.data.maintenance.CreateMaintenanceRequestInput
import com.nartueih.licensemanager.data.maintenance.MaintenanceListOutcome
import com.nartueih.licensemanager.data.maintenance.MaintenanceMutationOutcome
import com.nartueih.licensemanager.data.maintenance.MaintenanceRequest
import com.nartueih.licensemanager.data.maintenance.MaintenanceRequestRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class MaintenanceRequestUiState(
    val isLoading: Boolean = true,
    val items: List<MaintenanceRequest> = emptyList(),
    val openCount: Int = 0,
    val loadError: String? = null,
    val isCreateOpen: Boolean = false,
    val selectedDeviceId: String = "",
    val category: String = "hardware",
    val priority: String = "normal",
    val title: String = "",
    val description: String = "",
    val isSubmitting: Boolean = false,
    val formError: String? = null,
    val cancellingRequestId: String? = null,
    val message: String? = null,
)

class MaintenanceRequestViewModel(
    private val repository: MaintenanceRequestRepository,
) : ViewModel() {
    private val _uiState = MutableStateFlow(MaintenanceRequestUiState())
    val uiState: StateFlow<MaintenanceRequestUiState> = _uiState.asStateFlow()

    init {
        load()
    }

    fun retry() = load()

    fun sync() = load(showLoading = false)

    fun openCreate(deviceId: String?) {
        if (deviceId != null && _uiState.value.items.any {
                it.deviceId == deviceId && (it.status == "pending" || it.status == "in_progress")
            }
        ) {
            _uiState.update {
                it.copy(
                    isCreateOpen = false,
                    message = "Thiết bị này đã có một yêu cầu bảo trì đang mở.",
                )
            }
            return
        }
        _uiState.update {
            it.copy(
                isCreateOpen = true,
                selectedDeviceId = deviceId.orEmpty(),
                category = "hardware",
                priority = "normal",
                title = "",
                description = "",
                formError = null,
                message = null,
            )
        }
    }

    fun dismissCreate() {
        if (_uiState.value.isSubmitting) return
        _uiState.update { it.copy(isCreateOpen = false, formError = null) }
    }

    fun updateDevice(value: String) = updateForm { copy(selectedDeviceId = value, formError = null) }
    fun updateCategory(value: String) = updateForm { copy(category = value, formError = null) }
    fun updatePriority(value: String) = updateForm { copy(priority = value, formError = null) }
    fun updateTitle(value: String) = updateForm { copy(title = value.take(200), formError = null) }
    fun updateDescription(value: String) = updateForm { copy(description = value, formError = null) }
    fun dismissMessage() = _uiState.update { it.copy(message = null) }

    fun submit() {
        val state = _uiState.value
        val title = state.title.trim()
        val description = state.description.trim()
        if (state.selectedDeviceId.isBlank() || title.isBlank() || description.isBlank()) {
            _uiState.update {
                it.copy(formError = "Vui lòng chọn thiết bị và nhập đầy đủ tiêu đề, mô tả.")
            }
            return
        }
        _uiState.update { it.copy(isSubmitting = true, formError = null) }
        viewModelScope.launch {
            val outcome = repository.create(
                CreateMaintenanceRequestInput(
                    deviceId = state.selectedDeviceId,
                    category = state.category,
                    priority = state.priority,
                    title = title,
                    description = description,
                ),
            )
            _uiState.update { current ->
                when (outcome) {
                    is MaintenanceMutationOutcome.Success -> current.copy(
                        items = listOf(outcome.item) + current.items.filterNot { it.id == outcome.item.id },
                        openCount = (listOf(outcome.item) + current.items.filterNot { it.id == outcome.item.id }).countOpen(),
                        isCreateOpen = false,
                        isSubmitting = false,
                        formError = null,
                        message = "Đã gửi yêu cầu bảo trì.",
                    )
                    MaintenanceMutationOutcome.OpenRequestExists -> current.copy(
                        isSubmitting = false,
                        formError = "Thiết bị này đã có một yêu cầu bảo trì đang mở.",
                    )
                    MaintenanceMutationOutcome.InvalidState -> current.copy(
                        isSubmitting = false,
                        formError = "Trạng thái yêu cầu không còn cho phép thao tác này.",
                    )
                    MaintenanceMutationOutcome.ConnectionError -> current.copy(
                        isSubmitting = false,
                        formError = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    )
                    MaintenanceMutationOutcome.ServerError -> current.copy(
                        isSubmitting = false,
                        formError = "Không thể gửi yêu cầu bảo trì. Vui lòng thử lại sau.",
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
                    is MaintenanceMutationOutcome.Success -> {
                        val updated = current.items.map { if (it.id == outcome.item.id) outcome.item else it }
                        current.copy(
                            items = updated,
                            openCount = updated.countOpen(),
                            cancellingRequestId = null,
                            message = "Đã hủy yêu cầu bảo trì.",
                        )
                    }
                    MaintenanceMutationOutcome.InvalidState -> current.copy(
                        cancellingRequestId = null,
                        message = "Yêu cầu này không còn có thể hủy.",
                    )
                    MaintenanceMutationOutcome.ConnectionError -> current.copy(
                        cancellingRequestId = null,
                        message = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    )
                    else -> current.copy(
                        cancellingRequestId = null,
                        message = "Không thể hủy yêu cầu bảo trì.",
                    )
                }
            }
        }
    }

    private fun load(showLoading: Boolean = true) {
        if (showLoading) _uiState.update { it.copy(isLoading = true, loadError = null) }
        viewModelScope.launch {
            val outcome = repository.listMine()
            if (!showLoading && outcome !is MaintenanceListOutcome.Success) return@launch
            _uiState.update { current ->
                when (outcome) {
                    is MaintenanceListOutcome.Success -> current.copy(
                        isLoading = false,
                        items = outcome.items,
                        openCount = outcome.openCount,
                    )
                    MaintenanceListOutcome.ConnectionError -> current.copy(
                        isLoading = false,
                        loadError = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    )
                    MaintenanceListOutcome.ServerError -> current.copy(
                        isLoading = false,
                        loadError = "Không thể tải yêu cầu bảo trì. Vui lòng thử lại sau.",
                    )
                }
            }
        }
    }

    private inline fun updateForm(transform: MaintenanceRequestUiState.() -> MaintenanceRequestUiState) {
        _uiState.update(transform)
    }

    class Factory(
        private val repository: MaintenanceRequestRepository,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T =
            MaintenanceRequestViewModel(repository) as T
    }
}

private fun List<MaintenanceRequest>.countOpen() = count { it.status == "pending" || it.status == "in_progress" }
