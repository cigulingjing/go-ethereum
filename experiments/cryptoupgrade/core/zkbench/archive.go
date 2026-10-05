package zkbench

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench"
)

type Vectors struct {
	VK     []byte
	Proof  []byte
	Public []byte
}

type ArtifactSizes struct {
	NativeBytes        int64
	WASMBytes          int64
	NativeArchiveBytes int64
}

func AlgorithmDir(zkgoRoot, algorithm string) string {
	return filepath.Join(zkgoRoot, algorithm)
}

func WASMPath(zkgoRoot, algorithm string) string {
	return filepath.Join(zkgoRoot, algorithm, algorithm+".wasm")
}

func NativeDir(zkgoRoot, algorithm string) string {
	return filepath.Join(zkgoRoot, algorithm, "native")
}

func testdataDir(zkgoRoot, algorithm string) string {
	return filepath.Join(zkgoRoot, algorithm, "testdata")
}

func CheckArtifacts(zkgoRoot, algorithm string) error {
	native := NativeDir(zkgoRoot, algorithm)
	if err := requireSourceTree(native); err != nil {
		return fmt.Errorf("%w: %s: %v", pqcbench.ErrMissingArtifacts, algorithm, err)
	}
	wasm := WASMPath(zkgoRoot, algorithm)
	info, err := os.Stat(wasm)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return fmt.Errorf("%w: %s: missing wasm %s", pqcbench.ErrMissingArtifacts, algorithm, wasm)
	}
	if _, err := LoadVectors(zkgoRoot, algorithm); err != nil {
		return fmt.Errorf("%w: %s: %v", pqcbench.ErrMissingArtifacts, algorithm, err)
	}
	return nil
}

func LoadVectors(zkgoRoot, algorithm string) (Vectors, error) {
	dir := testdataDir(zkgoRoot, algorithm)
	vk, err := os.ReadFile(filepath.Join(dir, "vk.bin"))
	if err != nil {
		return Vectors{}, err
	}
	proof, err := os.ReadFile(filepath.Join(dir, "proof.bin"))
	if err != nil {
		return Vectors{}, err
	}
	pub, err := os.ReadFile(filepath.Join(dir, "public.bin"))
	if err != nil {
		return Vectors{}, err
	}
	if len(vk) == 0 || len(proof) == 0 || len(pub) == 0 {
		return Vectors{}, fmt.Errorf("empty testdata in %s", dir)
	}
	return Vectors{VK: vk, Proof: proof, Public: pub}, nil
}

func MeasureSizes(zkgoRoot, algorithm string) (ArtifactSizes, error) {
	var sizes ArtifactSizes
	native := NativeDir(zkgoRoot, algorithm)
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
		// 只计抽出的算法 .c（native/verify.c），不含 mcl/blst 封装库、宿主 check 与备份。
		if !isAuthoredCSource(native, path, name) {
			return nil
		}
		sizes.NativeBytes += info.Size()
		return nil
	})
	if err != nil {
		return ArtifactSizes{}, err
	}
	info, err := os.Stat(WASMPath(zkgoRoot, algorithm))
	if err != nil {
		return ArtifactSizes{}, err
	}
	sizes.WASMBytes = info.Size()
	return sizes, nil
}

func skipLibraryOrBackupDir(name string) bool {
	switch strings.ToLower(name) {
	case "src", "include", "vendor", "third_party", "lib", "backup", "backups":
		return true
	}
	return strings.HasPrefix(name, ".")
}

func isAuthoredCSource(native, path, name string) bool {
	if strings.ToLower(filepath.Ext(name)) != ".c" {
		return false
	}
	if isBackupName(name) {
		return false
	}
	if filepath.Dir(path) != native {
		return false
	}
	// check.c 只用于宿主核对 testdata，不进入 CGO/WASM 对照。
	return !strings.EqualFold(name, "check.c")
}

func isBackupName(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".bak") || strings.HasSuffix(lower, ".orig") || strings.HasSuffix(lower, "~") {
		return true
	}
	return strings.Contains(lower, ".bak.") || strings.HasPrefix(lower, "backup")
}

func requireSourceTree(native string) error {
	verify := filepath.Join(native, "verify.c")
	info, err := os.Stat(verify)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return fmt.Errorf("missing verify.c in %s", native)
	}
	blst := filepath.Join(native, "src", "server.c")
	mcl := filepath.Join(native, "src", "fp.cpp")
	if fileOK(blst) || fileOK(mcl) {
		return nil
	}
	return fmt.Errorf("missing pairing backend (src/server.c or src/fp.cpp) in %s", native)
}

func fileOK(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}
