package com.nartueih.licensemanager.data.auth

import kotlinx.serialization.Serializable

@Serializable
data class EmployeeUser(
    val id: String,
    val email: String,
    val fullName: String,
    val employeeCode: String,
    val departmentId: String?,
    val departmentName: String?,
)

@Serializable
data class EmployeeSession(
    val accessToken: String,
    val refreshToken: String,
    val expiresInSeconds: Int,
    val user: EmployeeUser,
)

sealed interface LoginOutcome {
    data class Success(val session: EmployeeSession) : LoginOutcome
    data object InvalidCredentials : LoginOutcome
    data object EmployeeOnly : LoginOutcome
    data object AccountLocked : LoginOutcome
    data object ConnectionError : LoginOutcome
    data object ServerError : LoginOutcome
}

sealed interface RefreshOutcome {
    data class Success(val session: EmployeeSession) : RefreshOutcome
    data object InvalidSession : RefreshOutcome
    data object ConnectionError : RefreshOutcome
    data object ServerError : RefreshOutcome
}

interface AuthRepository {
    suspend fun login(email: String, password: String): LoginOutcome

    suspend fun refresh(refreshToken: String): RefreshOutcome

    suspend fun logout(refreshToken: String)
}
