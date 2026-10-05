package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
)

func isInlinePrivateKey(value string) bool {
	value = strings.TrimSpace(strings.TrimPrefix(value, "0x"))
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func normalizePrivateKey(cfg *Config, value string) string {
	value = strings.TrimSpace(value)
	if value == "" || isInlinePrivateKey(value) {
		return value
	}
	return cfg.resolvePath(value)
}

func loadPrivateKeyHex(value string) (string, error) {
	value = strings.TrimSpace(value)
	if isInlinePrivateKey(value) {
		return strings.TrimPrefix(value, "0x"), nil
	}
	raw, err := os.ReadFile(value)
	if err != nil {
		return "", fmt.Errorf("read private key %s: %w", value, err)
	}
	hexKey := strings.TrimSpace(strings.TrimPrefix(string(raw), "0x"))
	if !isInlinePrivateKey(hexKey) {
		return "", fmt.Errorf("private key %s is not a valid 32-byte hex secret", value)
	}
	return hexKey, nil
}

// importSignerKeystore 从 YAML 引用的私钥生成 geth 可解锁的 keystore。
func importSignerKeystore(privateKey, password, datadir string) error {
	hexKey, err := loadPrivateKeyHex(privateKey)
	if err != nil {
		return err
	}
	key, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	ksDir := filepath.Join(datadir, "keystore")
	if err := os.MkdirAll(ksDir, 0700); err != nil {
		return err
	}
	if entries, err := os.ReadDir(ksDir); err == nil && len(entries) > 0 {
		return nil
	}
	ks := keystore.NewKeyStore(ksDir, keystore.StandardScryptN, keystore.StandardScryptP)
	if _, err := ks.ImportECDSA(key, password); err != nil {
		return fmt.Errorf("import signer keystore: %w", err)
	}
	return nil
}
