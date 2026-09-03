package com.nartueih.licensemanager.data.dashboard

data class EmployeeDashboardSummary(
    val licenseCount: Int,
    val deviceCount: Int,
    val unreadNotificationCount: Int,
)

sealed interface DashboardLoadOutcome {
    data class Success(val summary: EmployeeDashboardSummary) : DashboardLoadOutcome
    data object ConnectionError : DashboardLoadOutcome
    data object ServerError : DashboardLoadOutcome
}

interface EmployeeDashboardRepository {
    suspend fun load(): DashboardLoadOutcome
}
