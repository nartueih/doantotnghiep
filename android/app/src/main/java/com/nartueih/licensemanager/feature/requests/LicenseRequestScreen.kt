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
import androidx.compose.material.icons.automirrored.rounded.Assignment
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.VpnKey
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuAnchorType
import androidx.compose.material3.ExposedDropdownMenuBox
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
import com.nartueih.licensemanager.data.licenserequests.LicenseRequest
import com.nartueih.licensemanager.data.licenserequests.RequestableSoftware
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun EmployeeLicenseRequestScreen(viewModel: LicenseRequestViewModel) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }
    var cancelCandidate by remember { mutableStateOf<LicenseRequest?>(null) }
    val pendingProductIds = state.items
        .filter { it.status == "pending" }
        .mapTo(mutableSetOf(), LicenseRequest::softwareProductId)
    val availableSoftware = state.software.filterNot { it.id in pendingProductIds }

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.dismissMessage()
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
            when {
                state.isLoading -> CenteredLicenseRequestStatus {
                    CircularProgressIndicator()
                    Text("Đang tải yêu cầu license…")
                }
                state.loadError != null -> CenteredLicenseRequestStatus {
                    Text(
                        state.loadError.orEmpty(),
                        color = MaterialTheme.colorScheme.error,
                        textAlign = TextAlign.Center,
                    )
                    Button(onClick = viewModel::retry) { Text("Thử lại") }
                }
                else -> LicenseRequestList(
                    state = state,
                    canCreate = availableSoftware.isNotEmpty(),
                    onCreate = { viewModel.openCreate(availableSoftware.firstOrNull()?.id) },
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
        LicenseRequestCreateDialog(
            state = state,
            software = availableSoftware,
            onDismiss = viewModel::dismissCreate,
            onSoftwareChanged = viewModel::updateSoftware,
            onPriorityChanged = viewModel::updatePriority,
            onReasonChanged = viewModel::updateReason,
            onSubmit = viewModel::submit,
        )
    }

    cancelCandidate?.let { item ->
        AlertDialog(
            onDismissRequest = { cancelCandidate = null },
            title = { Text("Hủy yêu cầu cấp license?") },
            text = { Text("Yêu cầu ${item.softwareProductName} sẽ được chuyển sang trạng thái đã hủy.") },
            confirmButton = {
                TextButton(
                    onClick = {
                        viewModel.cancel(item.id)
                        cancelCandidate = null
                    },
                ) { Text("Hủy yêu cầu", color = MaterialTheme.colorScheme.error) }
            },
            dismissButton = { TextButton(onClick = { cancelCandidate = null }) { Text("Quay lại") } },
        )
    }
}

@Composable
private fun LicenseRequestList(
    state: LicenseRequestUiState,
    canCreate: Boolean,
    onCreate: () -> Unit,
    onRefresh: () -> Unit,
    onCancel: (LicenseRequest) -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Column(modifier = Modifier.padding(top = 16.dp, bottom = 4.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            "Yêu cầu cấp license",
                            style = MaterialTheme.typography.headlineMedium,
                            fontWeight = FontWeight.Bold,
                        )
                        Text(
                            "${state.pendingCount} yêu cầu đang chờ",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 4.dp),
                        )
                    }
                    IconButton(onClick = onRefresh) {
                        Icon(Icons.Rounded.Refresh, contentDescription = "Làm mới")
                    }
                    Button(onClick = onCreate, enabled = canCreate) {
                        Icon(Icons.Rounded.Add, contentDescription = null, modifier = Modifier.size(18.dp))
                        Text("Tạo yêu cầu", modifier = Modifier.padding(start = 6.dp))
                    }
                }
                if (!canCreate && state.software.isNotEmpty()) {
                    Text(
                        "Các phần mềm hiện có đều đã được gửi yêu cầu.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        style = MaterialTheme.typography.bodySmall,
                        modifier = Modifier.padding(top = 8.dp),
                    )
                }
            }
        }
        if (state.items.isEmpty()) {
            item { EmptyLicenseRequestCard(canCreate, onCreate) }
        } else {
            items(state.items, key = LicenseRequest::id) { item ->
                LicenseRequestCard(
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
private fun LicenseRequestCard(
    item: LicenseRequest,
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
                    Icon(Icons.Rounded.VpnKey, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                }
                Column(modifier = Modifier.weight(1f)) {
                    Text(item.softwareProductName, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium)
                    Text(
                        "Gửi ${item.createdAt.toLicenseRequestDateTime()}",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                LicenseRequestStatusPill(item.status)
            }

            SmallLicenseRequestPill(item.priority.toLicenseRequestPriorityLabel(), item.priority.toLicenseRequestPriorityColor())
            Text(item.reason, color = MaterialTheme.colorScheme.onSurfaceVariant)

            when (item.status) {
                "approved" -> RequestResultCard(
                    title = item.selectedLicenseName ?: "License đã được cấp",
                    message = item.responseNote ?: "IT đã duyệt và cấp license cho bạn.",
                    color = Color(0xFF16835B),
                )
                "rejected" -> RequestResultCard(
                    title = item.decisionReason.toDecisionReasonLabel(),
                    message = item.responseNote ?: "Yêu cầu đã bị từ chối.",
                    color = MaterialTheme.colorScheme.error,
                )
                "cancelled" -> Text(
                    "Bạn đã hủy yêu cầu này.",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            item.reviewedByName?.takeIf(String::isNotBlank)?.let {
                LicenseRequestDetailRow("Người duyệt", it)
            }
            item.reviewedAt?.let { LicenseRequestDetailRow("Xử lý lúc", it.toLicenseRequestDateTime()) }
            item.cancelledAt?.let { LicenseRequestDetailRow("Hủy lúc", it.toLicenseRequestDateTime()) }

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

@Composable
private fun RequestResultCard(title: String, message: String, color: Color) {
    Surface(color = color.copy(alpha = 0.1f), shape = RoundedCornerShape(12.dp)) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(title, color = color, fontWeight = FontWeight.Bold)
            Text(message, modifier = Modifier.padding(top = 4.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun LicenseRequestCreateDialog(
    state: LicenseRequestUiState,
    software: List<RequestableSoftware>,
    onDismiss: () -> Unit,
    onSoftwareChanged: (String) -> Unit,
    onPriorityChanged: (String) -> Unit,
    onReasonChanged: (String) -> Unit,
    onSubmit: () -> Unit,
) {
    val selectedSoftware = software.find { it.id == state.selectedSoftwareId }
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
                    Icon(Icons.AutoMirrored.Rounded.Assignment, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                    Column {
                        Text("Tạo yêu cầu cấp license", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
                        Text(
                            "Cho bộ phận IT biết phần mềm và mục đích sử dụng.",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                }
                LicenseRequestSelector(
                    label = "Phần mềm",
                    value = selectedSoftware?.let { "${it.name} · ${it.publisher}" }.orEmpty(),
                    options = software.map { it.id to "${it.name} · ${it.publisher}" },
                    enabled = !state.isSubmitting,
                    onSelected = onSoftwareChanged,
                )
                selectedSoftware?.let { SoftwareSnapshot(it) }
                LicenseRequestSelector(
                    label = "Mức ưu tiên",
                    value = state.priority.toLicenseRequestPriorityLabel(),
                    options = listOf("normal" to "Bình thường", "high" to "Cao", "urgent" to "Khẩn cấp"),
                    enabled = !state.isSubmitting,
                    onSelected = onPriorityChanged,
                )
                OutlinedTextField(
                    value = state.reason,
                    onValueChange = onReasonChanged,
                    label = { Text("Lý do sử dụng") },
                    placeholder = { Text("Nhập mục đích công việc — không được bỏ trống") },
                    enabled = !state.isSubmitting,
                    minLines = 4,
                    modifier = Modifier.fillMaxWidth(),
                )
                state.formError?.let {
                    Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                }
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
                    TextButton(onClick = onDismiss, enabled = !state.isSubmitting) { Text("Đóng") }
                    Button(
                        onClick = onSubmit,
                        enabled = !state.isSubmitting && state.selectedSoftwareId.isNotBlank() && state.reason.isNotBlank(),
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
private fun LicenseRequestSelector(
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
private fun SoftwareSnapshot(item: RequestableSoftware) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.55f),
        shape = RoundedCornerShape(14.dp),
    ) {
        Column(modifier = Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            LicenseRequestDetailRow("Nhà phát hành", item.publisher)
            LicenseRequestDetailRow("Phiên bản", item.version.ifBlank { "Chưa cập nhật" })
            item.description.takeIf(String::isNotBlank)?.let { LicenseRequestDetailRow("Mô tả", it) }
        }
    }
}

@Composable
private fun LicenseRequestDetailRow(label: String, value: String) {
    Row(modifier = Modifier.fillMaxWidth()) {
        Text(label, modifier = Modifier.weight(0.36f), color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, modifier = Modifier.weight(0.64f), fontWeight = FontWeight.Medium)
    }
}

@Composable
private fun LicenseRequestStatusPill(status: String) {
    val (label, color) = when (status) {
        "pending" -> "Đang chờ" to Color(0xFFD06B12)
        "approved" -> "Đã duyệt" to Color(0xFF16835B)
        "rejected" -> "Từ chối" to MaterialTheme.colorScheme.error
        "cancelled" -> "Đã hủy" to MaterialTheme.colorScheme.onSurfaceVariant
        else -> status to MaterialTheme.colorScheme.onSurfaceVariant
    }
    SmallLicenseRequestPill(label, color)
}

@Composable
private fun SmallLicenseRequestPill(label: String, color: Color) {
    Text(
        label,
        modifier = Modifier
            .background(color.copy(alpha = 0.12f), RoundedCornerShape(50))
            .padding(horizontal = 9.dp, vertical = 5.dp),
        color = color,
        style = MaterialTheme.typography.labelMedium,
        fontWeight = FontWeight.SemiBold,
    )
}

@Composable
private fun EmptyLicenseRequestCard(canCreate: Boolean, onCreate: () -> Unit) {
    Card(modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(20.dp)) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(28.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icon(Icons.AutoMirrored.Rounded.Assignment, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
            Text("Chưa có yêu cầu cấp license", fontWeight = FontWeight.Bold)
            Text(
                "Khi cần thêm phần mềm cho công việc, hãy gửi yêu cầu để bộ phận IT xử lý.",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
            )
            OutlinedButton(onClick = onCreate, enabled = canCreate) {
                Text(if (canCreate) "Tạo yêu cầu đầu tiên" else "Chưa có phần mềm có thể yêu cầu")
            }
        }
    }
}

@Composable
private fun CenteredLicenseRequestStatus(content: @Composable () -> Unit) {
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

private fun String.toLicenseRequestPriorityLabel() = when (this) {
    "urgent" -> "Khẩn cấp"
    "high" -> "Cao"
    else -> "Bình thường"
}

@Composable
private fun String.toLicenseRequestPriorityColor() = when (this) {
    "urgent" -> MaterialTheme.colorScheme.error
    "high" -> Color(0xFFD06B12)
    else -> MaterialTheme.colorScheme.onSurfaceVariant
}

private fun String?.toDecisionReasonLabel() = when (this) {
    "out_of_stock" -> "Tạm hết license"
    "not_approved" -> "Không được phê duyệt"
    "other" -> "Lý do khác"
    else -> "Yêu cầu đã bị từ chối"
}

private fun String.toLicenseRequestDateTime(): String = runCatching {
    Instant.parse(this).atZone(ZoneId.systemDefault()).format(licenseRequestDateTimeFormatter)
}.getOrDefault(this)

private val licenseRequestDateTimeFormatter = DateTimeFormatter.ofPattern("dd/MM/yyyy HH:mm")
