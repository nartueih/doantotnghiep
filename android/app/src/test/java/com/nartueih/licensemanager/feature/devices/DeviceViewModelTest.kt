package com.nartueih.licensemanager.feature.devices

import com.nartueih.licensemanager.MainDispatcherRule
import com.nartueih.licensemanager.data.devices.DeviceListOutcome
import com.nartueih.licensemanager.data.devices.EmployeeDevice
import com.nartueih.licensemanager.data.devices.EmployeeDeviceRepository
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class DeviceViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    @Test
    fun initializationLoadsAssignedDevices() = runTest {
        val device = employeeDevice()
        val viewModel = DeviceViewModel(
            QueueDeviceRepository(DeviceListOutcome.Success(listOf(device))),
        )

        advanceUntilIdle()

        assertFalse(viewModel.uiState.value.isLoading)
        assertEquals(listOf(device), viewModel.uiState.value.items)
        assertNull(viewModel.uiState.value.error)
    }

    @Test
    fun retryReplacesConnectionErrorWithDevices() = runTest {
        val device = employeeDevice()
        val repository = QueueDeviceRepository(
            DeviceListOutcome.ConnectionError,
            DeviceListOutcome.Success(listOf(device)),
        )
        val viewModel = DeviceViewModel(repository)
        advanceUntilIdle()

        viewModel.retry()
        advanceUntilIdle()

        assertEquals(2, repository.listCount)
        assertEquals(listOf(device), viewModel.uiState.value.items)
        assertNull(viewModel.uiState.value.error)
    }

    @Test
    fun syncKeepsCurrentDevicesVisibleWhileRequestIsRunning() = runTest {
        val device = employeeDevice()
        val gate = CompletableDeferred<DeviceListOutcome>()
        val repository = object : EmployeeDeviceRepository {
            var calls = 0
            override suspend fun list(): DeviceListOutcome {
                calls++
                return if (calls == 1) DeviceListOutcome.Success(listOf(device)) else gate.await()
            }
        }
        val viewModel = DeviceViewModel(repository)
        advanceUntilIdle()

        viewModel.sync()

        assertFalse(viewModel.uiState.value.isLoading)
        assertEquals(listOf(device), viewModel.uiState.value.items)
        gate.complete(DeviceListOutcome.Success(listOf(device)))
        advanceUntilIdle()
    }

    private fun employeeDevice() = EmployeeDevice(
        id = "device-1",
        assetCode = "DEMO-LT-001",
        serialNumber = "DEMO-SEED-DELL-001",
        name = "Laptop Dell Latitude",
        deviceType = "laptop",
        manufacturer = "Dell",
        model = "Latitude 7450",
        status = "assigned",
        purchasedAt = "2026-01-10",
        warrantyExpiresAt = "2028-01-10",
    )

    private class QueueDeviceRepository(
        vararg outcomes: DeviceListOutcome,
    ) : EmployeeDeviceRepository {
        private val outcomes = ArrayDeque(outcomes.toList())
        var listCount = 0

        override suspend fun list(): DeviceListOutcome {
            listCount++
            return outcomes.removeFirst()
        }
    }
}
