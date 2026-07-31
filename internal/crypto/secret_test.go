package crypto_test

import (
	"testing"

	"github.com/puppe1990/kiwify_dashboard/internal/crypto"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := crypto.DeriveKey("test-app-secret-for-unit-tests-32b")
	ct, err := crypto.Encrypt(key, "super-secret-value")
	if err != nil {
		t.Fatal(err)
	}
	if ct == "super-secret-value" {
		t.Fatal("ciphertext must not equal plaintext")
	}
	pt, err := crypto.Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if pt != "super-secret-value" {
		t.Fatalf("got %q", pt)
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	key1 := crypto.DeriveKey("secret-one-aaaaaaaaaaaaaaaaaaaa")
	key2 := crypto.DeriveKey("secret-two-bbbbbbbbbbbbbbbbbbbb")
	ct, err := crypto.Encrypt(key1, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := crypto.Decrypt(key2, ct); err == nil {
		t.Fatal("expected error")
	}
}
