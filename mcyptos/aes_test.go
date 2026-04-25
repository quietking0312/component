package mcyptos

import (
	"bytes"
	"fmt"
	"testing"
)

func TestAESCipher(t *testing.T) {
	key := []byte("1234567890123456") // 16 bytes for AES-128
	iv := []byte("abcdefghabcdefgh")  // 16 bytes

	c, err := NewAESCipher(key, iv)
	if err != nil {
		t.Fatalf("NewAESCipher failed: %v", err)
	}

	plaintext := []byte("hello world")
	ciphertext, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := c.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("decrypted text mismatch: got %s, want %s", decrypted, plaintext)
	}
}

func TestAESCipherWithNilIV(t *testing.T) {
	key := []byte("12345678901234567890123456789012") // 32 bytes for AES-256

	c, err := NewAESCipher(key, nil)
	if err != nil {
		t.Fatalf("NewAESCipher failed: %v", err)
	}

	plaintext := []byte("test data with AES-256")
	ciphertext, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := c.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("decrypted text mismatch: got %s, want %s", decrypted, plaintext)
	}
}

func TestBase64(t *testing.T) {
	data := []byte("hello base64")
	encoded := EncodeBase64(data)
	decoded, err := DecodeBase64(encoded)
	if err != nil {
		t.Fatalf("DecodeBase64 failed: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Fatalf("base64 round-trip failed")
	}
}

func TestAESCipher_InvalidKey(t *testing.T) {
	_, err := NewAESCipher([]byte("short"), nil)
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestAESCipher_InvalidCiphertext(t *testing.T) {
	key := []byte("1234567890123456")
	c, _ := NewAESCipher(key, nil)
	_, err := c.Decrypt([]byte("short"))
	if err == nil {
		t.Fatal("expected error for invalid ciphertext length")
	}
}

func TestDecryptAES_Base64Key(t *testing.T) {
	k := "teujWMYGcQob6OcCVRruyHMtENRcIlvuM4ghIWqYCiF"
	kstr, err := DecodeBase64(k + "=")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(kstr))
	fmt.Println(string(kstr))
}
