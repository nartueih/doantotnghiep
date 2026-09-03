package com.nartueih.licensemanager.app

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.nartueih.licensemanager.feature.auth.LoginRoute
import com.nartueih.licensemanager.feature.auth.LoginViewModel
import com.nartueih.licensemanager.feature.devices.DeviceViewModel
import com.nartueih.licensemanager.feature.home.HomeViewModel
import com.nartueih.licensemanager.feature.licenses.LicenseViewModel
import com.nartueih.licensemanager.feature.requests.MaintenanceRequestViewModel
import com.nartueih.licensemanager.feature.requests.LicenseRequestViewModel
import com.nartueih.licensemanager.feature.notifications.NotificationViewModel

@Composable
fun LicenseManagerApp(
    appViewModel: AppViewModel,
    loginViewModel: LoginViewModel,
    homeViewModelFactory: HomeViewModel.Factory,
    licenseViewModelFactory: LicenseViewModel.Factory,
    deviceViewModelFactory: DeviceViewModel.Factory,
    licenseRequestViewModelFactory: LicenseRequestViewModel.Factory,
    maintenanceRequestViewModelFactory: MaintenanceRequestViewModel.Factory,
    notificationViewModelFactory: NotificationViewModel.Factory,
) {
    val appState by appViewModel.uiState.collectAsStateWithLifecycle()

    when (val state = appState) {
        AppUiState.Loading -> AppLoadingScreen()
        AppUiState.SignedOut -> {
            LaunchedEffect(Unit) {
                loginViewModel.onSignedOut()
            }
            LoginRoute(viewModel = loginViewModel)
        }
        is AppUiState.SignedIn -> {
            val homeViewModel: HomeViewModel = viewModel(
                key = "home-${state.session.accessToken}",
                factory = homeViewModelFactory,
            )
            val licenseViewModel: LicenseViewModel = viewModel(
                key = "licenses-${state.session.accessToken}",
                factory = licenseViewModelFactory,
            )
            val deviceViewModel: DeviceViewModel = viewModel(
                key = "devices-${state.session.accessToken}",
                factory = deviceViewModelFactory,
            )
            val maintenanceRequestViewModel: MaintenanceRequestViewModel = viewModel(
                key = "maintenance-requests-${state.session.accessToken}",
                factory = maintenanceRequestViewModelFactory,
            )
            val licenseRequestViewModel: LicenseRequestViewModel = viewModel(
                key = "license-requests-${state.session.accessToken}",
                factory = licenseRequestViewModelFactory,
            )
            val notificationViewModel: NotificationViewModel = viewModel(
                key = "notifications-${state.session.accessToken}",
                factory = notificationViewModelFactory,
            )
            EmployeeMainScreen(
                session = state.session,
                homeViewModel = homeViewModel,
                licenseViewModel = licenseViewModel,
                deviceViewModel = deviceViewModel,
                licenseRequestViewModel = licenseRequestViewModel,
                maintenanceRequestViewModel = maintenanceRequestViewModel,
                notificationViewModel = notificationViewModel,
                onLogoutClicked = appViewModel::onLogoutClicked,
            )
        }
    }
}

@Composable
private fun AppLoadingScreen() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        CircularProgressIndicator()
    }
}
