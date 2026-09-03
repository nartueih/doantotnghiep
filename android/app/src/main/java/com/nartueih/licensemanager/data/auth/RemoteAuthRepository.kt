package com.nartueih.licensemanager.data.auth

import java.io.IOException
import retrofit2.HttpException

internal class RemoteAuthRepository(
    private val api: AuthApi,
) : AuthRepository {
    override suspend fun login(email: String, password: String): LoginOutcome {
        val response = try {
            api.login(LoginRequestDto(email = email, password = password))
        } catch (error: HttpException) {
            return when (error.code()) {
                401 -> LoginOutcome.InvalidCredentials
                403 -> LoginOutcome.AccountLocked
                in 500..599 -> LoginOutcome.ServerError
                else -> throw error
            }
        } catch (_: IOException) {
            return LoginOutcome.ConnectionError
        }
        if (response.user.role != "employee") {
            revokeRefreshTokenBestEffort(response.tokens.refreshToken)
            return LoginOutcome.EmployeeOnly
        }
        return LoginOutcome.Success(session = response.toEmployeeSession())
    }

    override suspend fun refresh(refreshToken: String): RefreshOutcome {
        val response = try {
            api.refresh(LogoutRequestDto(refreshToken))
        } catch (error: HttpException) {
            return when (error.code()) {
                401, 403 -> RefreshOutcome.InvalidSession
                in 500..599 -> RefreshOutcome.ServerError
                else -> throw error
            }
        } catch (_: IOException) {
            return RefreshOutcome.ConnectionError
        }
        if (response.user.role != "employee" || response.user.status != "active") {
            revokeRefreshTokenBestEffort(response.tokens.refreshToken)
            return RefreshOutcome.InvalidSession
        }
        return RefreshOutcome.Success(response.toEmployeeSession())
    }

    override suspend fun logout(refreshToken: String) {
        revokeRefreshTokenBestEffort(refreshToken)
    }

    private suspend fun revokeRefreshTokenBestEffort(refreshToken: String) {
        try {
            api.logout(LogoutRequestDto(refreshToken))
        } catch (_: IOException) {
            // The app does not retain this token, so a network failure must not allow access.
        } catch (_: HttpException) {
            // Logout is best effort for accounts that mobile does not permit.
        }
    }
}

private fun LoginResponseDto.toEmployeeSession() = EmployeeSession(
    accessToken = tokens.accessToken,
    refreshToken = tokens.refreshToken,
    expiresInSeconds = tokens.expiresInSeconds,
    user = EmployeeUser(
        id = user.id,
        email = user.email,
        fullName = user.fullName,
        employeeCode = user.employeeCode,
        departmentId = user.departmentId,
        departmentName = user.departmentName,
    ),
)
