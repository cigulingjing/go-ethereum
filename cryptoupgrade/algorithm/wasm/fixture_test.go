package wasm

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tetratelabs/wazero"
)

func TestFixturesAreValidWASM(t *testing.T) {
	for _, name := range []string{
		"add.wasm",
		"sha256.wasm",
		"blake2b.wasm",
		"pbkdf2_sha256.wasm",
		"dh2048.wasm",
		"pedersen_commit.wasm",
		"schnorr_proof.wasm",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(".", name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			if len(data) < 4 || string(data[:4]) != "\x00asm" {
				t.Fatalf("%s has invalid wasm magic", name)
			}

			rt := wazero.NewRuntime(context.Background())
			defer rt.Close(context.Background())
			compiled, err := rt.CompileModule(context.Background(), data)
			if err != nil {
				t.Fatalf("compile %s: %v", name, err)
			}
			defer compiled.Close(context.Background())

			foundExecute := false
			for _, fn := range compiled.ExportedFunctions() {
				for _, exportName := range fn.ExportNames() {
					if exportName == "execute" {
						foundExecute = true
						break
					}
				}
				if foundExecute {
					break
				}
			}
			if !foundExecute {
				t.Fatalf("%s does not export execute", name)
			}
		})
	}
}
