package com.nartueih.licensemanager.feature.notifications

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.nartueih.licensemanager.data.notifications.EmployeeNotification
import com.nartueih.licensemanager.data.notifications.NotificationListOutcome
import com.nartueih.licensemanager.data.notifications.NotificationMarkAllOutcome
import com.nartueih.licensemanager.data.notifications.NotificationMutationOutcome
import com.nartueih.licensemanager.data.notifications.NotificationRepository
import java.time.Instant
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class NotificationUiState(
    val isLoading: Boolean = true,
    val items: List<EmployeeNotification> = emptyList(),
    val unreadCount: Int = 0,
    val actionId: String? = null,
    val isMarkingAll: Boolean = false,
    val error: String? = null,
)

class NotificationViewModel(
    private val repository: NotificationRepository,
    private val now: () -> String = { Instant.now().toString() },
) : ViewModel() {
    private val _uiState = MutableStateFlow(NotificationUiState())
    val uiState: StateFlow<NotificationUiState> = _uiState.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        load(showLoading = true)
    }

    fun sync() {
        load(showLoading = false)
    }

    private fun load(showLoading: Boolean) {
        if (showLoading) _uiState.update { it.copy(isLoading = true, error = null) }
        viewModelScope.launch {
            val outcome = repository.list()
            if (!showLoading && outcome !is NotificationListOutcome.Success) return@launch
            _uiState.update { current ->
                when (outcome) {
                    is NotificationListOutcome.Success -> current.copy(
                        isLoading = false,
                        items = outcome.items,
                        unreadCount = outcome.unreadCount,
                    )
                    NotificationListOutcome.ConnectionError -> current.copy(
                        isLoading = false,
                        error = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    )
                    NotificationListOutcome.ServerError -> current.copy(
                        isLoading = false,
                        error = "Không thể tải thông báo. Vui lòng thử lại sau.",
                    )
                }
            }
        }
    }

    fun markRead(notificationId: String) {
        val currentItem = _uiState.value.items.find { it.id == notificationId } ?: return
        if (currentItem.readAt != null || _uiState.value.actionId != null) return
        _uiState.update { it.copy(actionId = notificationId, error = null) }
        viewModelScope.launch {
            _uiState.update { current ->
                when (val outcome = repository.markRead(notificationId)) {
                    is NotificationMutationOutcome.Success -> current.copy(
                        items = current.items.map { if (it.id == outcome.item.id) outcome.item else it },
                        unreadCount = (current.unreadCount - 1).coerceAtLeast(0),
                        actionId = null,
                    )
                    NotificationMutationOutcome.ConnectionError -> current.copy(
                        actionId = null,
                        error = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    )
                    NotificationMutationOutcome.ServerError -> current.copy(
                        actionId = null,
                        error = "Không thể đánh dấu thông báo đã đọc.",
                    )
                }
            }
        }
    }

    fun markAllRead() {
        if (_uiState.value.unreadCount == 0 || _uiState.value.isMarkingAll) return
        _uiState.update { it.copy(isMarkingAll = true, error = null) }
        viewModelScope.launch {
            _uiState.update { current ->
                when (repository.markAllRead()) {
                    is NotificationMarkAllOutcome.Success -> {
                        val readAt = now()
                        current.copy(
                            items = current.items.map { if (it.readAt == null) it.copy(readAt = readAt) else it },
                            unreadCount = 0,
                            isMarkingAll = false,
                        )
                    }
                    NotificationMarkAllOutcome.ConnectionError -> current.copy(
                        isMarkingAll = false,
                        error = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    )
                    NotificationMarkAllOutcome.ServerError -> current.copy(
                        isMarkingAll = false,
                        error = "Không thể đánh dấu tất cả thông báo.",
                    )
                }
            }
        }
    }

    fun dismissError() = _uiState.update { it.copy(error = null) }

    class Factory(
        private val repository: NotificationRepository,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T = NotificationViewModel(repository) as T
    }
}
