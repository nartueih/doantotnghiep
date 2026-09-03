package com.nartueih.licensemanager.core.session

import java.security.GeneralSecurityException
import javax.crypto.KeyGenerator
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertThrows
import org.junit.Test

class AesGcmSessionCipherTest {
    private val key = KeyGenerator.getInstance("AES").apply { init(256) }.generateKey()
    private val cipher = AesGcmSessionCipher { key }

    @Test
    fun encryptThenDecryptRestoresPlaintextWithoutStoringItDirectly() {
        val plaintext = "access-token|refresh-token".toByteArray()

        val encrypted = cipher.encrypt(plaintext)

        assertFalse(encrypted.ciphertext.contentEquals(plaintext))
        assertArrayEquals(plaintext, cipher.decrypt(encrypted))
    }

    @Test
    fun decryptRejectsCiphertextThatWasModified() {
        val encrypted = cipher.encrypt("sensitive-session".toByteArray())
        encrypted.ciphertext[0] = (encrypted.ciphertext[0].toInt() xor 1).toByte()

        assertThrows(GeneralSecurityException::class.java) {
            cipher.decrypt(encrypted)
        }
    }
}
