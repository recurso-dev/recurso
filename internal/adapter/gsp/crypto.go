package gsp

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

// decryptSEK decrypts the Session Encryption Key (SEK) using RSA private key.
// NIC uses RSA/ECB/PKCS1Padding for the SEK encryption.
func decryptSEK(encryptedSEKBase64 string, privateKey *rsa.PrivateKey) ([]byte, error) {
	encryptedSEK, err := base64.StdEncoding.DecodeString(encryptedSEKBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode SEK: %w", err)
	}

	sek, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, encryptedSEK)
	if err != nil {
		return nil, fmt.Errorf("failed to RSA decrypt SEK: %w", err)
	}

	return sek, nil
}

// encryptPayload encrypts data using AES-256/ECB/PKCS5Padding, returns base64 encoded string.
// NIC mandates ECB mode for payload encryption.
func encryptPayload(data []byte, sek []byte) (string, error) {
	block, err := aes.NewCipher(sek)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	padded := pkcs5Pad(data, aes.BlockSize)
	encrypted := make([]byte, len(padded))

	// ECB mode: encrypt each block independently
	for i := 0; i < len(padded); i += aes.BlockSize {
		block.Encrypt(encrypted[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// decryptResponse decrypts a base64 encoded AES-256/ECB/PKCS5Padding response.
func decryptResponse(encryptedBase64 string, sek []byte) ([]byte, error) {
	encrypted, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode response: %w", err)
	}

	block, err := aes.NewCipher(sek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	if len(encrypted)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("encrypted data is not a multiple of block size")
	}

	decrypted := make([]byte, len(encrypted))

	// ECB mode: decrypt each block independently
	for i := 0; i < len(encrypted); i += aes.BlockSize {
		block.Decrypt(decrypted[i:i+aes.BlockSize], encrypted[i:i+aes.BlockSize])
	}

	unpadded, err := pkcs5Unpad(decrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to unpad decrypted data: %w", err)
	}

	return unpadded, nil
}

// parseRSAPrivateKey parses a PEM encoded RSA private key.
func parseRSAPrivateKey(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	// Try PKCS8 first, then PKCS1
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS8 key is not RSA")
		}
		return rsaKey, nil
	}

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
	}
	return rsaKey, nil
}

// pkcs5Pad adds PKCS5/PKCS7 padding.
func pkcs5Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

// pkcs5Unpad removes PKCS5/PKCS7 padding.
func pkcs5Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padding := int(data[len(data)-1])
	if padding > len(data) || padding > aes.BlockSize || padding == 0 {
		return nil, fmt.Errorf("invalid padding size: %d", padding)
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding bytes")
		}
	}
	return data[:len(data)-padding], nil
}
