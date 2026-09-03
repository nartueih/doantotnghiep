package com.nartueih.licensemanager.feature.requests

import com.nartueih.licensemanager.MainDispatcherRule
import com.nartueih.licensemanager.data.licenserequests.CreateLicenseRequestInput
import com.nartueih.licensemanager.data.licenserequests.LicenseRequest
import com.nartueih.licensemanager.data.licenserequests.LicenseRequestLoadOutcome
import com.nartueih.licensemanager.data.licenserequests.LicenseRequestMutationOutcome
import com.nartueih.licensemanager.data.licenserequests.LicenseRequestRepository
import com.nartueih.licensemanager.data.licenserequests.RequestableSoftware
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class LicenseRequestViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    @Test
    fun initializationLoadsRequestsAndSoftware() = runTest {
        val request = request()
        val software = software()
        val viewModel = LicenseRequestViewModel(
            FakeRepository(loadOutcome = LicenseRequestLoadOutcome.Success(listOf(request), listOf(software))),
        )

        advanceUntilIdle()

        assertFalse(viewModel.uiState.value.isLoading)
        assertEquals(listOf(request), viewModel.uiState.value.items)
        assertEquals(listOf(software), viewModel.uiState.value.software)
        assertEquals(1, viewModel.uiState.value.pendingCount)
    }

    @Test
    fun successfulSubmitTrimsReasonAndAddsPendingRequest() = runTest {
        val created = request()
        val repository = FakeRepository(createOutcome = LicenseRequestMutationOutcome.Success(created))
        val viewModel = LicenseRequestViewModel(repository)
        advanceUntilIdle()
        viewModel.openCreate("product-1")
        viewModel.updatePriority("high")
        viewModel.updateReason("  Cần dùng cho công việc thiết kế.  ")

        viewModel.submit()
        advanceUntilIdle()

        assertEquals(
            CreateLicenseRequestInput("product-1", "high", "Cần dùng cho công việc thiết kế."),
            repository.lastCreateInput,
        )
        assertFalse(viewModel.uiState.value.isCreateOpen)
        assertEquals(created, viewModel.uiState.value.items.first())
        assertEquals("Đã gửi yêu cầu cấp license.", viewModel.uiState.value.message)
    }

    @Test
    fun blankSoftwareOrReasonIsRejectedLocally() = runTest {
        val repository = FakeRepository()
        val viewModel = LicenseRequestViewModel(repository)
        advanceUntilIdle()
        viewModel.openCreate(null)

        viewModel.submit()
        advanceUntilIdle()

        assertNull(repository.lastCreateInput)
        assertEquals("Vui lòng chọn phần mềm và nhập lý do sử dụng.", viewModel.uiState.value.formError)
    }

    @Test
    fun pendingDuplicateKeepsFormOpenAndShowsFriendlyError() = runTest {
        val repository = FakeRepository(createOutcome = LicenseRequestMutationOutcome.PendingRequestExists)
        val viewModel = LicenseRequestViewModel(repository)
        advanceUntilIdle()
        viewModel.openCreate("product-1")
        viewModel.updateReason("Cần sử dụng")

        viewModel.submit()
        advanceUntilIdle()

        assertTrue(viewModel.uiState.value.isCreateOpen)
        assertEquals("Bạn đã có một yêu cầu đang chờ cho phần mềm này.", viewModel.uiState.value.formError)
    }

    @Test
    fun cancellingPendingRequestUpdatesListAndCount() = runTest {
        val pending = request()
        val cancelled = pending.copy(status = "cancelled", cancelledAt = "2026-09-01T10:00:00Z")
        val repository = FakeRepository(
            loadOutcome = LicenseRequestLoadOutcome.Success(listOf(pending), listOf(software())),
            cancelOutcome = LicenseRequestMutationOutcome.Success(cancelled),
        )
        val viewModel = LicenseRequestViewModel(repository)
        advanceUntilIdle()

        viewModel.cancel("request-1")
        advanceUntilIdle()

        assertEquals("request-1", repository.lastCancelledId)
        assertEquals("cancelled", viewModel.uiState.value.items.single().status)
        assertEquals(0, viewModel.uiState.value.pendingCount)
        assertEquals("Đã hủy yêu cầu cấp license.", viewModel.uiState.value.message)
    }

    private fun software() = RequestableSoftware("product-1", "Adobe Creative Cloud", "Adobe", "2026", "Thiết kế")

    private fun request() = LicenseRequest(
        id = "request-1", softwareProductId = "product-1", softwareProductName = "Adobe Creative Cloud",
        priority = "urgent", reason = "Cần phục vụ thiết kế.", status = "pending",
        selectedLicenseName = null, assignmentId = null, reviewedByName = null,
        decisionReason = null, responseNote = null, createdAt = "2026-09-01T08:00:00Z",
        updatedAt = "2026-09-01T08:00:00Z", reviewedAt = null, cancelledAt = null,
    )

    private class FakeRepository(
        private val loadOutcome: LicenseRequestLoadOutcome = LicenseRequestLoadOutcome.Success(
            emptyList(),
            listOf(RequestableSoftware("product-1", "Adobe Creative Cloud", "Adobe", "2026", "Thiết kế")),
        ),
        private val createOutcome: LicenseRequestMutationOutcome = LicenseRequestMutationOutcome.ServerError,
        private val cancelOutcome: LicenseRequestMutationOutcome = LicenseRequestMutationOutcome.ServerError,
    ) : LicenseRequestRepository {
        var lastCreateInput: CreateLicenseRequestInput? = null
        var lastCancelledId: String? = null

        override suspend fun load() = loadOutcome
        override suspend fun create(input: CreateLicenseRequestInput): LicenseRequestMutationOutcome {
            lastCreateInput = input
            return createOutcome
        }
        override suspend fun cancel(requestId: String): LicenseRequestMutationOutcome {
            lastCancelledId = requestId
            return cancelOutcome
        }
    }
}
