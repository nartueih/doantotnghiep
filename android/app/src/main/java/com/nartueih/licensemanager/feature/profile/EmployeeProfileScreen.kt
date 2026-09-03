package com.nartueih.licensemanager.feature.profile

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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.AdminPanelSettings
import androidx.compose.material.icons.rounded.Badge
import androidx.compose.material.icons.rounded.Business
import androidx.compose.material.icons.rounded.Email
import androidx.compose.material.icons.rounded.Info
import androidx.compose.material.icons.rounded.Logout
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.nartueih.licensemanager.data.auth.EmployeeSession
import com.nartueih.licensemanager.data.auth.EmployeeUser

data class EmployeeProfileUiModel(
    val fullName: String,
    val employeeCode: String,
    val email: String,
    val departmentName: String,
    val role: String,
) {
    companion object {
        fun from(user: EmployeeUser) = EmployeeProfileUiModel(
            fullName = user.fullName,
            employeeCode = user.employeeCode,
            email = user.email,
            departmentName = user.departmentName?.takeIf { it.isNotBlank() } ?: "Chưa phân phòng",
            role = "Nhân viên",
        )
    }
}

@Composable
fun EmployeeProfileScreen(
    session: EmployeeSession,
    onLogoutClicked: () -> Unit,
) {
    val profile = EmployeeProfileUiModel.from(session.user)
    var showLogoutConfirmation by rememberSaveable { mutableStateOf(false) }

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
            item { Spacer(modifier = Modifier.height(12.dp)) }
            item { ProfileHeader(profile) }
            item { ProfileInformationCard(profile) }
            item { AdministratorNotice() }
            item {
                OutlinedButton(
                    onClick = { showLogoutConfirmation = true },
                    modifier = Modifier.fillMaxWidth(),
                    colors = ButtonDefaults.outlinedButtonColors(
                        contentColor = MaterialTheme.colorScheme.error,
                    ),
                ) {
                    Icon(
                        imageVector = Icons.Rounded.Logout,
                        contentDescription = null,
                    )
                    Text(
                        text = "Đăng xuất",
                        modifier = Modifier.padding(start = 8.dp),
                        fontWeight = FontWeight.SemiBold,
                    )
                }
            }
            item { Spacer(modifier = Modifier.height(20.dp)) }
        }
    }

    if (showLogoutConfirmation) {
        AlertDialog(
            onDismissRequest = { showLogoutConfirmation = false },
            icon = {
                Icon(
                    imageVector = Icons.Rounded.Logout,
                    contentDescription = null,
                )
            },
            title = { Text("Đăng xuất khỏi ứng dụng?") },
            text = { Text("Bạn sẽ cần đăng nhập lại để tiếp tục sử dụng License Manager.") },
            confirmButton = {
                TextButton(onClick = onLogoutClicked) {
                    Text("Đăng xuất", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = {
                TextButton(onClick = { showLogoutConfirmation = false }) {
                    Text("Ở lại")
                }
            },
        )
    }
}

@Composable
private fun ProfileHeader(profile: EmployeeProfileUiModel) {
    Column(
        modifier = Modifier.fillMaxWidth(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Text(
            text = "Cá nhân",
            modifier = Modifier.fillMaxWidth(),
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
        )
        Spacer(modifier = Modifier.height(6.dp))
        Box(
            modifier = Modifier
                .size(88.dp)
                .background(MaterialTheme.colorScheme.primaryContainer, CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                imageVector = Icons.Rounded.Person,
                contentDescription = null,
                modifier = Modifier.size(48.dp),
                tint = MaterialTheme.colorScheme.primary,
            )
        }
        Text(
            text = profile.fullName,
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold,
            textAlign = TextAlign.Center,
        )
        Text(
            text = profile.employeeCode,
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun ProfileInformationCard(profile: EmployeeProfileUiModel) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
    ) {
        Column(modifier = Modifier.padding(horizontal = 18.dp, vertical = 8.dp)) {
            ProfileInformationRow(Icons.Rounded.Badge, "Mã nhân viên", profile.employeeCode)
            HorizontalDivider()
            ProfileInformationRow(Icons.Rounded.Email, "Email", profile.email)
            HorizontalDivider()
            ProfileInformationRow(Icons.Rounded.Business, "Phòng ban", profile.departmentName)
            HorizontalDivider()
            ProfileInformationRow(Icons.Rounded.AdminPanelSettings, "Vai trò", profile.role)
        }
    }
}

@Composable
private fun ProfileInformationRow(
    icon: ImageVector,
    label: String,
    value: String,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 14.dp),
        horizontalArrangement = Arrangement.spacedBy(14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(40.dp)
                .background(MaterialTheme.colorScheme.primaryContainer, CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                modifier = Modifier.size(21.dp),
                tint = MaterialTheme.colorScheme.primary,
            )
        }
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = label,
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                text = value,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.SemiBold,
            )
        }
    }
}

@Composable
private fun AdministratorNotice() {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.secondaryContainer,
        ),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.Top,
        ) {
            Icon(
                imageVector = Icons.Rounded.Info,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSecondaryContainer,
            )
            Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                Text(
                    text = "Thông tin chỉ đọc",
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSecondaryContainer,
                )
                Text(
                    text = "Thông tin hồ sơ do quản trị viên quản lý. Hãy liên hệ IT nếu bạn cần cập nhật.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSecondaryContainer,
                )
            }
        }
    }
}
