package pqcbench

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Vectors 是归档 testdata 中的一组验签输入。
type Vectors struct {
	PublicKey []byte
	Message   []byte
	Signature []byte
}

// ArtifactSizes 只统计归档目录，不读临时构建产物。
type ArtifactSizes struct {
	NativeBytes        int64
	WASMBytes          int64
	NativeArchiveBytes int64
}

func AlgorithmDir(pqcgoRoot, algorithm string) string {
	return filepath.Join(pqcgoRoot, algorithm)
}

func WASMPath(pqcgoRoot, algorithm string) string {
	return filepath.Join(pqcgoRoot, algorithm, algorithm+".wasm")
}

func NativeDir(pqcgoRoot, algorithm string) string {
	return filepath.Join(pqcgoRoot, algorithm, "native")
}

func testdataDir(pqcgoRoot, algorithm string) string {
	return filepath.Join(pqcgoRoot, algorithm, "testdata")
}

// CheckArtifacts 检查 native/、.wasm 与 testdata 是否齐全。
func CheckArtifacts(pqcgoRoot, algorithm string) error {
	native := NativeDir(pqcgoRoot, algorithm)
	if err := requireSourceTree(native); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrMissingArtifacts, algorithm, err)
	}
	wasm := WASMPath(pqcgoRoot, algorithm)
	info, err := os.Stat(wasm)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return fmt.Errorf("%w: %s: missing wasm %s", ErrMissingArtifacts, algorithm, wasm)
	}
	if _, err := LoadVectors(pqcgoRoot, algorithm); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrMissingArtifacts, algorithm, err)
	}
	return nil
}

func LoadVectors(pqcgoRoot, algorithm string) (Vectors, error) {
	dir := testdataDir(pqcgoRoot, algorithm)
	pk, err := os.ReadFile(filepath.Join(dir, "pk.bin"))
	if err != nil {
		return Vectors{}, err
	}
	message, err := os.ReadFile(filepath.Join(dir, "message.bin"))
	if err != nil {
		return Vectors{}, err
	}
	signature, err := os.ReadFile(filepath.Join(dir, "signature.bin"))
	if err != nil {
		return Vectors{}, err
	}
	if len(pk) == 0 || len(message) == 0 || len(signature) == 0 {
		return Vectors{}, fmt.Errorf("empty testdata in %s", dir)
	}
	return Vectors{PublicKey: pk, Message: message, Signature: signature}, nil
}

func MeasureSizes(pqcgoRoot, algorithm string) (ArtifactSizes, error) {
	var sizes ArtifactSizes
	native := NativeDir(pqcgoRoot, algorithm)
	err := filepath.WalkDir(native, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipLibraryOrBackupDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		name := d.Name()
		if name == "libverify.a" {
			sizes.NativeArchiveBytes = info.Size()
			return nil
		}
		// 只计抽出的算法 .c（native/verify.c），不含 PQMagic 封装库、头文件与备份。
		if !isAuthoredCSource(native, path, name) {
			return nil
		}
		sizes.NativeBytes += info.Size()
		return nil
	})
	if err != nil {
		return ArtifactSizes{}, err
	}
	info, err := os.Stat(WASMPath(pqcgoRoot, algorithm))
	if err != nil {
		return ArtifactSizes{}, err
	}
	sizes.WASMBytes = info.Size()
	return sizes, nil
}

func skipLibraryOrBackupDir(name string) bool {
	switch strings.ToLower(name) {
	case "src", "include", "vendor", "third_party", "lib", "hash", "utils", "backup", "backups":
		return true
	}
	return strings.HasPrefix(name, ".")
}

func isAuthoredCSource(native, path, name string) bool {
	if !strings.EqualFold(name, "verify.c") {
		return false
	}
	if isBackupName(name) {
		return false
	}
	return filepath.Dir(path) == native
}

func isBackupName(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".bak") || strings.HasSuffix(lower, ".orig") || strings.HasSuffix(lower, "~") {
		return true
	}
	return strings.Contains(lower, ".bak.") || strings.HasPrefix(lower, "backup")
}

func requireSourceTree(native string) error {
	entries, err := os.ReadDir(native)
	if err != nil {
		return err
	}
	hasC := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".c") {
			hasC = true
			break
		}
	}
	if !hasC {
		return fmt.Errorf("no C sources in %s", native)
	}
	return nil
}
