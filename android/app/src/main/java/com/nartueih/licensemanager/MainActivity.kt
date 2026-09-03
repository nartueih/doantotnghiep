package com.nartueih.licensemanager

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.lifecycle.viewmodel.compose.viewModel
import com.nartueih.licensemanager.app.AppViewModel
import com.nartueih.licensemanager.app.LicenseManagerApp
import com.nartueih.licensemanager.core.network.createAuthenticatedHttpClient
import com.nartueih.licensemanager.core.session.SessionRefresher
import com.nartueih.licensemanager.core.session.createSessionStore
import com.nartueih.licensemanager.data.auth.createAuthRepository
import com.nartueih.licensemanager.data.dashboard.createEmployeeDashboardRepository
import com.nartueih.licensemanager.data.devices.createEmployeeDeviceRepository
import com.nartueih.licensemanager.data.licenses.createEmployeeLicenseRepository
import com.nartueih.licensemanager.data.maintenance.createMaintenanceRequestRepository
import com.nartueih.licensemanager.data.licenserequests.createLicenseRequestRepository
import com.nartueih.licensemanager.data.notifications.createNotificationRepository
import com.nartueih.licensemanager.feature.auth.LoginViewModel
import com.nartueih.licensemanager.feature.devices.DeviceViewModel
import com.nartueih.licensemanager.feature.home.HomeViewModel
import com.nartueih.licensemanager.feature.licenses.LicenseViewModel
import com.nartueih.licensemanager.feature.requests.MaintenanceRequestViewModel
import com.nartueih.licensemanager.feature.requests.LicenseRequestViewModel
import com.nartueih.licensemanager.feature.notifications.NotificationViewModel
import com.nartueih.licensemanager.ui.theme.LicenseManagerTheme

class MainActivity : ComponentActivity() {
    private val authRepository by lazy {
        createAuthRepository(BuildConfig.API_BASE_URL)
    }
    private val sessionStore by lazy {
        createSessionStore(applicationContext)
    }
    private val sessionRefresher by lazy {
        SessionRefresher(sessionStore, authRepository)
    }
    private val authenticatedHttpClient by lazy {
        createAuthenticatedHttpClient(sessionStore, sessionRefresher)
    }
    private val dashboardRepository by lazy {
        createEmployeeDashboardRepository(BuildConfig.API_BASE_URL, authenticatedHttpClient)
    }
    private val licenseRepository by lazy {
        createEmployeeLicenseRepository(BuildConfig.API_BASE_URL, authenticatedHttpClient)
    }
    private val deviceRepository by lazy {
        createEmployeeDeviceRepository(BuildConfig.API_BASE_URL, authenticatedHttpClient)
    }
    private val maintenanceRequestRepository by lazy {
        createMaintenanceRequestRepository(BuildConfig.API_BASE_URL, authenticatedHttpClient)
    }
    private val licenseRequestRepository by lazy {
        createLicenseRequestRepository(BuildConfig.API_BASE_URL, authenticatedHttpClient)
    }
    private val notificationRepository by lazy {
        createNotificationRepository(BuildConfig.API_BASE_URL, authenticatedHttpClient)
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            LicenseManagerTheme(dynamicColor = false) {
                val appViewModel: AppViewModel = viewModel(
                    factory = AppViewModel.Factory(sessionStore, authRepository),
                )
                val loginViewModel: LoginViewModel = viewModel(
                    factory = LoginViewModel.Factory(authRepository, sessionStore),
                )
                LicenseManagerApp(
                    appViewModel = appViewModel,
                    loginViewModel = loginViewModel,
                    homeViewModelFactory = HomeViewModel.Factory(dashboardRepository),
                    licenseViewModelFactory = LicenseViewModel.Factory(licenseRepository),
                    deviceViewModelFactory = DeviceViewModel.Factory(deviceRepository),
                    licenseRequestViewModelFactory = LicenseRequestViewModel.Factory(licenseRequestRepository),
                    maintenanceRequestViewModelFactory = MaintenanceRequestViewModel.Factory(
                        maintenanceRequestRepository,
                    ),
                    notificationViewModelFactory = NotificationViewModel.Factory(notificationRepository),
                )
            }
        }
    }
}
