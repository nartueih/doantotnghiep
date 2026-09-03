package com.nartueih.licensemanager.feature.auth

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class LoginFormValidatorTest {
    @Test
    fun blankCredentialsReturnRequiredFieldErrors() {
        val result = LoginFormValidator.validate(email = "   ", password = "")

        assertFalse(result.isValid)
        assertEquals("Email không được bỏ trống.", result.emailError)
        assertEquals("Mật khẩu không được bỏ trống.", result.passwordError)
    }

    @Test
    fun malformedEmailReturnsFormatError() {
        val result = LoginFormValidator.validate(
            email = "khong-phai-email",
            password = "ChangeMe123!",
        )

        assertFalse(result.isValid)
        assertEquals("Email không đúng định dạng.", result.emailError)
        assertNull(result.passwordError)
    }

    @Test
    fun validCredentialsReturnTrimmedEmail() {
        val result = LoginFormValidator.validate(
            email = "  employee@local.test  ",
            password = "ChangeMe123!",
        )

        assertTrue(result.isValid)
        assertEquals("employee@local.test", result.normalizedEmail)
        assertNull(result.emailError)
        assertNull(result.passwordError)
    }
}
