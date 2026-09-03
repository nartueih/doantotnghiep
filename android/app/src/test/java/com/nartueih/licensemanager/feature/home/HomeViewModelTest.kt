package com.nartueih.licensemanager.feature.home

import com.nartueih.licensemanager.MainDispatcherRule
import com.nartueih.licensemanager.data.dashboard.DashboardLoadOutcome
import com.nartueih.licensemanager.data.dashboard.EmployeeDashboardRepository
import com.nartueih.licensemanager.data.dashboard.EmployeeDashboardSummary
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class HomeViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    @Test
    fun initializationLoadsDashboardSummary() = runTest {
        val summary = EmployeeDashboardSummary(licenseCount = 4, deviceCount = 2, unreadNotificationCount = 3)
        val viewModel = HomeViewModel(QueueDashboardRepository(DashboardLoadOutcome.Success(summary)))

        advanceUntilIdle()

        assertEquals(HomeUiState.Content(summary), viewModel.uiState.value)
    }

    @Test
    fun connectionFailureShowsRetryableMessage() = runTest {
        val viewModel = HomeViewModel(QueueDashboardRepository(DashboardLoadOutcome.ConnectionError))

        advanceUntilIdle()

        assertEquals(
            HomeUiState.Error("Không thể kết nối tới máy chủ. Vui lòng thử lại."),
            viewModel.uiState.value,
        )
    }

    @Test
    fun retryLoadsDashboardAgain() = runTest {
        val summary = EmployeeDashboardSummary(licenseCount = 5, deviceCount = 1, unreadNotificationCount = 0)
        val repository = QueueDashboardRepository(
            DashboardLoadOutcome.ServerError,
            DashboardLoadOutcome.Success(summary),
        )
        val viewModel = HomeViewModel(repository)
        advanceUntilIdle()

        viewModel.retry()
        advanceUntilIdle()

        assertEquals(2, repository.loadCount)
        assertEquals(HomeUiState.Content(summary), viewModel.uiState.value)
    }

    private class QueueDashboardRepository(
        vararg outcomes: DashboardLoadOutcome,
    ) : EmployeeDashboardRepository {
        private val outcomes = ArrayDeque(outcomes.toList())
        var loadCount = 0

        override suspend fun load(): DashboardLoadOutcome {
            loadCount++
            return outcomes.removeFirst()
        }
    }
}
