package com.nartueih.licensemanager.feature.home

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
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
import com.nartueih.licensemanager.data.auth.EmployeeSession
import com.nartueih.licensemanager.data.dashboard.EmployeeDashboardSummary

@Composable
fun EmployeeHomeScreen(
    session: EmployeeSession,
    viewModel: HomeViewModel,
    onLicenseCardClicked: () -> Unit,
    onDeviceCardClicked: () -> Unit,
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Surface(
        modifier = Modifier.fillMaxSize(),
        color = MaterialTheme.colorScheme.background,
    ) {
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 20.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            item { Spacer(modifier = Modifier.height(20.dp)) }
            item {
                HomeHeader(
                    fullName = session.user.fullName,
                    employeeCode = session.user.employeeCode,
                    departmentName = session.user.departmentName,
                )
            }
            item {
                when (val state = uiState) {
                    HomeUiState.Loading -> DashboardLoadingCard()
                    is HomeUiState.Content -> DashboardSummary(
                        summary = state.summary,
                        onLicenseCardClicked = onLicenseCardClicked,
                        onDeviceCardClicked = onDeviceCardClicked,
                    )
                    is HomeUiState.Error -> DashboardErrorCard(
                        message = state.message,
                        onRetryClicked = viewModel::retry,
                    )
                }
            }
            item { Spacer(modifier = Modifier.height(20.dp)) }
        }
    }
}

@Composable
private fun HomeHeader(
    fullName: String,
    employeeCode: String,
    departmentName: String?,
) {
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(
            text = "Tổng quan",
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
        )
        Text(
            text = "Xin chào, $fullName",
            style = MaterialTheme.typography.titleLarge,
            color = MaterialTheme.colorScheme.primary,
            fontWeight = FontWeight.SemiBold,
        )
        Text(
            text = "$employeeCode · ${departmentName ?: "Chưa phân phòng"}",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun DashboardSummary(
    summary: EmployeeDashboardSummary,
    onLicenseCardClicked: () -> Unit,
    onDeviceCardClicked: () -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(
            text = "Dữ liệu của tôi",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold,
        )
        StatisticCard(
            label = "License được cấp",
            value = summary.licenseCount,
            marker = "L",
            markerColor = MaterialTheme.colorScheme.primary,
            onClick = onLicenseCardClicked,
        )
        StatisticCard(
            label = "Thiết bị đang quản lý",
            value = summary.deviceCount,
            marker = "T",
            markerColor = Color(0xFF6D3FD1),
            onClick = onDeviceCardClicked,
        )
        StatisticCard(
            label = "Thông báo chưa đọc",
            value = summary.unreadNotificationCount,
            marker = "N",
            markerColor = Color(0xFFD06B12),
        )
    }
}

@Composable
private fun StatisticCard(
    label: String,
    value: Int,
    marker: String,
    markerColor: Color,
    onClick: (() -> Unit)? = null,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(enabled = onClick != null) { onClick?.invoke() },
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
    ) {
        Row(
            modifier = Modifier.padding(18.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Box(
                modifier = Modifier
                    .size(46.dp)
                    .background(markerColor.copy(alpha = 0.12f), CircleShape),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = marker,
                    color = markerColor,
                    fontWeight = FontWeight.Bold,
                )
            }
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = label,
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Text(
                    text = value.toString(),
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                )
            }
        }
    }
}

@Composable
private fun DashboardLoadingCard() {
    StatusCard {
        CircularProgressIndicator(modifier = Modifier.size(36.dp))
        Text("Đang tải dữ liệu tổng quan…")
    }
}

@Composable
private fun DashboardErrorCard(
    message: String,
    onRetryClicked: () -> Unit,
) {
    StatusCard {
        Text(
            text = message,
            color = MaterialTheme.colorScheme.error,
            textAlign = TextAlign.Center,
        )
        Button(onClick = onRetryClicked) {
            Text("Thử lại")
        }
    }
}

@Composable
private fun StatusCard(content: @Composable ColumnScope.() -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(28.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(14.dp),
            content = content,
        )
    }
}
