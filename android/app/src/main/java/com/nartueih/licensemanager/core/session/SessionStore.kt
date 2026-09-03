package com.nartueih.licensemanager.core.session

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import com.nartueih.licensemanager.data.auth.EmployeeSession
import java.util.Base64
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import kotlinx.serialization.json.Json

interface SessionStore {
    val session: Flow<EmployeeSession?>

    suspend fun save(session: EmployeeSession)

    suspend fun clear()
}

internal class EncryptedSessionStore(
    private val dataStore: DataStore<Preferences>,
    private val cipher: AesGcmSessionCipher,
    private val json: Json = Json { ignoreUnknownKeys = true },
) : SessionStore {
    override val session: Flow<EmployeeSession?> = dataStore.data.map(::decode)

    override suspend fun save(session: EmployeeSession) {
        val plaintext = json.encodeToString(EmployeeSession.serializer(), session).encodeToByteArray()
        val encrypted = cipher.encrypt(plaintext)
        dataStore.edit { preferences ->
            preferences[CIPHERTEXT] = Base64.getEncoder().encodeToString(encrypted.ciphertext)
            preferences[INITIALIZATION_VECTOR] = Base64.getEncoder()
                .encodeToString(encrypted.initializationVector)
        }
    }

    override suspend fun clear() {
        dataStore.edit { preferences ->
            preferences.remove(CIPHERTEXT)
            preferences.remove(INITIALIZATION_VECTOR)
        }
    }

    private fun decode(preferences: Preferences): EmployeeSession? {
        val ciphertext = preferences[CIPHERTEXT] ?: return null
        val initializationVector = preferences[INITIALIZATION_VECTOR] ?: return null
        return try {
            val plaintext = cipher.decrypt(
                EncryptedPayload(
                    ciphertext = Base64.getDecoder().decode(ciphertext),
                    initializationVector = Base64.getDecoder().decode(initializationVector),
                ),
            )
            json.decodeFromString(EmployeeSession.serializer(), plaintext.decodeToString())
        } catch (_: Exception) {
            null
        }
    }

    private companion object {
        val CIPHERTEXT = stringPreferencesKey("session_ciphertext")
        val INITIALIZATION_VECTOR = stringPreferencesKey("session_iv")
    }
}

private val Context.employeeSessionDataStore by preferencesDataStore(
    name = "encrypted_employee_session",
)

fun createSessionStore(context: Context): SessionStore {
    val keyProvider = AndroidKeystoreSecretKeyProvider()
    return EncryptedSessionStore(
        dataStore = context.applicationContext.employeeSessionDataStore,
        cipher = AesGcmSessionCipher(keyProvider::getOrCreate),
    )
}
