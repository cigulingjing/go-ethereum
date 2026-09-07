package cryptoupgrade

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ethereum/go-ethereum/cryptoupgrade/builtin"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/compiler"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/pluginruntime"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/wasmruntime"
	"github.com/ethereum/go-ethereum/log"
)

// Convert str's first char to capital
func capitalString(str string) string {
	if str == "" {
		return str
	}
	bytes := []byte(str)
	if bytes[0] >= 'a' && bytes[0] <= 'z' {
		bytes[0] = bytes[0] - 32
	}
	return string(bytes)
}

func CallProcessor(algoName string, gas uint64, encodedInput []byte) ([]byte, uint64, error) {
	algoName = capitalString(algoName)

	log.Info("Call upgrade algorithm", "name", algoName, "input", encodedInput)

	p, ok := builtin.Lookup(algoName)
	if ok {
		return callBuiltinAlgorithm(p, gas, encodedInput)
	}
	return callUpgradeAlgo(algoName, gas, encodedInput)

}

func CallProcessorAt(algoName string, gas uint64, encodedInput []byte, blockNumber uint64) ([]byte, uint64, error) {
	algoName = capitalString(algoName)

	log.Info("Call upgrade algorithm", "name", algoName, "block", blockNumber, "input", encodedInput)

	p, ok := builtin.Lookup(algoName)
	if ok {
		return callBuiltinAlgorithm(p, gas, encodedInput)
	}
	funcInfo, ok := getActiveAlgorithmVersionInfoAt(algoName, blockNumber)
	if !ok {
		return nil, gas, fmt.Errorf("algorithm %s is not loaded", algoName)
	}
	prepared, ok := runtimeAlgorithmRepository.PreparedVersion(algoName, funcInfo.Version)
	if !ok {
		return nil, gas, requiredVersionMissingError(algoName, funcInfo.Version)
	}
	wasmPath, compiledPath := runtimeArtifactPaths(algoName, prepared.Version)
	return callUpgradeAlgoWithInfo(algoName, wasmPath, compiledPath, gas, encodedInput, prepared.Base())
}

func RequiredGas(algoName string) (uint64, error) {
	algoName = capitalString(algoName)

	if p, ok := builtin.Lookup(algoName); ok {
		return p.RequiredGas(), nil
	}
	funcInfo, ok := getActiveAlgorithmVersionInfo(algoName)
	if !ok {
		return 0, fmt.Errorf("algorithm %s is not loaded", algoName)
	}
	prepared, ok := runtimeAlgorithmRepository.PreparedVersion(algoName, funcInfo.Version)
	if !ok {
		return 0, requiredVersionMissingError(algoName, funcInfo.Version)
	}
	return prepared.Gas, nil
}

func RequiredGasAt(algoName string, blockNumber uint64) (uint64, error) {
	algoName = capitalString(algoName)

	if p, ok := builtin.Lookup(algoName); ok {
		return p.RequiredGas(), nil
	}
	funcInfo, ok := getActiveAlgorithmVersionInfoAt(algoName, blockNumber)
	if !ok {
		return 0, fmt.Errorf("algorithm %s is not loaded", algoName)
	}
	prepared, ok := runtimeAlgorithmRepository.PreparedVersion(algoName, funcInfo.Version)
	if !ok {
		return 0, requiredVersionMissingError(algoName, funcInfo.Version)
	}
	return prepared.Gas, nil
}

func RequiredGasForCall(input []byte) (uint64, error) {
	algoName, _, err := ParseCall(input)
	if err != nil {
		return 0, err
	}
	return RequiredGas(algoName)
}

func RequiredGasForCallAt(input []byte, blockNumber uint64) (uint64, error) {
	algoName, _, err := ParseCall(input)
	if err != nil {
		return 0, err
	}
	return RequiredGasAt(algoName, blockNumber)
}

func RunCall(input []byte) ([]byte, error) {
	algoName, encodedInput, err := ParseCall(input)
	if err != nil {
		return nil, err
	}
	ret, _, err := CallProcessor(algoName, ^uint64(0), encodedInput)
	return ret, err
}

func RunCallAt(input []byte, blockNumber uint64) ([]byte, error) {
	algoName, encodedInput, err := ParseCall(input)
	if err != nil {
		return nil, err
	}
	ret, _, err := CallProcessorAt(algoName, ^uint64(0), encodedInput, blockNumber)
	return ret, err
}

func callUpgradeAlgo(funcName string, gas uint64, encodedInput []byte) ([]byte, uint64, error) {
	funcInfo, ok := getActiveAlgorithmVersionInfo(funcName)
	if !ok {
		return nil, gas, fmt.Errorf("algorithm %s is not loaded", funcName)
	}
	prepared, ok := runtimeAlgorithmRepository.PreparedVersion(funcName, funcInfo.Version)
	if !ok {
		return nil, gas, requiredVersionMissingError(funcName, funcInfo.Version)
	}
	wasmPath, compiledPath := runtimeArtifactPaths(funcName, prepared.Version)
	return callUpgradeAlgoWithInfo(funcName, wasmPath, compiledPath, gas, encodedInput, prepared.Base())
}

func callUpgradeAlgoWithInfo(funcName string, wasmPath string, compiledPath string, gas uint64, encodedInput []byte, funcInfo algoInfo) ([]byte, uint64, error) {
	log.Info("Loaded upgrade algorithm", "name", funcName, "wasmPath", wasmPath, "compiledPath", compiledPath)
	// Gas sufficient check
	if gas < funcInfo.Gas {
		return nil, gas, errors.New("out of gas")
	}
	output, err := wasmruntime.Default.Execute(context.Background(), wasmPath, compiledPath, encodedInput)
	if err != nil {
		log.Error("Failed to call wasm algorithm", "name", funcName, "path", wasmPath, "err", err)
		return nil, gas, err
	}

	// Gas deduction and return output
	remainGas := gas - funcInfo.Gas
	log.Info("Successfully called upgrade algorithm", "gas", gas, "output", output)
	return output, remainGas, nil
}

func callBuiltinAlgorithm(algo builtin.Algorithm, gas uint64, encodedInput []byte) ([]byte, uint64, error) {
	// Gas sufficient check
	if gas < algo.RequiredGas() {
		return nil, gas, errors.New("out of gas")
	}

	// Get type list
	itype, otype := algo.GetTypeList()

	// Decode input
	input, err := UnpackInput(encodedInput, itype)
	if err != nil {
		log.Error("Failed to unpack callFunc input", "err", err)
		return nil, gas, err
	}

	// Call algorithm
	p := algo.TargetFunc()
	output, err := callFunction(p, input)
	if err != nil {
		log.Error("Failed to call builtin algorithm", "err", err)
		return nil, gas, err
	}

	// Decode output
	encodedOutput, err := PackOutput(output, otype)
	if err != nil {
		log.Error("Failed to pack builtin output", "err", err)
		return nil, gas, err
	}

	// Gas deduction
	remainGas := gas - algo.RequiredGas()
	log.Info("Successfully called builtin algorithm", "gas", gas, "output", encodedOutput)
	return encodedOutput, remainGas, err
}

// Call fun in plugin, parameter and return are mutable types
// Provide a unified calling entry. Return fn's return as an interface{} list
func callFunction(fn interface{}, args []interface{}) (ret []interface{}, err error) {
	return pluginruntime.CallFunction(fn, args)
}

// PluginCompile 保留旧 Go plugin 编译入口；WASM 激活路径不再调用它。
func PluginCompile(srcPath string, outputPath string) error {
	if err := directoryInit(); err != nil {
		return err
	}
	return compilePlugin(context.Background(), srcPath, outputPath)
}

func compilePlugin(ctx context.Context, srcPath, outputPath string) error {
	return compiler.Compile(ctx, srcPath, outputPath, compiler.BuildContext{
		GoBinary:   pluginGoBinary(),
		WorkingDir: pluginBuildDir(),
		BuildTags:  []string{"urfave_cli_no_docs", "ckzg"},
		Trimpath:   true,
		CGOEnabled: true,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	})
}

func runtimeArtifactPaths(algoName string, version uint64) (string, string) {
	if version == 1 {
		return runtimeAlgorithmRepository.SourcePath(algoName), runtimeAlgorithmRepository.PluginPath(algoName)
	}
	return runtimeAlgorithmRepository.VersionSourcePath(algoName, version), runtimeAlgorithmRepository.VersionPluginPath(algoName, version)
}

func requiredVersionMissingError(algoName string, version uint64) error {
	if version == 1 {
		return fmt.Errorf("%w: algorithm %s is not loaded", wasmruntime.ErrRequiredVersionMissing, algoName)
	}
	return fmt.Errorf("%w: algorithm %s version %d is not loaded", wasmruntime.ErrRequiredVersionMissing, algoName, version)
}

func pluginGoBinary() string {
	if goBin := os.Getenv("CRYPTOUPGRADE_GO"); goBin != "" {
		return goBin
	}
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
	if _, err := os.Stat(goBin); err == nil {
		return goBin
	}
	return "go"
}

func pluginBuildDir() string {
	buildDir := os.Getenv("CRYPTOUPGRADE_MODULE")
	if buildDir == "" {
		return ""
	}
	if _, err := os.Stat(filepath.Join(buildDir, "go.mod")); err != nil {
		return ""
	}
	return buildDir
}
