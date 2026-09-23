// Groth16 BLS12-381 Verify：从 gnark 抽出的配对验证核，独立可执行。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "groth16_bls12381_verify: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dir := testdataDir()
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	proof, vk, public, err := loadArchived(dir)
	if err != nil {
		return fmt.Errorf("load testdata %s: %w", dir, err)
	}
	start := time.Now()
	if err := Verify(proof, vk, public); err != nil {
		return err
	}
	fmt.Printf("ok  curve=bls12-381  testdata=%s  took=%s\n", dir, time.Since(start))
	return nil
}

func testdataDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "testdata"
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}
