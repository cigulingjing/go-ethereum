package main

import (
	"crypto/aes"
	"crypto/cipher"
)

// AesCBCEncrypt encrypts plaintext with AES-CBC and PKCS#7 padding.
func AesCBCEncrypt(key, iv, plaintext []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil || len(iv) != block.BlockSize() {
		return nil
	}
	padded := aesCBCPKCS7Pad(plaintext, block.BlockSize())
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)
	return ciphertext
}

func aesCBCPKCS7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+padding)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(padding)
	}
	return out
}
