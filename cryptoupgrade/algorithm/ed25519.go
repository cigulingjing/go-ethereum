package main

import (
	"crypto/ed25519"
	"crypto/rand"
)

// Ed25519Keygen returns a private key. A 32-byte seed makes keygen deterministic.
func Ed25519Keygen(seed []byte) []byte {
	if len(seed) == ed25519.SeedSize {
		return append([]byte(nil), ed25519.NewKeyFromSeed(seed)...)
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil
	}
	return append([]byte(nil), privateKey...)
}

// Ed25519PublicKey derives the public key from a private key or seed.
func Ed25519PublicKey(privateKey []byte) []byte {
	if len(privateKey) == ed25519.SeedSize {
		privateKey = ed25519.NewKeyFromSeed(privateKey)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil
	}
	publicKey := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
	return append([]byte(nil), publicKey...)
}

// Ed25519Sign signs message with an Ed25519 private key or seed.
func Ed25519Sign(privateKey, message []byte) []byte {
	if len(privateKey) == ed25519.SeedSize {
		privateKey = ed25519.NewKeyFromSeed(privateKey)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil
	}
	return ed25519.Sign(ed25519.PrivateKey(privateKey), message)
}

// Ed25519Verify verifies an Ed25519 signature.
func Ed25519Verify(publicKey, message, signature []byte) bool {
	if len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(publicKey), message, signature)
}
