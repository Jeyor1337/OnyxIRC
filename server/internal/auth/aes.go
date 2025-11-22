package auth

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "fmt"
    "io"
)

func GenerateAESKey(keySize int) ([]byte, error) {
    if keySize != 128 && keySize != 192 && keySize != 256 {
        return nil, fmt.Errorf("invalid AES key size: must be 128, 192, or 256 bits")
    }

    key := make([]byte, keySize/8) 
    if _, err := rand.Read(key); err != nil {
        return nil, fmt.Errorf("failed to generate AES key: %w", err)
    }

    return key, nil
}

func EncryptAESGCM(key, plaintext []byte) ([]byte, error) {
    
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, fmt.Errorf("failed to generate nonce: %w", err)
    }

    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

    return ciphertext, nil
}

func DecryptAESGCM(key, ciphertext []byte) ([]byte, error) {
    
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("decryption failed: %w", err)
    }

    return plaintext, nil
}

func EncryptAESCBC(key, plaintext []byte) ([]byte, error) {
    
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    plaintext = pkcs7Pad(plaintext, block.BlockSize())

    iv := make([]byte, aes.BlockSize)
    if _, err := io.ReadFull(rand.Reader, iv); err != nil {
        return nil, fmt.Errorf("failed to generate IV: %w", err)
    }

    ciphertext := make([]byte, len(plaintext))
    mode := cipher.NewCBCEncrypter(block, iv)
    mode.CryptBlocks(ciphertext, plaintext)

    return append(iv, ciphertext...), nil
}

func DecryptAESCBC(key, ciphertext []byte) ([]byte, error) {
    
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    if len(ciphertext) < aes.BlockSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    iv := ciphertext[:aes.BlockSize]
    ciphertext = ciphertext[aes.BlockSize:]

    if len(ciphertext)%aes.BlockSize != 0 {
        return nil, fmt.Errorf("ciphertext is not a multiple of block size")
    }

    plaintext := make([]byte, len(ciphertext))
    mode := cipher.NewCBCDecrypter(block, iv)
    mode.CryptBlocks(plaintext, ciphertext)

    plaintext, err = pkcs7Unpad(plaintext, block.BlockSize())
    if err != nil {
        return nil, fmt.Errorf("failed to remove padding: %w", err)
    }

    return plaintext, nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
    padding := blockSize - (len(data) % blockSize)
    padText := make([]byte, padding)
    for i := range padText {
        padText[i] = byte(padding)
    }
    return append(data, padText...)
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
    length := len(data)
    if length == 0 {
        return nil, fmt.Errorf("invalid padding: empty data")
    }

    padding := int(data[length-1])
    if padding > blockSize || padding == 0 {
        return nil, fmt.Errorf("invalid padding size")
    }

    for i := length - padding; i < length; i++ {
        if data[i] != byte(padding) {
            return nil, fmt.Errorf("invalid padding")
        }
    }

    return data[:length-padding], nil
}
