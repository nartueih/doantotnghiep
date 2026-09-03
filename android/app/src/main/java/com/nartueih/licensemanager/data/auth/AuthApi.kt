package com.nartueih.licensemanager.data.auth

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import retrofit2.http.Body
import retrofit2.http.POST

@Serializable
internal data class LoginRequestDto(
    val email: String,
    val password: String,
)

@Serializable
internal data class LogoutRequestDto(
    @SerialName("refresh_token") val refreshToken: String,
)

@Serializable
internal data class LoginResponseDto(
    val tokens: TokenPairDto,
    val user: UserDto,
)

@Serializable
internal data class TokenPairDto(
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String,
    @SerialName("token_type") val tokenType: String,
    @SerialName("expires_in") val expiresInSeconds: Int,
)

@Serializable
internal data class UserDto(
    val id: String,
    val email: String,
    @SerialName("full_name") val fullName: String,
    @SerialName("employee_code") val employeeCode: String,
    @SerialName("department_id") val departmentId: String? = null,
    @SerialName("department_name") val departmentName: String? = null,
    val role: String,
    val status: String,
)

internal interface AuthApi {
    @POST("auth/login")
    suspend fun login(@Body request: LoginRequestDto): LoginResponseDto

    @POST("auth/refresh")
    suspend fun refresh(@Body request: LogoutRequestDto): LoginResponseDto

    @POST("auth/logout")
    suspend fun logout(@Body request: LogoutRequestDto)
}

internal fun createAuthApi(
    baseUrl: String,
    client: OkHttpClient = OkHttpClient(),
): AuthApi {
    val json = Json { ignoreUnknownKeys = true }
    return Retrofit.Builder()
        .baseUrl(baseUrl)
        .client(client)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()
        .create(AuthApi::class.java)
}

fun createAuthRepository(baseUrl: String): AuthRepository = RemoteAuthRepository(
    api = createAuthApi(baseUrl),
)
