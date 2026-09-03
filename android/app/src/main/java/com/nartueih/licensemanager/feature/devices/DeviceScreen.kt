package com.nartueih.licensemanager.feature.devices

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Build
import androidx.compose.material.icons.rounded.Devices
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.nartueih.licensemanager.data.devices.EmployeeDevice

@Composable
fun EmployeeDeviceScreen(
    viewModel: DeviceViewModel,
    onMaintenanceRequested: (EmployeeDevice) -> Unit,
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()

    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
        when {
            state.isLoading -> CenteredDeviceStatus {
                CircularProgressIndicator()
                Text("Đang tải thiết bị của bạn…")
            }
            state.error != null -> CenteredDeviceStatus {
                Text(
                    text = state.error.orEmpty(),
                    color = MaterialTheme.colorScheme.error,
                    textAlign = TextAlign.Center,
                )
                Button(onClick = viewModel::retry) { Text("Thử lại") }
            }
            else -> DeviceList(
                items = state.items,
                onMaintenanceRequested = onMaintenanceRequested,
            )
        }
    }
}

@Composable
private fun DeviceList(
    items: List<EmployeeDevice>,
    onMaintenanceRequested: (EmployeeDevice) -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Column(modifier = Modifier.padding(top = 20.dp, bottom = 4.dp)) {
                Text(
                    text = "Thiết bị của tôi",
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = "${items.size} thiết bị đang được giao",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 4.dp),
                )
            }
        }
        if (items.isEmpty()) {
            item { EmptyDeviceCard() }
        } else {
            items(items, key = EmployeeDevice::id) { device ->
                DeviceCard(
                    item = device,
                    onMaintenanceRequested = { onMaintenanceRequested(device) },
                )
            }
        }
        item { Spacer(modifier = Modifier.height(16.dp)) }
    }
}

@Composable
private fun DeviceCard(
    item: EmployeeDevice,
    onMaintenanceRequested: () -> Unit,
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
    ) {
        Column(
            modifier = Modifier.padding(18.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Box(
                    modifier = Modifier
                        .size(46.dp)
                        .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.12f), CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = Icons.Rounded.Devices,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                    )
                }
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = item.name,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        text = item.assetCode,
                        color = MaterialTheme.colorScheme.primary,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
                DeviceStatusPill(item.status)
            }

            DeviceDetailRow("Loại", item.deviceType.toDeviceTypeLabel())
            DeviceDetailRow("Hãng / Model", listOfNotNull(item.manufacturer, item.model).joinToString(" · ").ifBlank { "Chưa cập nhật" })
            DeviceDetailRow("Serial", item.serialNumber?.takeIf(String::isNotBlank) ?: "Chưa cập nhật")
            DeviceDetailRow("Ngày mua", item.purchasedAt?.takeIf(String::isNotBlank) ?: "Chưa cập nhật")
            DeviceDetailRow("Hết bảo hành", item.warrantyExpiresAt?.takeIf(String::isNotBlank) ?: "Chưa cập nhật")

            OutlinedButton(
                onClick = onMaintenanceRequested,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Icon(
                    imageVector = Icons.Rounded.Build,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp),
                )
                Text("Yêu cầu bảo trì", modifier = Modifier.padding(start = 8.dp))
            }
        }
    }
}

@Composable
private fun DeviceDetailRow(label: String, value: String) {
    Row(modifier = Modifier.fillMaxWidth()) {
        Text(
            text = label,
            modifier = Modifier.weight(0.36f),
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(
            text = value,
            modifier = Modifier.weight(0.64f),
            fontWeight = FontWeight.Medium,
        )
    }
}

@Composable
private fun DeviceStatusPill(status: String) {
    val (label, color) = when (status) {
        "assigned" -> "Đang sử dụng" to Color(0xFF16835B)
        "available" -> "Sẵn sàng" to MaterialTheme.colorScheme.primary
        "maintenance" -> "Bảo trì" to Color(0xFFD06B12)
        "retired" -> "Ngừng sử dụng" to MaterialTheme.colorScheme.onSurfaceVariant
        "lost" -> "Thất lạc" to MaterialTheme.colorScheme.error
        else -> status to MaterialTheme.colorScheme.onSurfaceVariant
    }
    Text(
        text = label,
        modifier = Modifier
            .background(color.copy(alpha = 0.12f), RoundedCornerShape(50))
            .padding(horizontal = 9.dp, vertical = 5.dp),
        color = color,
        style = MaterialTheme.typography.labelMedium,
        fontWeight = FontWeight.SemiBold,
    )
}

@Composable
private fun EmptyDeviceCard() {
    Card(modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(20.dp)) {
        Text(
            text = "Bạn chưa được giao thiết bị nào.",
            modifier = Modifier
                .fillMaxWidth()
                .padding(28.dp),
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun CenteredDeviceStatus(content: @Composable () -> Unit) {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .padding(28.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) { content() }
    }
}

private fun String.toDeviceTypeLabel() = when (this) {
    "laptop" -> "Laptop"
    "desktop" -> "Máy tính bàn"
    "workstation" -> "Máy trạm"
    "server" -> "Máy chủ"
    "mobile" -> "Điện thoại"
    "tablet" -> "Máy tính bảng"
    else -> this
}
