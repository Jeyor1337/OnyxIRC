package auth

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "crypto/x509"
    "encoding/pem"
    "fmt"
    "os"
)

type RSAKeyPair struct {
    PrivateKey *rsa.PrivateKey
    PublicKey  *rsa.PublicKey
}

func GenerateRSAKeyPair(keySize int) (*RSAKeyPair, error) {
    if keySize != 2048 && keySize != 4096 {
        return nil, fmt.Errorf("invalid key size: must be 2048 or 4096")
    }

    privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
    if err != nil {
        return nil, fmt.Errorf("failed to generate RSA key: %w", err)
    }

    return &RSAKeyPair{
        PrivateKey: privateKey,
        PublicKey:  &privateKey.PublicKey,
    }, nil
}

func (kp *RSAKeyPair) SavePrivateKeyToFile(filename string) error {
    
    privateKeyBytes := x509.MarshalPKCS1PrivateKey(kp.PrivateKey)
    privateKeyPEM := &pem.Block{
        Type:  "RSA PRIVATE KEY",
        Bytes: privateKeyBytes,
    }

    file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
    if err != nil {
        return fmt.Errorf("failed to create private key file: %w", err)
    }
    defer file.Close()

    if err := pem.Encode(file, privateKeyPEM); err != nil {
        return fmt.Errorf("failed to write private key: %w", err)
    }

    return nil
}

func (kp *RSAKeyPair) SavePublicKeyToFile(filename string) error {
    
    publicKeyBytes, err := x509.MarshalPKIXPublicKey(kp.PublicKey)
    if err != nil {
        return fmt.Errorf("failed to marshal public key: %w", err)
    }

    publicKeyPEM := &pem.Block{
        Type:  "RSA PUBLIC KEY",
        Bytes: publicKeyBytes,
    }

    file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
    if err != nil {
        return fmt.Errorf("failed to create public key file: %w", err)
    }
    defer file.Close()

    if err := pem.Encode(file, publicKeyPEM); err != nil {
        return fmt.Errorf("failed to write public key: %w", err)
    }

    return nil
}

func LoadPrivateKeyFromFile(filename string) (*rsa.PrivateKey, error) {
    
    keyData, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read private key file: %w", err)
    }

    block, _ := pem.Decode(keyData)
    if block == nil {
        return nil, fmt.Errorf("failed to decode PEM block")
    }

    privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
    if err != nil {
        return nil, fmt.Errorf("failed to parse private key: %w", err)
    }

    return privateKey, nil
}

func LoadPublicKeyFromFile(filename string) (*rsa.PublicKey, error) {
    
    keyData, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read public key file: %w", err)
    }

    block, _ := pem.Decode(keyData)
    if block == nil {
        return nil, fmt.Errorf("failed to decode PEM block")
    }

    publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil {
        return nil, fmt.Errorf("failed to parse public key: %w", err)
    }

    publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
    if !ok {
        return nil, fmt.Errorf("not an RSA public key")
    }

    return publicKey, nil
}

func EncryptRSA(publicKey *rsa.PublicKey, plaintext []byte) ([]byte, error) {
    ciphertext, err := rsa.EncryptOAEP(
        sha256.New(),
        rand.Reader,
        publicKey,
        plaintext,
        nil,
    )
    if err != nil {
        return nil, fmt.Errorf("RSA encryption failed: %w", err)
    }

    return ciphertext, nil
}

func DecryptRSA(privateKey *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
    plaintext, err := rsa.DecryptOAEP(
        sha256.New(),
        rand.Reader,
        privateKey,
        ciphertext,
        nil,
    )
    if err != nil {
        return nil, fmt.Errorf("RSA decryption failed: %w", err)
    }

    return plaintext, nil
}

func (kp *RSAKeyPair) GetPublicKeyPEM() ([]byte, error) {
    publicKeyBytes, err := x509.MarshalPKIXPublicKey(kp.PublicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal public key: %w", err)
    }

    publicKeyPEM := pem.EncodeToMemory(&pem.Block{
        Type:  "RSA PUBLIC KEY",
        Bytes: publicKeyBytes,
    })

    return publicKeyPEM, nil
}
