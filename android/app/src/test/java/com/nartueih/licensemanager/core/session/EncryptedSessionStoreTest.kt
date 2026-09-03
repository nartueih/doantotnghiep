package com.nartueih.licensemanager.core.session

import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.emptyPreferences
import androidx.datastore.preferences.core.stringPreferencesKey
import com.nartueih.licensemanager.data.auth.EmployeeSession
import com.nartueih.licensemanager.data.auth.EmployeeUser
import javax.crypto.KeyGenerator
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class EncryptedSessionStoreTest {
    @Test
    fun savePersistsOnlyEncryptedDataAndSessionCanBeRestored() = runTest {
        val dataStore = InMemoryPreferencesDataStore()
        val store = EncryptedSessionStore(dataStore, newCipher())

        store.save(sampleSession)

        assertEquals(sampleSession, store.session.first())
        val persistedValues = dataStore.data.first().asMap().values.joinToString()
        assertFalse(persistedValues.contains("access-token"))
        assertFalse(persistedValues.contains("refresh-token"))
    }

    @Test
    fun clearRemovesThePersistedSession() = runTest {
        val dataStore = InMemoryPreferencesDataStore()
        val store = EncryptedSessionStore(dataStore, newCipher())
        store.save(sampleSession)

        store.clear()

        assertNull(store.session.first())
    }

    @Test
    fun corruptedEncryptedDataIsTreatedAsNoSession() = runTest {
        val dataStore = InMemoryPreferencesDataStore()
        val store = EncryptedSessionStore(dataStore, newCipher())
        store.save(sampleSession)
        dataStore.edit { preferences ->
            preferences[stringPreferencesKey("session_ciphertext")] = "not-valid-base64"
        }

        assertNull(store.session.first())
    }

    private class InMemoryPreferencesDataStore : DataStore<Preferences> {
        private val state = MutableStateFlow(emptyPreferences())
        override val data = state

        override suspend fun updateData(transform: suspend (t: Preferences) -> Preferences): Preferences {
            val updated = transform(state.value)
            state.value = updated
            return updated
        }
    }

    private fun newCipher(): AesGcmSessionCipher {
        val key = KeyGenerator.getInstance("AES").apply { init(256) }.generateKey()
        return AesGcmSessionCipher { key }
    }

    private val sampleSession = EmployeeSession(
        accessToken = "access-token",
        refreshToken = "refresh-token",
        expiresInSeconds = 900,
        user = EmployeeUser(
            id = "employee-1",
            email = "employee@local.test",
            fullName = "Nguyễn Hoàng Anh",
            employeeCode = "EMP-001",
            departmentId = "department-1",
            departmentName = "Công nghệ thông tin",
        ),
    )
}
