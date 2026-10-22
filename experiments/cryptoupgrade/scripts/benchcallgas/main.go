package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/ethereum/go-ethereum/cryptoupgrade/pricing"
)

const (
	blockGasLimit = uint64(30_000_000)
	threshold     = uint64(2)
	callCount     = uint64(5)
)

var algorithmNames = []string{
	"Add", "Sha256", "Blake2bSum256", "Pbkdf2Sha256",
	"Dh2048Secret", "PedersenCommit", "SchnorrVerify", "PolynomialMul",
}

type calibration struct {
	AnchorGas  uint64 `json:"anchorGas"`
	Algorithms map[string]struct {
		Metric           string `json:"metric"`
		ReferenceValue   uint64 `json:"referenceValue"`
		ReferenceBaseGas uint64 `json:"referenceBaseGas"`
		WasmSHA256       string `json:"wasmSha256"`
		Measurements     []struct {
			Value              uint64 `json:"value"`
			RawProportionalGas uint64 `json:"rawProportionalGas"`
		} `json:"measurements"`
	} `json:"algorithms"`
}

type sample struct {
	Algorithm           string   `json:"algorithm"`
	Metric              string   `json:"metric"`
	ReferenceValue      uint64   `json:"referenceValue"`
	BaseGas             uint64   `json:"baseGas"`
	CallGas             []uint64 `json:"callGas"`
	CumulativeGas       []uint64 `json:"cumulativeGas"`
	TotalFiveCallGas    uint64   `json:"totalFiveCallGas"`
	MaxCallsWithinLimit int      `json:"optimisticMaxCallsWithinLimit"`
}

type scenario struct {
	ParentCalls  uint64        `json:"parentCalls"`
	Multiplier   pricing.Ratio `json:"multiplier"`
	Samples      []sample      `json:"samples"`
	MixedOnceGas uint64        `json:"mixedOnceGas"`
}

type output struct {
	Kind              string             `json:"kind"`
	CalibrationSHA256 string             `json:"calibrationSHA256"`
	Threshold         uint64             `json:"threshold"`
	Alpha             pricing.Ratio      `json:"alpha"`
	LoadMapping       []pricing.LoadBand `json:"loadMapping"`
	GasLimit          uint64             `json:"blockGasLimit"`
	ReferenceCalls    uint64             `json:"callsPerAlgorithm"`
	Scenarios         []scenario         `json:"scenarios"`
}

func loadCalibration(path string) (calibration, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return calibration{}, "", err
	}
	var source calibration
	if err := json.Unmarshal(data, &source); err != nil {
		return calibration{}, "", err
	}
	if len(source.Algorithms) != len(algorithmNames) {
		return calibration{}, "", errors.New("expected exactly eight calibrated algorithms")
	}
	for _, name := range algorithmNames {
		entry, ok := source.Algorithms[name]
		if !ok || entry.Metric == "" || entry.ReferenceValue == 0 || entry.ReferenceBaseGas == 0 || entry.WasmSHA256 == "" {
			return calibration{}, "", fmt.Errorf("missing calibration for %s", name)
		}
	}
	return source, fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func calculate(calibrationPath string) (output, error) {
	return calculateForInput(calibrationPath, false)
}

func calculateForInput(calibrationPath string, blake32 bool) (output, error) {
	source, digest, err := loadCalibration(calibrationPath)
	if err != nil {
		return output{}, err
	}
	if blake32 {
		entry := source.Algorithms["Blake2bSum256"]
		entry.ReferenceBaseGas = 0
		for _, point := range entry.Measurements {
			if point.Value == 32 {
				entry.ReferenceBaseGas = point.RawProportionalGas
			}
		}
		if entry.ReferenceBaseGas == 0 {
			return output{}, errors.New("missing calibrated Blake2bSum256 32-byte level")
		}
		entry.ReferenceValue = 32
		source.Algorithms["Blake2bSum256"] = entry
	}
	load := []pricing.LoadBand{
		{MaxCalls: 0, Multiplier: pricing.Ratio{Numerator: 1, Denominator: 1}},
		{MaxCalls: 1, Multiplier: pricing.Ratio{Numerator: 11, Denominator: 10}},
		{MaxCalls: 2, Multiplier: pricing.Ratio{Numerator: 6, Denominator: 5}},
		{MaxCalls: 3, Multiplier: pricing.Ratio{Numerator: 13, Denominator: 10}},
		{MaxCalls: math.MaxUint64, Multiplier: pricing.Ratio{Numerator: 7, Denominator: 5}},
	}
	profile := pricing.Profile{Threshold: threshold, Alpha: pricing.Ratio{Numerator: 1, Denominator: 4}, Load: load}
	for _, name := range algorithmNames {
		entry := source.Algorithms[name]
		profile.Algorithms = append(profile.Algorithms, pricing.Algorithm{
			Name: name, Version: 1, Metric: entry.Metric,
			Bands: []pricing.InputBand{{MaxMetric: math.MaxUint64, Gas: entry.ReferenceBaseGas}},
		})
	}
	if err := profile.Validate(); err != nil {
		return output{}, err
	}
	result := output{Kind: "pricing-model-only; no EVM execution or transaction receipts", CalibrationSHA256: digest, Threshold: threshold, Alpha: profile.Alpha, LoadMapping: load, GasLimit: blockGasLimit, ReferenceCalls: callCount}
	for parentCalls := uint64(0); parentCalls <= 4; parentCalls++ {
		caseResult := scenario{ParentCalls: parentCalls, Multiplier: profile.Multiplier(parentCalls)}
		for _, name := range algorithmNames {
			entry := source.Algorithms[name]
			item := sample{Algorithm: name, Metric: entry.Metric, ReferenceValue: entry.ReferenceValue, BaseGas: entry.ReferenceBaseGas}
			for index := uint64(1); index <= callCount; index++ {
				gas, err := profile.Quote(name, 1, entry.ReferenceValue, index, parentCalls)
				if err != nil {
					return output{}, err
				}
				if item.TotalFiveCallGas > math.MaxUint64-gas {
					return output{}, fmt.Errorf("cumulative gas overflows for %s", name)
				}
				item.CallGas = append(item.CallGas, gas)
				item.TotalFiveCallGas += gas
				item.CumulativeGas = append(item.CumulativeGas, item.TotalFiveCallGas)
				if item.TotalFiveCallGas <= blockGasLimit-21_000 {
					item.MaxCallsWithinLimit = int(index)
				}
				if index == 1 {
					if caseResult.MixedOnceGas > math.MaxUint64-gas {
						return output{}, errors.New("mixed gas overflows")
					}
					caseResult.MixedOnceGas += gas
				}
			}
			caseResult.Samples = append(caseResult.Samples, item)
		}
		result.Scenarios = append(result.Scenarios, caseResult)
	}
	return result, nil
}

func main() {
	calibrationPath := flag.String("calibration", "experiments/cryptoupgrade/results/gas-complexity/2026-10-22-calibration.json", "calibration JSON path")
	blake32 := flag.Bool("blake32", false, "use calibrated 32-byte Blake2b level supported by the Solidity contract")
	resultPath := flag.String("out", "", "JSON result path, or stdout")
	flag.Parse()
	result, err := calculateForInput(*calibrationPath, *blake32)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if *resultPath == "" {
		_, err = os.Stdout.Write(data)
	} else {
		err = os.WriteFile(*resultPath, data, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
