package com.nartueih.licensemanager.feature.licenses

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
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.nartueih.licensemanager.data.licenses.EmployeeLicense

@Composable
fun EmployeeLicenseScreen(viewModel: LicenseViewModel) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()

    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
        when {
            state.isLoading -> CenteredStatus {
                CircularProgressIndicator()
                Text("Đang tải license của bạn…")
            }
            state.loadError != null -> CenteredStatus {
                Text(
                    text = state.loadError.orEmpty(),
                    color = MaterialTheme.colorScheme.error,
                    textAlign = TextAlign.Center,
                )
                Button(onClick = viewModel::retry) { Text("Thử lại") }
            }
            else -> LicenseList(
                items = state.items,
                revealingAssignmentId = state.revealingAssignmentId,
                onRevealKey = viewModel::revealKey,
                onRefresh = viewModel::retry,
            )
        }
    }

    state.revealedKey?.let { revealed ->
        LicenseKeyDialog(
            licenseName = revealed.licenseName,
            licenseKey = revealed.licenseKey,
            onDismiss = viewModel::dismissKeyResult,
        )
    }
    state.keyError?.let { message ->
        AlertDialog(
            onDismissRequest = viewModel::dismissKeyResult,
            title = { Text("Không thể xem key") },
            text = { Text(message) },
            confirmButton = {
                TextButton(onClick = viewModel::dismissKeyResult) { Text("Đóng") }
            },
        )
    }
}

@Composable
private fun LicenseList(
    items: List<EmployeeLicense>,
    revealingAssignmentId: String?,
    onRevealKey: (String) -> Unit,
    onRefresh: () -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Row(
                modifier = Modifier.padding(top = 20.dp, bottom = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text("License của tôi", style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
                    Text(
                        text = "${items.size} license đang được cấp",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 4.dp),
                    )
                }
                IconButton(onClick = onRefresh) {
                    Icon(Icons.Rounded.Refresh, contentDescription = "Làm mới")
                }
            }
        }
        if (items.isEmpty()) {
            item { EmptyLicenseCard() }
        } else {
            items(items, key = EmployeeLicense::assignmentId) { license ->
                LicenseCard(
                    item = license,
                    isRevealing = revealingAssignmentId == license.assignmentId,
                    onRevealKey = { onRevealKey(license.assignmentId) },
                )
            }
        }
        item { Spacer(modifier = Modifier.height(16.dp)) }
    }
}

@Composable
private fun LicenseCard(item: EmployeeLicense, isRevealing: Boolean, onRevealKey: () -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
    ) {
        Column(modifier = Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Box(
                    modifier = Modifier
                        .size(44.dp)
                        .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.12f), CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = item.licenseName.firstOrNull()?.uppercase() ?: "L",
                        color = MaterialTheme.colorScheme.primary,
                        fontWeight = FontWeight.Bold,
                    )
                }
                Column(modifier = Modifier.weight(1f)) {
                    Text(item.licenseName, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Text(item.licenseType.toDisplayLicenseType(), color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                StatusPill(item.lifecycleStatus)
            }
            DetailRow("Thời hạn", item.expiresAt ?: "Không giới hạn")
            DetailRow(
                "Cấp cho",
                if (item.assignmentSource == "device") {
                    "Thiết bị ${item.deviceAssetCode ?: "chưa xác định"}"
                } else {
                    "Tài khoản của bạn"
                },
            )
            item.notes?.takeIf(String::isNotBlank)?.let { DetailRow("Ghi chú", it) }
            if (item.canViewKey) {
                OutlinedButton(
                    onClick = onRevealKey,
                    enabled = !isRevealing,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (isRevealing) {
                        CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                    } else {
                        Text("Xem key kích hoạt")
                    }
                }
            } else {
                Text(
                    text = "Key do IT quản lý",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun DetailRow(label: String, value: String) {
    Row(modifier = Modifier.fillMaxWidth()) {
        Text(label, modifier = Modifier.weight(0.34f), color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, modifier = Modifier.weight(0.66f), fontWeight = FontWeight.Medium)
    }
}

@Composable
private fun StatusPill(status: String) {
    val (label, color) = when (status) {
        "active" -> "Đang dùng" to Color(0xFF16835B)
        "upcoming" -> "Sắp hiệu lực" to Color(0xFFD06B12)
        "expired" -> "Hết hạn" to MaterialTheme.colorScheme.error
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
private fun EmptyLicenseCard() {
    Card(modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(20.dp)) {
        Text(
            text = "Bạn chưa được cấp license nào.",
            modifier = Modifier
                .fillMaxWidth()
                .padding(28.dp),
            textAlign = TextAlign.Center,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Suppress("DEPRECATION")
@Composable
private fun LicenseKeyDialog(licenseName: String, licenseKey: String, onDismiss: () -> Unit) {
    val clipboard = LocalClipboardManager.current
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(licenseName) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Key kích hoạt")
                SelectionContainer {
                    Text(licenseKey, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                }
                Text(
                    text = "Không chia sẻ key này với người khác.",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        },
        confirmButton = {
            TextButton(onClick = { clipboard.setText(AnnotatedString(licenseKey)) }) { Text("Sao chép") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Đóng") } },
    )
}

@Composable
private fun CenteredStatus(content: @Composable () -> Unit) {
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

private fun String.toDisplayLicenseType() = when (this) {
    "subscription" -> "Thuê bao"
    "perpetual" -> "Vĩnh viễn"
    else -> this
}
