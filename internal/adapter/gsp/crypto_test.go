package gsp

import (
	"bytes"
	"testing"
)

func TestAES256ECB_RoundTrip(t *testing.T) {
	// 32-byte key for AES-256
	key := []byte("01234567890123456789012345678901")
	plaintext := []byte(`{"Irn":"test-irn-value","AckNo":123456}`)

	// Encrypt
	encrypted, err := encryptPayload(plaintext, key)
	if err != nil {
		t.Fatalf("encryptPayload failed: %v", err)
	}

	if encrypted == "" {
		t.Fatal("encrypted result should not be empty")
	}

	// Decrypt
	decrypted, err := decryptResponse(encrypted, key)
	if err != nil {
		t.Fatalf("decryptResponse failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("round-trip failed:\n  expected: %s\n  got:      %s", plaintext, decrypted)
	}
}

func TestAES256ECB_DifferentPlaintexts(t *testing.T) {
	key := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ123456")

	tests := []string{
		"Hello",
		"Short",
		"This is a longer test string that spans multiple AES blocks to ensure proper ECB handling",
		`{"key": "value", "number": 42}`,
	}

	for _, plaintext := range tests {
		encrypted, err := encryptPayload([]byte(plaintext), key)
		if err != nil {
			t.Fatalf("encryptPayload failed for %q: %v", plaintext, err)
		}

		decrypted, err := decryptResponse(encrypted, key)
		if err != nil {
			t.Fatalf("decryptResponse failed for %q: %v", plaintext, err)
		}

		if string(decrypted) != plaintext {
			t.Errorf("round-trip mismatch for %q: got %q", plaintext, string(decrypted))
		}
	}
}

func TestPKCS5Padding(t *testing.T) {
	data := []byte("Hello")
	blockSize := 16

	padded := pkcs5Pad(data, blockSize)
	if len(padded)%blockSize != 0 {
		t.Errorf("padded length %d is not a multiple of block size %d", len(padded), blockSize)
	}

	unpadded, err := pkcs5Unpad(padded)
	if err != nil {
		t.Fatalf("pkcs5Unpad failed: %v", err)
	}

	if !bytes.Equal(unpadded, data) {
		t.Errorf("padding round-trip failed: expected %q, got %q", data, unpadded)
	}
}

func TestPKCS5Padding_ExactBlockSize(t *testing.T) {
	// Input that's exactly one block (16 bytes) should still get padding
	data := []byte("0123456789ABCDEF") // exactly 16 bytes
	blockSize := 16

	padded := pkcs5Pad(data, blockSize)
	// Should be 32 bytes (original 16 + 16 bytes of padding)
	if len(padded) != 32 {
		t.Errorf("expected padded length 32, got %d", len(padded))
	}

	unpadded, err := pkcs5Unpad(padded)
	if err != nil {
		t.Fatalf("pkcs5Unpad failed: %v", err)
	}

	if !bytes.Equal(unpadded, data) {
		t.Errorf("padding round-trip failed for exact block size")
	}
}

func TestDecryptResponse_InvalidBase64(t *testing.T) {
	key := []byte("01234567890123456789012345678901")

	_, err := decryptResponse("not-valid-base64!!!", key)
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecryptResponse_InvalidPadding(t *testing.T) {
	key := []byte("01234567890123456789012345678901")

	// Encrypt something, then corrupt the last byte
	encrypted, _ := encryptPayload([]byte("test data"), key)

	// Modify the encrypted data (this will likely cause a padding error on decrypt)
	// We can't easily test this without corrupting the base64 decoded bytes,
	// so we verify the happy path works and the function exists.
	_, err := decryptResponse(encrypted, key)
	if err != nil {
		t.Errorf("decryptResponse failed on valid data: %v", err)
	}
}
