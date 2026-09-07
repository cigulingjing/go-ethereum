package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cryptoupgrade/wasmtool"
)

func main() {
	var (
		source   = flag.String("source", "", "Go algorithm source file")
		output   = flag.String("out", "", "output wasm file")
		function = flag.String("function", "", "algorithm function name")
		itype    = flag.String("itype", "", "comma-separated input ABI types")
		otype    = flag.String("otype", "", "comma-separated output ABI types")
		timeout  = flag.Duration("timeout", 2*time.Minute, "TinyGo build timeout")
	)
	flag.Parse()

	if *source == "" || *output == "" || *function == "" {
		fmt.Fprintln(os.Stderr, "-source, -out and -function are required")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	wasm, err := wasmtool.BuildPath(ctx, *source, wasmtool.Spec{
		Function:    *function,
		InputTypes:  splitTypes(*itype),
		OutputTypes: splitTypes(*otype),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "build wasm: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, wasm, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write wasm: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s <- %s (%s -> %s)\n", *output, *source, *itype, *otype)
}

func splitTypes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if typ := strings.TrimSpace(part); typ != "" {
			out = append(out, typ)
		}
	}
	return out
}
