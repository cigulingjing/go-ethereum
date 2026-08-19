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
	} else {
		pluginPath := sofilePath(algoName)
		return callUpgradeAlgo(algoName, pluginPath, gas, encodedInput)
	}

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
	if _, ok := getUploadedAlgorithmVersionInfo(algoName, funcInfo.Version); ok {
		if prepared, ok := runtimeAlgorithmRepository.PreparedVersion(algoName, funcInfo.Version); ok {
			funcInfo = prepared
		} else {
			if funcInfo.Version == 1 {
				return nil, gas, fmt.Errorf("algorithm %s is not loaded", algoName)
			}
			return nil, gas, fmt.Errorf("algorithm %s version %d is not loaded", algoName, funcInfo.Version)
		}
	}
	pluginPath := sofilePath(algoName)
	if funcInfo.Version != 1 {
		pluginPath = runtimeAlgorithmRepository.VersionPluginPath(algoName, funcInfo.Version)
	}
	return callUpgradeAlgoWithInfo(algoName, pluginPath, gas, encodedInput, funcInfo.Base())
}

func RequiredGas(algoName string) (uint64, error) {
	algoName = capitalString(algoName)

	if p, ok := builtin.Lookup(algoName); ok {
		return p.RequiredGas(), nil
	}
	funcInfo, ok := getAlgorithmInfo(algoName)
	if !ok {
		return 0, fmt.Errorf("algorithm %s is not loaded", algoName)
	}
	return funcInfo.Gas, nil
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
	if _, ok := getUploadedAlgorithmVersionInfo(algoName, funcInfo.Version); ok {
		if prepared, ok := runtimeAlgorithmRepository.PreparedVersion(algoName, funcInfo.Version); ok {
			return prepared.Gas, nil
		}
		if funcInfo.Version == 1 {
			return 0, fmt.Errorf("algorithm %s is not loaded", algoName)
		}
		return 0, fmt.Errorf("algorithm %s version %d is not loaded", algoName, funcInfo.Version)
	}
	return funcInfo.Gas, nil
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

func callUpgradeAlgo(funcName string, pluginPath string, gas uint64, encodedInput []byte) ([]byte, uint64, error) {
	// Get algorithm info
	funcInfo, ok := getAlgorithmInfo(funcName)
	if !ok {
		return nil, gas, fmt.Errorf("algorithm %s is not loaded", funcName)
	}
	return callUpgradeAlgoWithInfo(funcName, pluginPath, gas, encodedInput, funcInfo)
}

func callUpgradeAlgoWithInfo(funcName string, pluginPath string, gas uint64, encodedInput []byte, funcInfo algoInfo) ([]byte, uint64, error) {
	inputType, outputType := funcInfo.InputTypes(), funcInfo.OutputTypes()
	log.Info("Loaded upgrade algorithm ABI types", "input", inputType, "output", outputType)

	// Gas sufficient check
	if gas < funcInfo.Gas {
		return nil, gas, errors.New("out of gas")
	}

	// Decode input
	input, err := UnpackInput(encodedInput, inputType)
	if err != nil {
		log.Error("Failed to unpack callFunc input", "err", err)
		// fmt.Println("Unpack input from callFunc error:", err)
		return nil, gas, err
	}

	// Call algorithm. Attention 1.[]interface{} and ...interface{} 2.Algorithm name must be capital
	output, err := callPlugin(pluginPath, funcName, input)
	if err != nil {
		log.Error("Failed to call plugin", "err", err)
		// fmt.Println("Call Plugin error", err)
		return nil, gas, err
	}

	// Output encode
	encodedOutput, err := PackOutput(output, outputType)
	if err != nil {
		log.Error("Failed to pack plugin output", "err", err)
		// fmt.Println("Pack output error:", err)
		return nil, gas, err
	}

	// Gas deduction and return output
	remainGas := gas - funcInfo.Gas
	log.Info("Successfully called upgrade algorithm", "gas", gas, "output", encodedOutput)
	return encodedOutput, remainGas, nil
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
func callPlugin(pluginPath string, funName string, args []interface{}) ([]interface{}, error) {
	fn, err := lookupPluginFunction(pluginPath, funName)
	if err != nil {
		log.Error("Failed to load plugin function", "name", funName, "path", pluginPath, "err", err)
		return nil, err
	}

	// Call func and return
	ReturnList, err := callFunction(fn, args)
	if err != nil {
		log.Error("Failed to call plugin function", "name", funName, "path", pluginPath, "err", err)
		return nil, err
	} else {
		return ReturnList, nil
	}
}

// Provide a unified calling entry. Return fn's return as an interface{} list
func callFunction(fn interface{}, args []interface{}) (ret []interface{}, err error) {
	return pluginruntime.CallFunction(fn, args)
}

// Go plugin compile
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
