package com.nartueih.licensemanager.feature.auth

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import com.nartueih.licensemanager.data.auth.EmployeeUser
import com.nartueih.licensemanager.ui.theme.LicenseManagerTheme
import org.junit.Rule
import org.junit.Test

class LoginScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun blankLoginShowsValidationMessages() {
        composeRule.setContent {
            LicenseManagerTheme(dynamicColor = false) {
                LoginRoute()
            }
        }

        composeRule.onNodeWithText("Đăng nhập").performClick()

        composeRule.onNodeWithText("Email không được bỏ trống.").assertIsDisplayed()
        composeRule.onNodeWithText("Mật khẩu không được bỏ trống.").assertIsDisplayed()
    }

    @Test
    fun passwordVisibilityButtonChangesItsLabelWhenToggled() {
        composeRule.setContent {
            LicenseManagerTheme(dynamicColor = false) {
                LoginRoute()
            }
        }

        composeRule.onNodeWithText("Hiện").performClick()

        composeRule.onNodeWithText("Ẩn").assertIsDisplayed()
    }

    @Test
    fun generalLoginErrorIsDisplayed() {
        composeRule.setContent {
            LicenseManagerTheme(dynamicColor = false) {
                LoginScreen(
                    uiState = LoginUiState(
                        generalError = "Không thể kết nối tới máy chủ. Vui lòng thử lại.",
                    ),
                    onEmailChanged = {},
                    onPasswordChanged = {},
                    onLoginClicked = {},
                )
            }
        }

        composeRule
            .onNodeWithText("Không thể kết nối tới máy chủ. Vui lòng thử lại.")
            .assertIsDisplayed()
    }

    @Test
    fun submittingLoginShowsProgressLabel() {
        composeRule.setContent {
            LicenseManagerTheme(dynamicColor = false) {
                LoginScreen(
                    uiState = LoginUiState(isSubmitting = true),
                    onEmailChanged = {},
                    onPasswordChanged = {},
                    onLoginClicked = {},
                )
            }
        }

        composeRule.onNodeWithText("Đang đăng nhập...").assertIsDisplayed()
    }

    @Test
    fun authenticatedEmployeeSeesSuccessGreeting() {
        composeRule.setContent {
            LicenseManagerTheme(dynamicColor = false) {
                LoginScreen(
                    uiState = LoginUiState(
                        authenticatedUser = EmployeeUser(
                            id = "user-1",
                            email = "employee@local.test",
                            fullName = "Nguyễn Hoàng Anh",
                            employeeCode = "DEMO-002",
                            departmentId = "department-1",
                            departmentName = "Công nghệ thông tin",
                        ),
                    ),
                    onEmailChanged = {},
                    onPasswordChanged = {},
                    onLoginClicked = {},
                )
            }
        }

        composeRule
            .onNodeWithText("Đăng nhập thành công. Xin chào Nguyễn Hoàng Anh!")
            .assertIsDisplayed()
    }
}
