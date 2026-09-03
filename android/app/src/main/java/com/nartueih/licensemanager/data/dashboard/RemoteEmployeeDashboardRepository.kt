package com.nartueih.licensemanager.data.dashboard

import java.io.IOException
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import okhttp3.OkHttpClient
import retrofit2.HttpException

internal class RemoteEmployeeDashboardRepository(
    private val api: EmployeeDashboardApi,
) : EmployeeDashboardRepository {
    override suspend fun load(): DashboardLoadOutcome = try {
        coroutineScope {
            val licenses = async { api.licenses() }
            val devices = async { api.devices() }
            val notifications = async { api.notifications() }

            DashboardLoadOutcome.Success(
                EmployeeDashboardSummary(
                    licenseCount = licenses.await().total,
                    deviceCount = devices.await().total,
                    unreadNotificationCount = notifications.await().unreadCount,
                ),
            )
        }
    } catch (_: IOException) {
        DashboardLoadOutcome.ConnectionError
    } catch (_: HttpException) {
        DashboardLoadOutcome.ServerError
    }
}

fun createEmployeeDashboardRepository(
    baseUrl: String,
    client: OkHttpClient,
): EmployeeDashboardRepository = RemoteEmployeeDashboardRepository(
    api = createEmployeeDashboardApi(baseUrl, client),
)
