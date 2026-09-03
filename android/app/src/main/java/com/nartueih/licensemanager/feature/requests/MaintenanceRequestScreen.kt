package com.nartueih.licensemanager.feature.requests

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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Build
import androidx.compose.material.icons.rounded.Devices
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuAnchorType
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.nartueih.licensemanager.data.devices.EmployeeDevice
import com.nartueih.licensemanager.data.maintenance.MaintenanceRequest
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun EmployeeMaintenanceRequestScreen(
    viewModel: MaintenanceRequestViewModel,
    devices: List<EmployeeDevice>,
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }
    var cancelCandidate by remember { mutableStateOf<MaintenanceRequest?>(null) }
    val openDeviceIds = state.items
        .filter { it.status == "pending" || it.status == "in_progress" }
        .mapTo(mutableSetOf(), MaintenanceRequest::deviceId)
    val availableDevices = devices.filterNot { it.id in openDeviceIds }

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.dismissMessage()
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
            when {
                state.isLoading -> CenteredRequestStatus {
                    CircularProgressIndicator()
                    Text("Đang tải yêu cầu bảo trì…")
                }
                state.loadError != null -> CenteredRequestStatus {
                    Text(
                        text = state.loadError.orEmpty(),
                        color = MaterialTheme.colorScheme.error,
                        textAlign = TextAlign.Center,
                    )
                    Button(onClick = viewModel::retry) { Text("Thử lại") }
                }
                else -> MaintenanceRequestList(
                    state = state,
                    canCreate = availableDevices.isNotEmpty(),
                    onCreate = { viewModel.openCreate(availableDevices.firstOrNull()?.id) },
                    onRefresh = viewModel::retry,
                    onCancel = { cancelCandidate = it },
                )
            }
        }
        SnackbarHost(
            hostState = snackbarHostState,
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .padding(16.dp),
        )
    }

    if (state.isCreateOpen) {
        MaintenanceCreateDialog(
            state = state,
            devices = availableDevices,
            onDismiss = viewModel::dismissCreate,
            onDeviceChanged = viewModel::updateDevice,
            onCategoryChanged = viewModel::updateCategory,
            onPriorityChanged = viewModel::updatePriority,
            onTitleChanged = viewModel::updateTitle,
            onDescriptionChanged = viewModel::updateDescription,
            onSubmit = viewModel::submit,
        )
    }

    cancelCandidate?.let { item ->
        AlertDialog(
            onDismissRequest = { cancelCandidate = null },
            title = { Text("Hủy yêu cầu bảo trì?") },
            text = { Text("Yêu cầu cho ${item.deviceAssetCode} sẽ được chuyển sang trạng thái đã hủy.") },
            confirmButton = {
                TextButton(
                    onClick = {
                        viewModel.cancel(item.id)
                        cancelCandidate = null
                    },
                ) { Text("Hủy yêu cầu", color = MaterialTheme.colorScheme.error) }
            },
            dismissButton = {
                TextButton(onClick = { cancelCandidate = null }) { Text("Quay lại") }
            },
        )
    }
}

@Composable
private fun MaintenanceRequestList(
    state: MaintenanceRequestUiState,
    canCreate: Boolean,
    onCreate: () -> Unit,
    onRefresh: () -> Unit,
    onCancel: (MaintenanceRequest) -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Column(modifier = Modifier.padding(top = 20.dp, bottom = 4.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            "Yêu cầu bảo trì",
                            style = MaterialTheme.typography.headlineMedium,
                            fontWeight = FontWeight.Bold,
                        )
                        Text(
                            "${state.openCount} yêu cầu đang mở",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 4.dp),
                        )
                    }
                    IconButton(onClick = onRefresh) {
                        Icon(Icons.Rounded.Refresh, contentDescription = "Làm mới")
                    }
                    Button(onClick = onCreate, enabled = canCreate) {
                        Icon(Icons.Rounded.Add, contentDescription = null, modifier = Modifier.size(18.dp))
                        Text("Báo sự cố", modifier = Modifier.padding(start = 6.dp))
                    }
                }
                if (!canCreate && state.items.isNotEmpty()) {
                    Text(
                        "Mỗi thiết bị của bạn đã có yêu cầu đang mở.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.padding(top = 8.dp),
                    )
                }
            }
        }
        if (state.items.isEmpty()) {
            item { EmptyMaintenanceCard(canCreate = canCreate, onCreate = onCreate) }
        } else {
            items(state.items, key = MaintenanceRequest::id) { item ->
                MaintenanceRequestCard(
                    item = item,
                    isCancelling = state.cancellingRequestId == item.id,
                    onCancel = { onCancel(item) },
                )
            }
        }
        item { Spacer(modifier = Modifier.height(16.dp)) }
    }
}

@Composable
private fun MaintenanceRequestCard(
    item: MaintenanceRequest,
    isCancelling: Boolean,
    onCancel: () -> Unit,
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
    ) {
        Column(
            modifier = Modifier.padding(18.dp),
            verticalArrangement = Arrangement.spacedBy(11.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(11.dp)) {
                Box(
                    modifier = Modifier
                        .size(44.dp)
                        .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.12f), CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.Rounded.Devices, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                }
                Column(modifier = Modifier.weight(1f)) {
                    Text(item.deviceName, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium)
                    Text(
                        "${item.deviceAssetCode} · Serial: ${item.deviceSerialNumber ?: "Chưa cập nhật"}",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                MaintenanceStatusPill(item.status)
            }

            Row(horizontalArrangement = Arrangement.spacedBy(7.dp)) {
                SmallPill(item.category.toCategoryLabel(), MaterialTheme.colorScheme.primary)
                SmallPill(item.priority.toPriorityLabel(), item.priority.toPriorityColor())
            }
            Text(item.title, fontWeight = FontWeight.Bold)
            Text(item.description, color = MaterialTheme.colorScheme.onSurfaceVariant)
            RequestDetailRow("Hãng / Model", listOfNotNull(item.deviceManufacturer, item.deviceModel).joinToString(" · ").ifBlank { item.deviceType })
            RequestDetailRow("Ngày mua", item.devicePurchasedAt.toDisplayDate())
            RequestDetailRow("Bảo hành", item.deviceWarrantyExpiresAt.toDisplayDate())
            RequestDetailRow("Ngày gửi", item.createdAt.toDisplayDateTime())

            item.assignedToName?.takeIf(String::isNotBlank)?.let {
                RequestDetailRow("Phụ trách", it)
            }
            item.responseNote?.takeIf(String::isNotBlank)?.let {
                Surface(
                    color = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.55f),
                    shape = RoundedCornerShape(12.dp),
                ) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Text("Phản hồi từ IT", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.labelLarge)
                        Text(it, modifier = Modifier.padding(top = 4.dp))
                    }
                }
            }
            item.lifecycleLabel()?.let {
                Text(it, color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.bodySmall)
            }
            if (item.status == "pending") {
                OutlinedButton(
                    onClick = onCancel,
                    enabled = !isCancelling,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (isCancelling) {
                        CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                    } else {
                        Text("Hủy yêu cầu", color = MaterialTheme.colorScheme.error)
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun MaintenanceCreateDialog(
    state: MaintenanceRequestUiState,
    devices: List<EmployeeDevice>,
    onDismiss: () -> Unit,
    onDeviceChanged: (String) -> Unit,
    onCategoryChanged: (String) -> Unit,
    onPriorityChanged: (String) -> Unit,
    onTitleChanged: (String) -> Unit,
    onDescriptionChanged: (String) -> Unit,
    onSubmit: () -> Unit,
) {
    val selectedDevice = devices.find { it.id == state.selectedDeviceId }
    Dialog(onDismissRequest = onDismiss) {
        Surface(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(24.dp),
            tonalElevation = 6.dp,
        ) {
            Column(
                modifier = Modifier
                    .padding(20.dp)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    Icon(Icons.Rounded.Build, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                    Column {
                        Text("Báo sự cố thiết bị", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
                        Text(
                            "Thông tin thiết bị được lưu tại thời điểm gửi.",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                }

                SelectorField(
                    label = "Thiết bị",
                    value = selectedDevice?.let { "${it.assetCode} · ${it.name}" }.orEmpty(),
                    options = devices.map { it.id to "${it.assetCode} · ${it.name}" },
                    enabled = !state.isSubmitting,
                    onSelected = onDeviceChanged,
                )
                selectedDevice?.let { DeviceSnapshot(it) }
                SelectorField(
                    label = "Nhóm sự cố",
                    value = state.category.toCategoryLabel(),
                    options = listOf(
                        "hardware" to "Phần cứng", "software" to "Phần mềm", "network" to "Mạng",
                        "accessory" to "Phụ kiện", "other" to "Khác",
                    ),
                    enabled = !state.isSubmitting,
                    onSelected = onCategoryChanged,
                )
                SelectorField(
                    label = "Mức ưu tiên",
                    value = state.priority.toPriorityLabel(),
                    options = listOf("normal" to "Bình thường", "high" to "Cao", "urgent" to "Khẩn cấp"),
                    enabled = !state.isSubmitting,
                    onSelected = onPriorityChanged,
                )
                OutlinedTextField(
                    value = state.title,
                    onValueChange = onTitleChanged,
                    label = { Text("Tiêu đề sự cố") },
                    placeholder = { Text("Không được bỏ trống") },
                    enabled = !state.isSubmitting,
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = state.description,
                    onValueChange = onDescriptionChanged,
                    label = { Text("Mô tả chi tiết") },
                    placeholder = { Text("Mô tả biểu hiện, thời điểm và ảnh hưởng — không được bỏ trống") },
                    enabled = !state.isSubmitting,
                    minLines = 3,
                    modifier = Modifier.fillMaxWidth(),
                )
                state.formError?.let {
                    Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                }
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
                    TextButton(onClick = onDismiss, enabled = !state.isSubmitting) { Text("Đóng") }
                    Button(
                        onClick = onSubmit,
                        enabled = !state.isSubmitting && state.selectedDeviceId.isNotBlank() &&
                            state.title.isNotBlank() && state.description.isNotBlank(),
                        modifier = Modifier.padding(start = 8.dp),
                    ) {
                        if (state.isSubmitting) {
                            CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                        } else {
                            Text("Gửi yêu cầu")
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SelectorField(
    label: String,
    value: String,
    options: List<Pair<String, String>>,
    enabled: Boolean,
    onSelected: (String) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    ExposedDropdownMenuBox(
        expanded = expanded,
        onExpandedChange = { if (enabled) expanded = !expanded },
    ) {
        OutlinedTextField(
            value = value,
            onValueChange = {},
            readOnly = true,
            enabled = enabled,
            label = { Text(label) },
            placeholder = { Text("Chọn $label — không được bỏ trống") },
            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded) },
            modifier = Modifier
                .menuAnchor(ExposedDropdownMenuAnchorType.PrimaryNotEditable, enabled = enabled)
                .fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEach { (key, optionLabel) ->
                DropdownMenuItem(
                    text = { Text(optionLabel) },
                    onClick = {
                        onSelected(key)
                        expanded = false
                    },
                )
            }
        }
    }
}

@Composable
private fun DeviceSnapshot(device: EmployeeDevice) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.55f),
        shape = RoundedCornerShape(14.dp),
    ) {
        Column(modifier = Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            RequestDetailRow("Serial", device.serialNumber ?: "Chưa cập nhật")
            RequestDetailRow("Loại", device.deviceType)
            RequestDetailRow("Hãng / Model", listOfNotNull(device.manufacturer, device.model).joinToString(" · ").ifBlank { "Chưa cập nhật" })
            RequestDetailRow("Ngày mua", device.purchasedAt.toDisplayDate())
            RequestDetailRow("Bảo hành", device.warrantyExpiresAt.toDisplayDate())
        }
    }
}

@Composable
private fun RequestDetailRow(label: String, value: String) {
    Row(modifier = Modifier.fillMaxWidth()) {
        Text(label, modifier = Modifier.weight(0.36f), color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, modifier = Modifier.weight(0.64f), fontWeight = FontWeight.Medium)
    }
}

@Composable
private fun MaintenanceStatusPill(status: String) {
    val (label, color) = when (status) {
        "pending" -> "Đang chờ" to Color(0xFFD06B12)
        "in_progress" -> "Đang xử lý" to MaterialTheme.colorScheme.primary
        "completed" -> "Hoàn thành" to Color(0xFF16835B)
        "rejected" -> "Từ chối" to MaterialTheme.colorScheme.error
        "cancelled" -> "Đã hủy" to MaterialTheme.colorScheme.onSurfaceVariant
        else -> status to MaterialTheme.colorScheme.onSurfaceVariant
    }
    SmallPill(label, color)
}

@Composable
private fun SmallPill(label: String, color: Color) {
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
private fun EmptyMaintenanceCard(canCreate: Boolean, onCreate: () -> Unit) {
    Card(modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(20.dp)) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(28.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icon(Icons.Rounded.Build, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
            Text("Chưa có yêu cầu bảo trì", fontWeight = FontWeight.Bold)
            Text(
                "Khi thiết bị gặp sự cố, bạn có thể gửi thông tin tới bộ phận IT tại đây.",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
            )
            OutlinedButton(onClick = onCreate, enabled = canCreate) {
                Text(if (canCreate) "Báo sự cố đầu tiên" else "Chưa có thiết bị để yêu cầu")
            }
        }
    }
}

@Composable
private fun CenteredRequestStatus(content: @Composable () -> Unit) {
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

private fun String.toCategoryLabel() = when (this) {
    "hardware" -> "Phần cứng"
    "software" -> "Phần mềm"
    "network" -> "Mạng"
    "accessory" -> "Phụ kiện"
    "other" -> "Khác"
    else -> this
}

private fun String.toPriorityLabel() = when (this) {
    "urgent" -> "Khẩn cấp"
    "high" -> "Cao"
    else -> "Bình thường"
}

@Composable
private fun String.toPriorityColor() = when (this) {
    "urgent" -> MaterialTheme.colorScheme.error
    "high" -> Color(0xFFD06B12)
    else -> MaterialTheme.colorScheme.onSurfaceVariant
}

private fun String?.toDisplayDate(): String {
    if (this.isNullOrBlank()) return "Chưa cập nhật"
    return runCatching { LocalDate.parse(this).format(dateFormatter) }.getOrDefault(this)
}

private fun String.toDisplayDateTime(): String = runCatching {
    Instant.parse(this).atZone(ZoneId.systemDefault()).format(dateTimeFormatter)
}.getOrDefault(this)

private fun MaintenanceRequest.lifecycleLabel(): String? = when (status) {
    "in_progress" -> acceptedAt?.let { "Tiếp nhận ${it.toDisplayDateTime()}" }
    "completed" -> completedAt?.let { "Hoàn thành ${it.toDisplayDateTime()}" }
    "rejected" -> rejectedAt?.let { "Từ chối ${it.toDisplayDateTime()}" }
    "cancelled" -> cancelledAt?.let { "Đã hủy ${it.toDisplayDateTime()}" }
    else -> null
}

private val dateFormatter = DateTimeFormatter.ofPattern("dd/MM/yyyy")
private val dateTimeFormatter = DateTimeFormatter.ofPattern("dd/MM/yyyy HH:mm")
