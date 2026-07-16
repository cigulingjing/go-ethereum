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

// AesCBCDecrypt decrypts AES-CBC ciphertext and removes PKCS#7 padding.
func AesCBCDecrypt(key, iv, ciphertext []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil || len(iv) != block.BlockSize() {
		return nil
	}
	if len(ciphertext) == 0 || len(ciphertext)%block.BlockSize() != 0 {
		return nil
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)
	return aesCBCPKCS7Unpad(plaintext, block.BlockSize())
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

func aesCBCPKCS7Unpad(data []byte, blockSize int) []byte {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil
	}
	for _, b := range data[len(data)-padding:] {
		if int(b) != padding {
			return nil
		}
	}
	out := make([]byte, len(data)-padding)
	copy(out, data[:len(data)-padding])
	return out
}
