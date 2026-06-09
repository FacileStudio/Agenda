package crypto

import (
	"testing"
)

func TestRoundTrip(t *testing.T) {
	key := DeriveKey("test-encryption-key-for-courrier")
	plaintext := "my-secret-imap-password"

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if encrypted == plaintext {
		t.Fatal("encrypted output should differ from plaintext")
	}
	if encrypted == "" {
		t.Fatal("encrypted output should not be empty")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEmptyString(t *testing.T) {
	key := DeriveKey("test-key")

	enc, err := Encrypt("", key)
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	if enc != "" {
		t.Fatal("empty input should return empty output")
	}

	dec, err := Decrypt("", key)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if dec != "" {
		t.Fatal("empty input should return empty output")
	}
}

func TestUniqueNonce(t *testing.T) {
	key := DeriveKey("test-key")
	plaintext := "same-password"

	enc1, _ := Encrypt(plaintext, key)
	enc2, _ := Encrypt(plaintext, key)

	if enc1 == enc2 {
		t.Fatal("two encryptions of the same plaintext should produce different ciphertexts")
	}

	dec1, _ := Decrypt(enc1, key)
	dec2, _ := Decrypt(enc2, key)
	if dec1 != plaintext || dec2 != plaintext {
		t.Fatal("both should decrypt to the same plaintext")
	}
}

func TestWrongKey(t *testing.T) {
	key1 := DeriveKey("key-one")
	key2 := DeriveKey("key-two")

	encrypted, _ := Encrypt("secret", key1)
	_, err := Decrypt(encrypted, key2)
	if err == nil {
		t.Fatal("decrypting with wrong key should fail")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	k1 := DeriveKey("same-input")
	k2 := DeriveKey("same-input")

	if len(k1) != 32 {
		t.Fatalf("key should be 32 bytes, got %d", len(k1))
	}
	for i := range k1 {
		if k1[i] != k2[i] {
			t.Fatal("same input should produce same key")
		}
	}
}
