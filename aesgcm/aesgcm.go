package aesgcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	saltSize = 8
	keySize  = 32

	// AES-GCM 要求 nonce 长度为 12，参考 https://pkg.go.dev/crypto/cipher#example-NewGCM-Encrypt
	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	// 2^32 约等于 43 亿，因此个人使用不需要担心冲突。
	nonceSize = 12
)

// NewKey 利用 scrypt 算法，将一个字符串转化为一个 key.
// 参考 https://pkg.go.dev/golang.org/x/crypto/scrypt
func NewKey(passphrase string) []byte {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic(err)
	}
	dk, err := scrypt.Key([]byte(passphrase), salt, 32768, 8, 1, keySize)
	if err != nil {
		panic(err)
	}
	return dk
}

// NewNonce 生成一个随机的 nonce, 其长度为 nonceSize.
func NewNonce() []byte {
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err)
	}
	return nonce
}

// NewCipher 使用的 key 应该由 NewKey 生成。
func NewCipher(key []byte) cipher.Block {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	return block
}

// NewGCM 使用的 key 应该由 NewKey 生成。
func NewGCM(key []byte) AEAD {
	block := NewCipher(key)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	return AEAD{gcm}
}

// AEAD 只是简单地包裹了 cipher.AEAD, 以便提供更方便的 Seal 和 Open 方法。
type AEAD struct {
	gcm cipher.AEAD
}

// Seal 参考了 nacl/secretbox 的做法，将 nonce 与密文绑在一起。
func (aead AEAD) Seal(nonce, plaintext []byte) []byte {
	return aead.gcm.Seal(nonce, nonce, plaintext, nil)
}

// Encrypt 不需要用户提供 nonce, 采用一个新的随机 nonce 来加密。
func (aead AEAD) Encrypt(plaintext []byte) []byte {
	nonce := NewNonce()
	return aead.Seal(nonce, plaintext)
}

// Decrypt 解密 ciphertext, nonce 从 ciphertext 里获取。
func (aead AEAD) Decrypt(ciphertext []byte) ([]byte, error) {
	plaintext, err := aead.gcm.Open(
		nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
