package com.nartueih.licensemanager.core.network

import com.nartueih.licensemanager.core.session.SessionRefreshResult
import com.nartueih.licensemanager.core.session.SessionRefresher
import com.nartueih.licensemanager.core.session.SessionStore
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import okhttp3.Authenticator
import okhttp3.Interceptor
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.Route

fun createAuthenticatedHttpClient(
    sessionStore: SessionStore,
    sessionRefresher: SessionRefresher,
): OkHttpClient = OkHttpClient.Builder()
    .addInterceptor(SessionAuthorizationInterceptor(sessionStore))
    .authenticator(SessionAuthenticator(sessionRefresher))
    .build()

private class SessionAuthorizationInterceptor(
    private val sessionStore: SessionStore,
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val accessToken = runBlocking { sessionStore.session.first()?.accessToken }
        val request = accessToken?.let {
            chain.request().newBuilder()
                .header(AUTHORIZATION_HEADER, "Bearer $it")
                .build()
        } ?: chain.request()
        return chain.proceed(request)
    }
}

private class SessionAuthenticator(
    private val sessionRefresher: SessionRefresher,
) : Authenticator {
    override fun authenticate(route: Route?, response: Response): Request? {
        if (response.responseCount() >= MAX_ATTEMPTS) return null
        val failedAccessToken = response.request.header(AUTHORIZATION_HEADER)
            ?.removePrefix("Bearer ")
            ?.takeIf(String::isNotBlank)
            ?: return null

        return when (
            val result = runBlocking {
                sessionRefresher.refreshAfterUnauthorized(failedAccessToken)
            }
        ) {
            is SessionRefreshResult.Refreshed -> response.request.newBuilder()
                .header(AUTHORIZATION_HEADER, "Bearer ${result.accessToken}")
                .build()
            SessionRefreshResult.SessionExpired,
            SessionRefreshResult.TemporarilyUnavailable,
            -> null
        }
    }
}

private fun Response.responseCount(): Int {
    var count = 1
    var prior = priorResponse
    while (prior != null) {
        count += 1
        prior = prior.priorResponse
    }
    return count
}

private const val AUTHORIZATION_HEADER = "Authorization"
private const val MAX_ATTEMPTS = 2
