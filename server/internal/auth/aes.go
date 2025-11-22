package auth

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "fmt"
    "io"
)

// GenerateAESKey generates a random AES key of the specified size (in bits)
func GenerateAESKey(keySize int) ([]byte, error) {
    if keySize != 128 && keySize != 192 && keySize != 256 {
        return nil, fmt.Errorf("invalid AES key size: must be 128, 192, or 256 bits")
    }

    key := make([]byte, keySize/8) // Convert bits to bytes
    if _, err := rand.Read(key); err != nil {
        return nil, fmt.Errorf("failed to generate AES key: %w", err)
    }

    return key, nil
}

// EncryptAESGCM encrypts plaintext using AES-256-GCM
// Returns ciphertext with nonce prepended
func EncryptAESGCM(key, plaintext []byte) ([]byte, error) {
    // Create AES cipher
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    // Create GCM mode
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    // Generate nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, fmt.Errorf("failed to generate nonce: %w", err)
    }

    // Encrypt and authenticate
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

    return ciphertext, nil
}

// DecryptAESGCM decrypts ciphertext using AES-256-GCM
// Expects ciphertext with nonce prepended
func DecryptAESGCM(key, ciphertext []byte) ([]byte, error) {
    // Create AES cipher
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    // Create GCM mode
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    // Check minimum length
    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    // Extract nonce and ciphertext
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

    // Decrypt and verify
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("decryption failed: %w", err)
    }

    return plaintext, nil
}

// EncryptAESCBC encrypts plaintext using AES-CBC with PKCS7 padding
// Returns ciphertext with IV prepended
func EncryptAESCBC(key, plaintext []byte) ([]byte, error) {
    // Create AES cipher
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    // Apply PKCS7 padding
    plaintext = pkcs7Pad(plaintext, block.BlockSize())

    // Generate IV
    iv := make([]byte, aes.BlockSize)
    if _, err := io.ReadFull(rand.Reader, iv); err != nil {
        return nil, fmt.Errorf("failed to generate IV: %w", err)
    }

    // Encrypt
    ciphertext := make([]byte, len(plaintext))
    mode := cipher.NewCBCEncrypter(block, iv)
    mode.CryptBlocks(ciphertext, plaintext)

    // Prepend IV to ciphertext
    return append(iv, ciphertext...), nil
}

// DecryptAESCBC decrypts ciphertext using AES-CBC
// Expects ciphertext with IV prepended
func DecryptAESCBC(key, ciphertext []byte) ([]byte, error) {
    // Create AES cipher
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    // Check minimum length
    if len(ciphertext) < aes.BlockSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    // Extract IV and ciphertext
    iv := ciphertext[:aes.BlockSize]
    ciphertext = ciphertext[aes.BlockSize:]

    // Check ciphertext length
    if len(ciphertext)%aes.BlockSize != 0 {
        return nil, fmt.Errorf("ciphertext is not a multiple of block size")
    }

    // Decrypt
    plaintext := make([]byte, len(ciphertext))
    mode := cipher.NewCBCDecrypter(block, iv)
    mode.CryptBlocks(plaintext, ciphertext)

    // Remove PKCS7 padding
    plaintext, err = pkcs7Unpad(plaintext, block.BlockSize())
    if err != nil {
        return nil, fmt.Errorf("failed to remove padding: %w", err)
    }

    return plaintext, nil
}

// pkcs7Pad applies PKCS7 padding to data
func pkcs7Pad(data []byte, blockSize int) []byte {
    padding := blockSize - (len(data) % blockSize)
    padText := make([]byte, padding)
    for i := range padText {
        padText[i] = byte(padding)
    }
    return append(data, padText...)
}

// pkcs7Unpad removes PKCS7 padding from data
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
    length := len(data)
    if length == 0 {
        return nil, fmt.Errorf("invalid padding: empty data")
    }

    padding := int(data[length-1])
    if padding > blockSize || padding == 0 {
        return nil, fmt.Errorf("invalid padding size")
    }

    // Verify padding
    for i := length - padding; i < length; i++ {
        if data[i] != byte(padding) {
            return nil, fmt.Errorf("invalid padding")
        }
    }

    return data[:length-padding], nil
}
