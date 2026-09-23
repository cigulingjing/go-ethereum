// 四个后量子验签：从 experiments/pqcc 抽出，共用 common/，一个入口。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	aigis "github.com/ethereum/go-ethereum/experiments/pqc_verify/aigis_sig2"
	"github.com/ethereum/go-ethereum/experiments/pqc_verify/dilithium3"
	mldsa65 "github.com/ethereum/go-ethereum/experiments/pqc_verify/ml_dsa_65"
	slhdsa "github.com/ethereum/go-ethereum/experiments/pqc_verify/slh_dsa_shake_192f"
)

type algo struct {
	name    string
	dir     string
	aliases []string
	verify  func(pk, message, signature []byte) (bool, error)
}

func algorithms() []algo {
	return []algo{
		{aigis.Name, "aigis_sig2", []string{"aigis", "aigis_sig2"}, aigis.Verify},
		{dilithium3.Name, "dilithium3", []string{"dilithium", "dilithium3"}, dilithium3.Verify},
		{mldsa65.Name, "ml_dsa_65", []string{"ml_dsa", "mldsa65", "ml_dsa_65"}, mldsa65.Verify},
		{slhdsa.Name, "slh_dsa_shake_192f", []string{"slh_dsa", "slhdsa", "slh_dsa_shake_192f"}, slhdsa.Verify},
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "pqc_verify: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	selected := algorithms()
	testdataOverride := ""
	if len(args) > 0 {
		a, ok := findAlgo(args[0])
		if !ok {
			return fmt.Errorf("unknown algorithm %q (aigis_sig2, dilithium3, ml_dsa_65, slh_dsa_shake_192f)", args[0])
		}
		selected = []algo{a}
	}
	if len(args) > 1 {
		testdataOverride = args[1]
	}
	root := moduleRoot()
	for _, a := range selected {
		dir := testdataOverride
		if dir == "" {
			dir = filepath.Join(root, a.dir, "testdata")
		}
		if err := verifyOne(a, dir); err != nil {
			return err
		}
	}
	return nil
}

func verifyOne(a algo, dir string) error {
	pk, err := os.ReadFile(filepath.Join(dir, "pk.bin"))
	if err != nil {
		return err
	}
	message, err := os.ReadFile(filepath.Join(dir, "message.bin"))
	if err != nil {
		return err
	}
	sig, err := os.ReadFile(filepath.Join(dir, "signature.bin"))
	if err != nil {
		return err
	}
	start := time.Now()
	ok, err := a.verify(pk, message, sig)
	if err != nil {
		return fmt.Errorf("%s: %w", a.name, err)
	}
	if !ok {
		return fmt.Errorf("%s: verify failed", a.name)
	}
	fmt.Printf("ok  algorithm=%s  testdata=%s  took=%s\n", a.name, dir, time.Since(start))
	return nil
}

func findAlgo(name string) (algo, bool) {
	for _, a := range algorithms() {
		if name == a.name || name == a.dir {
			return a, true
		}
		for _, alias := range a.aliases {
			if name == alias {
				return a, true
			}
		}
	}
	return algo{}, false
}

func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(file)
}
