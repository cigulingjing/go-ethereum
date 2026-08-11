package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ethereum/go-ethereum/cryptoupgrade/network"
)

func main() {
	var (
		mode        = flag.String("mode", "render", "mode: render, init, or validate")
		configPath  = flag.String("config", "cryptoupgrade/examples/networks/local-2nodes.yaml", "network YAML config path")
		outputDir   = flag.String("out", "", "render output directory")
		geth        = flag.String("geth", "geth", "geth binary used by init and generated scripts")
		outputJSON  = flag.String("output-json", "", "optional JSON result path")
		cryptoSmoke = flag.Bool("crypto-smoke", false, "run optional cryptoupgrade Add smoke test during validate")
		timeout     = flag.Duration("timeout", 30*time.Second, "validation timeout")
	)
	flag.Parse()

	cfg, err := network.LoadConfig(*configPath)
	if err != nil {
		fatalf("load config: %v", err)
	}
	var result any
	switch *mode {
	case "render":
		result, err = network.Render(cfg, network.RenderOptions{OutputDir: *outputDir, Geth: *geth})
	case "init":
		result, err = renderAndInit(cfg, *outputDir, *geth)
	case "validate":
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		result, err = network.ValidateNetwork(ctx, cfg, network.ValidateOptions{
			Timeout:            *timeout,
			CryptoUpgradeSmoke: *cryptoSmoke || cfg.CryptoUpgrade.SmokeTest,
		})
	default:
		err = fmt.Errorf("unsupported mode %q", *mode)
	}
	if err != nil {
		fatalf("%s failed: %v", *mode, err)
	}
	if err := writeJSON(result, *outputJSON); err != nil {
		fatalf("write result: %v", err)
	}
}

func renderAndInit(cfg *network.Config, outputDir, geth string) (*network.Artifacts, error) {
	artifacts, err := network.Render(cfg, network.RenderOptions{OutputDir: outputDir, Geth: geth})
	if err != nil {
		return nil, err
	}
	for _, node := range artifacts.Nodes {
		cmd := exec.Command(geth, "init", "--datadir", node.Datadir, artifacts.GenesisPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("init node %s: %w", node.ID, err)
		}
	}
	return artifacts, nil
}

func writeJSON(result any, outputPath string) error {
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if outputPath == "" {
		_, err = os.Stdout.Write(raw)
		return err
	}
	return os.WriteFile(outputPath, raw, 0644)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
