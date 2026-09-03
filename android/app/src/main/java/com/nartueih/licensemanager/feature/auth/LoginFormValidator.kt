package com.nartueih.licensemanager.feature.auth

data class LoginValidationResult(
    val normalizedEmail: String,
    val emailError: String? = null,
    val passwordError: String? = null,
) {
    val isValid: Boolean
        get() = emailError == null && passwordError == null
}

object LoginFormValidator {
    private val emailPattern = Regex("^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$")

    fun validate(email: String, password: String): LoginValidationResult {
        val normalizedEmail = email.trim()

        return LoginValidationResult(
            normalizedEmail = normalizedEmail,
            emailError = when {
                normalizedEmail.isEmpty() -> "Email không được bỏ trống."
                !emailPattern.matches(normalizedEmail) -> "Email không đúng định dạng."
                else -> null
            },
            passwordError = if (password.isBlank()) "Mật khẩu không được bỏ trống." else null,
        )
    }
}
