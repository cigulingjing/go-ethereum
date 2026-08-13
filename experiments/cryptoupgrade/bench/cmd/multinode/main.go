// mutinode 负责为分布式环境提供配置，并不提供测试功能
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/network"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/smoke"
)

func main() {
	var (
		mode        = flag.String("mode", "render", "mode: render, init, validate, or smoke")
		configPath  = flag.String("config", "experiments/cryptoupgrade/deployments/networks/local-2nodes.yaml", "network YAML config path")
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
		networkResult, validateErr := network.ValidateNetwork(ctx, cfg, network.ValidateOptions{Timeout: *timeout})
		cancel()
		err = validateErr
		combined := combinedValidationResult{ValidationResult: networkResult}
		if err == nil && networkResult.OK && (*cryptoSmoke || cfg.CryptoUpgrade.SmokeTest) {
			smokeCtx, smokeCancel := context.WithTimeout(context.Background(), *timeout)
			combined.CryptoUpgrade = smoke.RunAdd(smokeCtx, cfg, smoke.Options{Timeout: *timeout})
			smokeCancel()
			if !combined.CryptoUpgrade.OK {
				networkResult.OK = false
			}
		}
		result = combined
	case "smoke":
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		result = smoke.RunAdd(ctx, cfg, smoke.Options{Timeout: *timeout})
		cancel()
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

type combinedValidationResult struct {
	*network.ValidationResult
	CryptoUpgrade *smoke.Result `json:"cryptoUpgrade,omitempty"`
}

func renderAndInit(cfg *network.Config, outputDir, geth string) (*network.Artifacts, error) {
	artifacts, err := network.Render(cfg, network.RenderOptions{OutputDir: outputDir, Geth: geth})
	if err != nil {
		return nil, err
	}
	// 完成geth初始化
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
