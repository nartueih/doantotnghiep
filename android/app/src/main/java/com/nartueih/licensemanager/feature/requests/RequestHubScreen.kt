package com.nartueih.licensemanager.feature.requests

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Build
import androidx.compose.material.icons.rounded.VpnKey
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.nartueih.licensemanager.data.devices.EmployeeDevice

enum class RequestSection {
    LICENSE,
    MAINTENANCE,
}

@Composable
fun EmployeeRequestHubScreen(
    selectedSection: RequestSection,
    onSectionSelected: (RequestSection) -> Unit,
    licenseRequestViewModel: LicenseRequestViewModel,
    maintenanceRequestViewModel: MaintenanceRequestViewModel,
    devices: List<EmployeeDevice>,
) {
    Column(modifier = Modifier.fillMaxSize()) {
        Surface(
            modifier = Modifier.fillMaxWidth(),
            color = MaterialTheme.colorScheme.surface,
            shadowElevation = 1.dp,
        ) {
            Row(
                modifier = Modifier.padding(horizontal = 20.dp, vertical = 10.dp),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                FilterChip(
                    selected = selectedSection == RequestSection.LICENSE,
                    onClick = { onSectionSelected(RequestSection.LICENSE) },
                    label = { Text("Cấp license") },
                    leadingIcon = { Icon(Icons.Rounded.VpnKey, contentDescription = null) },
                )
                FilterChip(
                    selected = selectedSection == RequestSection.MAINTENANCE,
                    onClick = { onSectionSelected(RequestSection.MAINTENANCE) },
                    label = { Text("Bảo trì") },
                    leadingIcon = { Icon(Icons.Rounded.Build, contentDescription = null) },
                )
            }
        }
        Box(modifier = Modifier.weight(1f)) {
            when (selectedSection) {
                RequestSection.LICENSE -> EmployeeLicenseRequestScreen(licenseRequestViewModel)
                RequestSection.MAINTENANCE -> EmployeeMaintenanceRequestScreen(
                    viewModel = maintenanceRequestViewModel,
                    devices = devices,
                )
            }
        }
    }
}
