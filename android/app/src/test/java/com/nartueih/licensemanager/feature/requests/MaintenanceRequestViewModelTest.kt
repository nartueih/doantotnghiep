package com.nartueih.licensemanager.feature.requests

import com.nartueih.licensemanager.MainDispatcherRule
import com.nartueih.licensemanager.data.maintenance.CreateMaintenanceRequestInput
import com.nartueih.licensemanager.data.maintenance.MaintenanceListOutcome
import com.nartueih.licensemanager.data.maintenance.MaintenanceMutationOutcome
import com.nartueih.licensemanager.data.maintenance.MaintenanceRequest
import com.nartueih.licensemanager.data.maintenance.MaintenanceRequestRepository
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
class MaintenanceRequestViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    @Test
    fun initializationLoadsPersonalRequests() = runTest {
        val item = request()
        val viewModel = MaintenanceRequestViewModel(
            FakeRepository(listOutcome = MaintenanceListOutcome.Success(listOf(item), openCount = 1)),
        )

        advanceUntilIdle()

        assertFalse(viewModel.uiState.value.isLoading)
        assertEquals(listOf(item), viewModel.uiState.value.items)
        assertEquals(1, viewModel.uiState.value.openCount)
    }

    @Test
    fun openCreatePrefillsDeviceAndSuccessfulSubmitAddsTrimmedRequest() = runTest {
        val created = request(status = "pending")
        val repository = FakeRepository(createOutcome = MaintenanceMutationOutcome.Success(created))
        val viewModel = MaintenanceRequestViewModel(repository)
        advanceUntilIdle()

        viewModel.openCreate("device-1")
        viewModel.updateCategory("software")
        viewModel.updatePriority("high")
        viewModel.updateTitle("  Không mở được Office  ")
        viewModel.updateDescription("  Ứng dụng báo lỗi khi khởi động.  ")
        viewModel.submit()
        advanceUntilIdle()

        assertEquals(
            CreateMaintenanceRequestInput(
                "device-1", "software", "high", "Không mở được Office", "Ứng dụng báo lỗi khi khởi động.",
            ),
            repository.lastCreateInput,
        )
        assertFalse(viewModel.uiState.value.isCreateOpen)
        assertEquals(created, viewModel.uiState.value.items.first())
        assertEquals("Đã gửi yêu cầu bảo trì.", viewModel.uiState.value.message)
    }

    @Test
    fun blankRequiredFieldsAreRejectedLocally() = runTest {
        val repository = FakeRepository()
        val viewModel = MaintenanceRequestViewModel(repository)
        advanceUntilIdle()

        viewModel.openCreate(null)
        viewModel.submit()
        advanceUntilIdle()

        assertNull(repository.lastCreateInput)
        assertEquals("Vui lòng chọn thiết bị và nhập đầy đủ tiêu đề, mô tả.", viewModel.uiState.value.formError)
    }

    @Test
    fun duplicateRequestKeepsFormOpenWithVietnameseMessage() = runTest {
        val repository = FakeRepository(createOutcome = MaintenanceMutationOutcome.OpenRequestExists)
        val viewModel = MaintenanceRequestViewModel(repository)
        advanceUntilIdle()
        viewModel.openCreate("device-1")
        viewModel.updateTitle("Lỗi máy")
        viewModel.updateDescription("Không thể tiếp tục làm việc")

        viewModel.submit()
        advanceUntilIdle()

        assertTrue(viewModel.uiState.value.isCreateOpen)
        assertEquals("Thiết bị này đã có một yêu cầu bảo trì đang mở.", viewModel.uiState.value.formError)
    }

    @Test
    fun cancelPendingRequestReplacesItWithCancelledVersion() = runTest {
        val pending = request(status = "pending")
        val cancelled = pending.copy(status = "cancelled", cancelledAt = "2026-09-01T10:00:00Z")
        val repository = FakeRepository(
            listOutcome = MaintenanceListOutcome.Success(listOf(pending), openCount = 1),
            cancelOutcome = MaintenanceMutationOutcome.Success(cancelled),
        )
        val viewModel = MaintenanceRequestViewModel(repository)
        advanceUntilIdle()

        viewModel.cancel("request-1")
        advanceUntilIdle()

        assertEquals("request-1", repository.lastCancelledId)
        assertEquals("cancelled", viewModel.uiState.value.items.single().status)
        assertEquals(0, viewModel.uiState.value.openCount)
        assertEquals("Đã hủy yêu cầu bảo trì.", viewModel.uiState.value.message)
    }

    private fun request(status: String = "in_progress") = MaintenanceRequest(
        id = "request-1", requesterName = "Nguyễn Hoàng Anh", deviceId = "device-1",
        deviceAssetCode = "DEMO-LT-001", deviceSerialNumber = "DEMO-SEED-DELL-001",
        deviceName = "Laptop Dell Latitude", deviceType = "laptop", deviceManufacturer = "Dell",
        deviceModel = "Latitude 7450", devicePurchasedAt = "2026-01-10",
        deviceWarrantyExpiresAt = "2028-01-10", category = "hardware", priority = "urgent",
        title = "Không nhận bàn phím", description = "Một số phím không hoạt động.", status = status,
        assignedToName = "Development Admin", responseNote = "Đang kiểm tra",
        createdAt = "2026-09-01T08:00:00Z", updatedAt = "2026-09-01T09:00:00Z",
        acceptedAt = "2026-09-01T09:00:00Z", completedAt = null, rejectedAt = null, cancelledAt = null,
    )

    private class FakeRepository(
        private val listOutcome: MaintenanceListOutcome = MaintenanceListOutcome.Success(emptyList(), 0),
        private val createOutcome: MaintenanceMutationOutcome = MaintenanceMutationOutcome.ServerError,
        private val cancelOutcome: MaintenanceMutationOutcome = MaintenanceMutationOutcome.ServerError,
    ) : MaintenanceRequestRepository {
        var lastCreateInput: CreateMaintenanceRequestInput? = null
        var lastCancelledId: String? = null

        override suspend fun listMine() = listOutcome
        override suspend fun create(input: CreateMaintenanceRequestInput): MaintenanceMutationOutcome {
            lastCreateInput = input
            return createOutcome
        }
        override suspend fun cancel(requestId: String): MaintenanceMutationOutcome {
            lastCancelledId = requestId
            return cancelOutcome
        }
    }
}
