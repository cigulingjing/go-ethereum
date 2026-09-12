// render 根据 YAML 配置生成私有链部署目录（genesis、节点目录、compose 等）。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/network"
)


func main() {
	configPath := flag.String("config", "", "network YAML path")
	outDir := flag.String("out", "", "output directory override")
	geth := flag.String("geth", "geth", "geth binary for start scripts")
	jsonOut := flag.String("output-json", "", "optional path to write artifacts JSON")
	flag.Parse()
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "usage: render -config <network.yaml> [-out <dir>] [-geth <path>]")
		os.Exit(2)
	}
	cfg, err := network.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	opts := network.RenderOptions{Geth: *geth}
	if *outDir != "" {
		opts.OutputDir = *outDir
	}
	artifacts, err := network.Render(cfg, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *jsonOut != "" {
		raw, err := json.MarshalIndent(artifacts, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.MkdirAll(filepath.Dir(*jsonOut), 0755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(*jsonOut, append(raw, '\n'), 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Println(artifacts.OutputDir)
}
