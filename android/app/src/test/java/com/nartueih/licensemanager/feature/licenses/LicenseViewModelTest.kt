package com.nartueih.licensemanager.feature.licenses

import com.nartueih.licensemanager.MainDispatcherRule
import com.nartueih.licensemanager.data.licenses.EmployeeLicense
import com.nartueih.licensemanager.data.licenses.EmployeeLicenseRepository
import com.nartueih.licensemanager.data.licenses.LicenseKeyOutcome
import com.nartueih.licensemanager.data.licenses.LicenseListOutcome
import com.nartueih.licensemanager.data.licenses.RevealedLicenseKey
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class LicenseViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    @Test
    fun initializationLoadsAssignedLicenses() = runTest {
        val item = employeeLicense()
        val viewModel = LicenseViewModel(
            FakeLicenseRepository(listOutcome = LicenseListOutcome.Success(listOf(item))),
        )

        advanceUntilIdle()

        assertFalse(viewModel.uiState.value.isLoading)
        assertEquals(listOf(item), viewModel.uiState.value.items)
        assertNull(viewModel.uiState.value.loadError)
    }

    @Test
    fun retryReplacesLoadErrorWithLicenses() = runTest {
        val item = employeeLicense()
        val repository = QueueLicenseRepository(
            LicenseListOutcome.ConnectionError,
            LicenseListOutcome.Success(listOf(item)),
        )
        val viewModel = LicenseViewModel(repository)
        advanceUntilIdle()

        viewModel.retry()
        advanceUntilIdle()

        assertEquals(2, repository.listCount)
        assertEquals(listOf(item), viewModel.uiState.value.items)
        assertNull(viewModel.uiState.value.loadError)
    }

    @Test
    fun revealKeyExposesReturnedKeyAndClearsProgress() = runTest {
        val revealed = RevealedLicenseKey(
            assignmentId = "assignment-1",
            licenseName = "Adobe Creative Cloud",
            licenseKey = "AAAA-BBBB-CCCC",
        )
        val viewModel = LicenseViewModel(
            FakeLicenseRepository(
                listOutcome = LicenseListOutcome.Success(listOf(employeeLicense())),
                keyOutcome = LicenseKeyOutcome.Success(revealed),
            ),
        )
        advanceUntilIdle()

        viewModel.revealKey("assignment-1")
        advanceUntilIdle()

        assertEquals(revealed, viewModel.uiState.value.revealedKey)
        assertNull(viewModel.uiState.value.revealingAssignmentId)
        assertNull(viewModel.uiState.value.keyError)
    }

    @Test
    fun unavailableKeyShowsVietnameseError() = runTest {
        val viewModel = LicenseViewModel(
            FakeLicenseRepository(
                listOutcome = LicenseListOutcome.Success(listOf(employeeLicense())),
                keyOutcome = LicenseKeyOutcome.Unavailable,
            ),
        )
        advanceUntilIdle()

        viewModel.revealKey("assignment-1")
        advanceUntilIdle()

        assertEquals(
            "Key license hiện không khả dụng. Vui lòng liên hệ IT.",
            viewModel.uiState.value.keyError,
        )
        assertNull(viewModel.uiState.value.revealedKey)
    }

    private fun employeeLicense() = EmployeeLicense(
        assignmentId = "assignment-1",
        licenseId = "license-1",
        licenseName = "Adobe Creative Cloud",
        licenseType = "subscription",
        assignmentSource = "device",
        deviceAssetCode = "LAP-001",
        assignedAt = "2026-08-20T08:30:00Z",
        expiresAt = "2026-12-31",
        lifecycleStatus = "active",
        notes = "Dùng cho thiết kế",
        canViewKey = true,
    )

    private class FakeLicenseRepository(
        private val listOutcome: LicenseListOutcome,
        private val keyOutcome: LicenseKeyOutcome = LicenseKeyOutcome.Unavailable,
    ) : EmployeeLicenseRepository {
        override suspend fun list() = listOutcome
        override suspend fun revealKey(assignmentId: String) = keyOutcome
    }

    private class QueueLicenseRepository(
        vararg outcomes: LicenseListOutcome,
    ) : EmployeeLicenseRepository {
        private val outcomes = ArrayDeque(outcomes.toList())
        var listCount = 0

        override suspend fun list(): LicenseListOutcome {
            listCount++
            return outcomes.removeFirst()
        }

        override suspend fun revealKey(assignmentId: String) = LicenseKeyOutcome.Unavailable
    }
}
