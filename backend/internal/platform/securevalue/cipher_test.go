package securevalue

import "testing"

func TestCipherRoundTrip(t *testing.T) {
	cipher, err := NewCipher("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}

	ciphertext, err := cipher.Encrypt("PHOTOSHOP-LICENSE-SECRET")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if string(ciphertext) == "PHOTOSHOP-LICENSE-SECRET" {
		t.Fatal("ciphertext must not equal plaintext")
	}
	plaintext, err := cipher.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plaintext != "PHOTOSHOP-LICENSE-SECRET" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
}
