package auth

import (
    "crypto/rsa"
    "encoding/base64"
    "fmt"
)

// CryptoManager manages the hybrid RSA/AES encryption system
type CryptoManager struct {
    rsaKeyPair *RSAKeyPair
    aesMode    string // "GCM" or "CBC"
}

// NewCryptoManager creates a new CryptoManager
func NewCryptoManager(rsaKeyPair *RSAKeyPair, aesMode string) *CryptoManager {
    return &CryptoManager{
        rsaKeyPair: rsaKeyPair,
        aesMode:    aesMode,
    }
}

// GetPublicKey returns the RSA public key
func (cm *CryptoManager) GetPublicKey() *rsa.PublicKey {
    return cm.rsaKeyPair.PublicKey
}

// GetPublicKeyPEM returns the RSA public key as PEM
func (cm *CryptoManager) GetPublicKeyPEM() ([]byte, error) {
    return cm.rsaKeyPair.GetPublicKeyPEM()
}

// EncryptMessage encrypts a message using AES with a session key
// Returns: encryptedMessage (base64), error
func (cm *CryptoManager) EncryptMessage(sessionKey []byte, message string) (string, error) {
    var ciphertext []byte
    var err error

    plaintext := []byte(message)

    switch cm.aesMode {
    case "GCM":
        ciphertext, err = EncryptAESGCM(sessionKey, plaintext)
    case "CBC":
        ciphertext, err = EncryptAESCBC(sessionKey, plaintext)
    default:
        return "", fmt.Errorf("unsupported AES mode: %s", cm.aesMode)
    }

    if err != nil {
        return "", fmt.Errorf("encryption failed: %w", err)
    }

    // Encode to base64 for transmission
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptMessage decrypts a message using AES with a session key
// Expects encryptedMessage to be base64-encoded
func (cm *CryptoManager) DecryptMessage(sessionKey []byte, encryptedMessage string) (string, error) {
    // Decode from base64
    ciphertext, err := base64.StdEncoding.DecodeString(encryptedMessage)
    if err != nil {
        return "", fmt.Errorf("base64 decode failed: %w", err)
    }

    var plaintext []byte

    switch cm.aesMode {
    case "GCM":
        plaintext, err = DecryptAESGCM(sessionKey, ciphertext)
    case "CBC":
        plaintext, err = DecryptAESCBC(sessionKey, ciphertext)
    default:
        return "", fmt.Errorf("unsupported AES mode: %s", cm.aesMode)
    }

    if err != nil {
        return "", fmt.Errorf("decryption failed: %w", err)
    }

    return string(plaintext), nil
}

// EncryptSessionKey encrypts an AES session key using RSA public key
// Returns base64-encoded encrypted key
func (cm *CryptoManager) EncryptSessionKey(publicKey *rsa.PublicKey, sessionKey []byte) (string, error) {
    encryptedKey, err := EncryptRSA(publicKey, sessionKey)
    if err != nil {
        return "", fmt.Errorf("failed to encrypt session key: %w", err)
    }

    return base64.StdEncoding.EncodeToString(encryptedKey), nil
}

// DecryptSessionKey decrypts an RSA-encrypted session key
// Expects encryptedKey to be base64-encoded
func (cm *CryptoManager) DecryptSessionKey(encryptedKey string) ([]byte, error) {
    // Decode from base64
    ciphertext, err := base64.StdEncoding.DecodeString(encryptedKey)
    if err != nil {
        return nil, fmt.Errorf("base64 decode failed: %w", err)
    }

    sessionKey, err := DecryptRSA(cm.rsaKeyPair.PrivateKey, ciphertext)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt session key: %w", err)
    }

    return sessionKey, nil
}

// GenerateSessionKey generates a new AES session key
func (cm *CryptoManager) GenerateSessionKey(keySize int) ([]byte, error) {
    return GenerateAESKey(keySize)
}

// EncryptWithPublicKey encrypts data directly with RSA public key (for small data)
func EncryptWithPublicKey(publicKey *rsa.PublicKey, data []byte) (string, error) {
    ciphertext, err := EncryptRSA(publicKey, data)
    if err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptWithPrivateKey decrypts data with RSA private key
func (cm *CryptoManager) DecryptWithPrivateKey(encryptedData string) ([]byte, error) {
    ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
    if err != nil {
        return nil, fmt.Errorf("base64 decode failed: %w", err)
    }

    return DecryptRSA(cm.rsaKeyPair.PrivateKey, ciphertext)
}
