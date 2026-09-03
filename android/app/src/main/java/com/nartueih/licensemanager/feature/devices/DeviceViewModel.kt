package com.nartueih.licensemanager.feature.devices

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.nartueih.licensemanager.data.devices.DeviceListOutcome
import com.nartueih.licensemanager.data.devices.EmployeeDevice
import com.nartueih.licensemanager.data.devices.EmployeeDeviceRepository
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class DeviceUiState(
    val isLoading: Boolean = true,
    val items: List<EmployeeDevice> = emptyList(),
    val error: String? = null,
)

class DeviceViewModel(
    private val repository: EmployeeDeviceRepository,
) : ViewModel() {
    private val _uiState = MutableStateFlow(DeviceUiState())
    val uiState: StateFlow<DeviceUiState> = _uiState.asStateFlow()

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
        if (showLoading) _uiState.update { it.copy(isLoading = true, error = null) }
        viewModelScope.launch {
            val outcome = repository.list()
            if (!showLoading && outcome !is DeviceListOutcome.Success) return@launch
            _uiState.value = when (outcome) {
                is DeviceListOutcome.Success -> DeviceUiState(items = outcome.items, isLoading = false)
                DeviceListOutcome.ConnectionError -> DeviceUiState(
                    isLoading = false,
                    error = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                )
                DeviceListOutcome.ServerError -> DeviceUiState(
                    isLoading = false,
                    error = "Không thể tải danh sách thiết bị. Vui lòng thử lại sau.",
                )
            }
        }
    }

    class Factory(
        private val repository: EmployeeDeviceRepository,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            return DeviceViewModel(repository) as T
        }
    }
}
