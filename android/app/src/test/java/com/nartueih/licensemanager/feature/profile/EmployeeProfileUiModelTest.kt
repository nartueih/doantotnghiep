package com.nartueih.licensemanager.feature.profile

import com.nartueih.licensemanager.data.auth.EmployeeUser
import org.junit.Assert.assertEquals
import org.junit.Test

class EmployeeProfileUiModelTest {
    @Test
    fun fromMapsTheAuthenticatedEmployeeInformation() {
        val user = EmployeeUser(
            id = "user-1",
            email = "anh.nguyen@local.test",
            fullName = "Nguyễn Văn Anh",
            employeeCode = "EMP-001",
            departmentId = "department-1",
            departmentName = "Công nghệ thông tin",
        )

        val model = EmployeeProfileUiModel.from(user)

        assertEquals("Nguyễn Văn Anh", model.fullName)
        assertEquals("EMP-001", model.employeeCode)
        assertEquals("anh.nguyen@local.test", model.email)
        assertEquals("Công nghệ thông tin", model.departmentName)
        assertEquals("Nhân viên", model.role)
    }

    @Test
    fun fromUsesUnassignedLabelWhenEmployeeHasNoDepartment() {
        val user = EmployeeUser(
            id = "user-2",
            email = "employee@local.test",
            fullName = "Nhân viên Demo",
            employeeCode = "EMP-002",
            departmentId = null,
            departmentName = " ",
        )

        val model = EmployeeProfileUiModel.from(user)

        assertEquals("Chưa phân phòng", model.departmentName)
    }
}
