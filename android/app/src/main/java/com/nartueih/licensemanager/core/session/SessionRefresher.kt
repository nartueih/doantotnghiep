package com.nartueih.licensemanager.core.session

import com.nartueih.licensemanager.data.auth.AuthRepository
import com.nartueih.licensemanager.data.auth.RefreshOutcome
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

sealed interface SessionRefreshResult {
    data class Refreshed(val accessToken: String) : SessionRefreshResult
    data object SessionExpired : SessionRefreshResult
    data object TemporarilyUnavailable : SessionRefreshResult
}

class SessionRefresher(
    private val sessionStore: SessionStore,
    private val authRepository: AuthRepository,
) {
    private val refreshMutex = Mutex()

    suspend fun refreshAfterUnauthorized(failedAccessToken: String): SessionRefreshResult =
        refreshMutex.withLock {
            val currentSession = sessionStore.session.first()
                ?: return@withLock SessionRefreshResult.SessionExpired

            if (currentSession.accessToken != failedAccessToken) {
                return@withLock SessionRefreshResult.Refreshed(currentSession.accessToken)
            }

            when (val outcome = authRepository.refresh(currentSession.refreshToken)) {
                is RefreshOutcome.Success -> {
                    sessionStore.save(outcome.session)
                    SessionRefreshResult.Refreshed(outcome.session.accessToken)
                }
                RefreshOutcome.InvalidSession -> {
                    sessionStore.clear()
                    SessionRefreshResult.SessionExpired
                }
                RefreshOutcome.ConnectionError,
                RefreshOutcome.ServerError,
                -> SessionRefreshResult.TemporarilyUnavailable
            }
        }
}
