package com.nartueih.licensemanager.feature.notifications

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Build
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.Error
import androidx.compose.material.icons.rounded.Notifications
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.nartueih.licensemanager.data.notifications.EmployeeNotification
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EmployeeNotificationTopBar(
    viewModel: NotificationViewModel,
    onNotificationSelected: (EmployeeNotification) -> Unit,
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    var isOpen by remember { mutableStateOf(false) }

    TopAppBar(
        title = {
            Column {
                Text("License Manager", fontWeight = FontWeight.Bold)
                Text(
                    "Không gian nhân viên",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    style = MaterialTheme.typography.labelMedium,
                )
            }
        },
        actions = {
            IconButton(
                onClick = {
                    isOpen = true
                    viewModel.refresh()
                },
            ) {
                BadgedBox(
                    badge = {
                        if (state.unreadCount > 0) {
                            Badge { Text(if (state.unreadCount > 99) "99+" else state.unreadCount.toString()) }
                        }
                    },
                ) {
                    Icon(Icons.Rounded.Notifications, contentDescription = "Thông báo")
                }
            }
        },
    )

    if (isOpen) {
        NotificationBottomSheet(
            state = state,
            onDismiss = { isOpen = false },
            onRefresh = viewModel::refresh,
            onMarkAllRead = viewModel::markAllRead,
            onNotificationSelected = { item ->
                viewModel.markRead(item.id)
                onNotificationSelected(item)
                isOpen = false
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun NotificationBottomSheet(
    state: NotificationUiState,
    onDismiss: () -> Unit,
    onRefresh: () -> Unit,
    onMarkAllRead: () -> Unit,
    onNotificationSelected: (EmployeeNotification) -> Unit,
) {
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(modifier = Modifier.fillMaxWidth()) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 20.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text("Thông báo", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                    Text(
                        "${state.unreadCount} thông báo chưa đọc",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                IconButton(onClick = onRefresh, enabled = !state.isLoading) {
                    Icon(Icons.Rounded.Refresh, contentDescription = "Làm mới thông báo")
                }
            }
            if (state.unreadCount > 0) {
                TextButton(
                    onClick = onMarkAllRead,
                    enabled = !state.isMarkingAll,
                    modifier = Modifier
                        .align(Alignment.End)
                        .padding(horizontal = 12.dp),
                ) {
                    if (state.isMarkingAll) {
                        CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                    } else {
                        Text("Đánh dấu tất cả đã đọc")
                    }
                }
            }
            state.error?.let {
                Surface(
                    color = MaterialTheme.colorScheme.errorContainer,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 20.dp, vertical = 6.dp),
                ) {
                    Row(
                        modifier = Modifier.padding(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Icon(Icons.Rounded.Error, contentDescription = null, tint = MaterialTheme.colorScheme.error)
                        Text(it, modifier = Modifier.weight(1f), color = MaterialTheme.colorScheme.onErrorContainer)
                        TextButton(onClick = onRefresh) { Text("Thử lại") }
                    }
                }
            }
            when {
                state.isLoading -> Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = 220.dp),
                    contentAlignment = Alignment.Center,
                ) { CircularProgressIndicator() }

                state.items.isEmpty() -> EmptyNotifications()

                else -> LazyColumn(
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(max = 560.dp),
                ) {
                    items(state.items, key = EmployeeNotification::id) { item ->
                        NotificationRow(
                            item = item,
                            isBusy = state.actionId == item.id,
                            onClick = { onNotificationSelected(item) },
                        )
                        HorizontalDivider()
                    }
                }
            }
        }
    }
}

@Composable
private fun NotificationRow(
    item: EmployeeNotification,
    isBusy: Boolean,
    onClick: () -> Unit,
) {
    val (icon, color) = item.notificationVisual()
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                if (item.readAt == null) MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.28f)
                else Color.Transparent,
            )
            .clickable(enabled = !isBusy, onClick = onClick)
            .padding(horizontal = 20.dp, vertical = 14.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Box(
            modifier = Modifier
                .size(42.dp)
                .background(color.copy(alpha = 0.12f), CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            if (isBusy) {
                CircularProgressIndicator(modifier = Modifier.size(20.dp), strokeWidth = 2.dp)
            } else {
                Icon(icon, contentDescription = null, tint = color, modifier = Modifier.size(22.dp))
            }
        }
        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    item.title,
                    modifier = Modifier.weight(1f),
                    fontWeight = if (item.readAt == null) FontWeight.Bold else FontWeight.SemiBold,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                if (item.readAt == null) {
                    Box(
                        modifier = Modifier
                            .padding(start = 8.dp)
                            .size(8.dp)
                            .background(MaterialTheme.colorScheme.primary, CircleShape),
                    )
                }
            }
            Text(item.message, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Text(
                item.createdAt.toNotificationDateTime(),
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

@Composable
private fun EmptyNotifications() {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = 240.dp)
            .padding(28.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(Icons.Rounded.Notifications, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
        Text("Chưa có thông báo", modifier = Modifier.padding(top = 10.dp), fontWeight = FontWeight.Bold)
        Text(
            "Phản hồi từ bộ phận IT sẽ xuất hiện tại đây.",
            modifier = Modifier.padding(top = 4.dp),
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun EmployeeNotification.notificationVisual(): Pair<ImageVector, Color> = when (type) {
    "license_request_approved", "maintenance_completed" -> Icons.Rounded.CheckCircle to Color(0xFF16835B)
    "maintenance_accepted" -> Icons.Rounded.Build to MaterialTheme.colorScheme.primary
    else -> Icons.Rounded.Error to MaterialTheme.colorScheme.error
}

private fun String.toNotificationDateTime(): String = runCatching {
    Instant.parse(this).atZone(ZoneId.systemDefault()).format(notificationDateTimeFormatter)
}.getOrDefault(this)

private val notificationDateTimeFormatter = DateTimeFormatter.ofPattern("dd/MM/yyyy HH:mm")
