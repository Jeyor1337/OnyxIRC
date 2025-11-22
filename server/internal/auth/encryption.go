package auth

import (
    "crypto/rsa"
    "encoding/base64"
    "fmt"
)

type CryptoManager struct {
    rsaKeyPair *RSAKeyPair
    aesMode    string 
}

func NewCryptoManager(rsaKeyPair *RSAKeyPair, aesMode string) *CryptoManager {
    return &CryptoManager{
        rsaKeyPair: rsaKeyPair,
        aesMode:    aesMode,
    }
}

func (cm *CryptoManager) GetPublicKey() *rsa.PublicKey {
    return cm.rsaKeyPair.PublicKey
}

func (cm *CryptoManager) GetPublicKeyPEM() ([]byte, error) {
    return cm.rsaKeyPair.GetPublicKeyPEM()
}

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

    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (cm *CryptoManager) DecryptMessage(sessionKey []byte, encryptedMessage string) (string, error) {
    
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

func (cm *CryptoManager) EncryptSessionKey(publicKey *rsa.PublicKey, sessionKey []byte) (string, error) {
    encryptedKey, err := EncryptRSA(publicKey, sessionKey)
    if err != nil {
        return "", fmt.Errorf("failed to encrypt session key: %w", err)
    }

    return base64.StdEncoding.EncodeToString(encryptedKey), nil
}

func (cm *CryptoManager) DecryptSessionKey(encryptedKey string) ([]byte, error) {
    
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

func (cm *CryptoManager) GenerateSessionKey(keySize int) ([]byte, error) {
    return GenerateAESKey(keySize)
}

func EncryptWithPublicKey(publicKey *rsa.PublicKey, data []byte) (string, error) {
    ciphertext, err := EncryptRSA(publicKey, data)
    if err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (cm *CryptoManager) DecryptWithPrivateKey(encryptedData string) ([]byte, error) {
    ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
    if err != nil {
        return nil, fmt.Errorf("base64 decode failed: %w", err)
    }

    return DecryptRSA(cm.rsaKeyPair.PrivateKey, ciphertext)
}
