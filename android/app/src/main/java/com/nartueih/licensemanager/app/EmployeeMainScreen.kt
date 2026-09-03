package com.nartueih.licensemanager.app

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.Assignment
import androidx.compose.material.icons.rounded.Devices
import androidx.compose.material.icons.rounded.Home
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.VpnKey
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.repeatOnLifecycle
import com.nartueih.licensemanager.data.auth.EmployeeSession
import com.nartueih.licensemanager.feature.home.EmployeeHomeScreen
import com.nartueih.licensemanager.feature.home.HomeViewModel
import com.nartueih.licensemanager.feature.devices.DeviceViewModel
import com.nartueih.licensemanager.feature.devices.EmployeeDeviceScreen
import com.nartueih.licensemanager.feature.licenses.EmployeeLicenseScreen
import com.nartueih.licensemanager.feature.licenses.LicenseViewModel
import com.nartueih.licensemanager.feature.requests.EmployeeRequestHubScreen
import com.nartueih.licensemanager.feature.requests.LicenseRequestViewModel
import com.nartueih.licensemanager.feature.requests.MaintenanceRequestViewModel
import com.nartueih.licensemanager.feature.requests.RequestSection
import com.nartueih.licensemanager.feature.notifications.EmployeeNotificationTopBar
import com.nartueih.licensemanager.feature.notifications.NotificationViewModel
import com.nartueih.licensemanager.feature.profile.EmployeeProfileScreen
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive

private const val EMPLOYEE_SYNC_INTERVAL_MILLIS = 30_000L

private enum class EmployeeDestination(val label: String, val icon: ImageVector) {
    OVERVIEW("Tổng quan", Icons.Rounded.Home),
    LICENSES("License", Icons.Rounded.VpnKey),
    DEVICES("Thiết bị", Icons.Rounded.Devices),
    REQUESTS("Yêu cầu", Icons.AutoMirrored.Rounded.Assignment),
    PROFILE("Cá nhân", Icons.Rounded.Person),
}

@Composable
fun EmployeeMainScreen(
    session: EmployeeSession,
    homeViewModel: HomeViewModel,
    licenseViewModel: LicenseViewModel,
    deviceViewModel: DeviceViewModel,
    licenseRequestViewModel: LicenseRequestViewModel,
    maintenanceRequestViewModel: MaintenanceRequestViewModel,
    notificationViewModel: NotificationViewModel,
    onLogoutClicked: () -> Unit,
) {
    var selectedName by rememberSaveable { mutableStateOf(EmployeeDestination.OVERVIEW.name) }
    var selectedRequestSectionName by rememberSaveable { mutableStateOf(RequestSection.LICENSE.name) }
    val selected = EmployeeDestination.valueOf(selectedName)
    val selectedRequestSection = RequestSection.valueOf(selectedRequestSectionName)
    val deviceState by deviceViewModel.uiState.collectAsStateWithLifecycle()
    val lifecycleOwner = LocalLifecycleOwner.current
    val dataSynchronizer = remember(
        homeViewModel,
        licenseViewModel,
        deviceViewModel,
        licenseRequestViewModel,
        maintenanceRequestViewModel,
        notificationViewModel,
    ) {
        EmployeeDataSynchronizer(
            syncOverview = homeViewModel::sync,
            syncLicenses = licenseViewModel::sync,
            syncDevices = deviceViewModel::sync,
            syncLicenseRequests = licenseRequestViewModel::sync,
            syncMaintenanceRequests = maintenanceRequestViewModel::sync,
            syncNotifications = notificationViewModel::sync,
        )
    }

    LaunchedEffect(lifecycleOwner, selected, selectedRequestSection) {
        lifecycleOwner.lifecycle.repeatOnLifecycle(Lifecycle.State.RESUMED) {
            while (isActive) {
                dataSynchronizer.sync(
                    destination = EmployeeSyncDestination.valueOf(selected.name),
                    requestSection = selectedRequestSection,
                )
                delay(EMPLOYEE_SYNC_INTERVAL_MILLIS)
            }
        }
    }

    Scaffold(
        topBar = {
            EmployeeNotificationTopBar(
                viewModel = notificationViewModel,
                onNotificationSelected = { notification ->
                    selectedName = EmployeeDestination.REQUESTS.name
                    if (notification.entityType == "maintenance_request") {
                        selectedRequestSectionName = RequestSection.MAINTENANCE.name
                        maintenanceRequestViewModel.retry()
                    } else {
                        selectedRequestSectionName = RequestSection.LICENSE.name
                        licenseRequestViewModel.retry()
                        if (notification.type == "license_request_approved") licenseViewModel.retry()
                    }
                },
            )
        },
        bottomBar = {
            NavigationBar {
                EmployeeDestination.entries.forEach { destination ->
                    NavigationBarItem(
                        selected = selected == destination,
                        onClick = { selectedName = destination.name },
                        icon = {
                            Icon(
                                imageVector = destination.icon,
                                contentDescription = destination.label,
                            )
                        },
                        label = { Text(destination.label) },
                    )
                }
            }
        },
    ) { innerPadding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            when (selected) {
                EmployeeDestination.OVERVIEW -> EmployeeHomeScreen(
                    session = session,
                    viewModel = homeViewModel,
                    onLicenseCardClicked = { selectedName = EmployeeDestination.LICENSES.name },
                    onDeviceCardClicked = { selectedName = EmployeeDestination.DEVICES.name },
                )
                EmployeeDestination.LICENSES -> EmployeeLicenseScreen(viewModel = licenseViewModel)
                EmployeeDestination.DEVICES -> EmployeeDeviceScreen(
                    viewModel = deviceViewModel,
                    onMaintenanceRequested = { device ->
                        maintenanceRequestViewModel.openCreate(device.id)
                        selectedRequestSectionName = RequestSection.MAINTENANCE.name
                        selectedName = EmployeeDestination.REQUESTS.name
                    },
                )
                EmployeeDestination.REQUESTS -> EmployeeRequestHubScreen(
                    selectedSection = selectedRequestSection,
                    onSectionSelected = { selectedRequestSectionName = it.name },
                    licenseRequestViewModel = licenseRequestViewModel,
                    maintenanceRequestViewModel = maintenanceRequestViewModel,
                    devices = deviceState.items,
                )
                EmployeeDestination.PROFILE -> EmployeeProfileScreen(
                    session = session,
                    onLogoutClicked = onLogoutClicked,
                )
            }
        }
    }
}
