package cryptoupgrade

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"

	"github.com/ethereum/go-ethereum/cryptoupgrade/preload"
	"github.com/ethereum/go-ethereum/log"
)

// Compile source go file to .sp file, only approve run. Below one only approve debug
func compilePlugin(src string, outputPath string) error {
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-tags=urfave_cli_no_docs,ckzg", "-trimpath", "-o", outputPath, src)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// fmt.Printf("go version: %v\n", runtime.Version())
	return cmd.Run()
}

// ! There may be conflicts between the go version and the go compiler version, so If you choose a plugin that supports dbg, it can only support debugging but not normal execution.
func compileModulePlugin(src string, outputPath string) error {
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-gcflags=all=-N -l", "-o", outputPath, src)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

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

	p, ok := preload.IsPreLoad(algoName)
	if ok {
		return callPreloadAlgo(p, gas, encodedInput)
	} else {
		pluginPath := sofilePath(algoName)
		return callUpgradeAlgo(algoName, pluginPath, gas, encodedInput)
	}

}

func RequiredGas(algoName string) (uint64, error) {
	algoName = capitalString(algoName)

	if p, ok := preload.IsPreLoad(algoName); ok {
		return p.RequiredGas(), nil
	}
	funcInfo, ok := getAlgorithmInfo(algoName)
	if !ok {
		return 0, fmt.Errorf("algorithm %s is not loaded", algoName)
	}
	return funcInfo.gas, nil
}

func RequiredGasForCall(input []byte) (uint64, error) {
	algoName, _, err := ParseCall(input)
	if err != nil {
		return 0, err
	}
	return RequiredGas(algoName)
}

func RunCall(input []byte) ([]byte, error) {
	algoName, encodedInput, err := ParseCall(input)
	if err != nil {
		return nil, err
	}
	ret, _, err := CallProcessor(algoName, ^uint64(0), encodedInput)
	return ret, err
}

func callUpgradeAlgo(funcName string, pluginPath string, gas uint64, encodedInput []byte) ([]byte, uint64, error) {
	// Get algorithm info
	funcInfo, ok := getAlgorithmInfo(funcName)
	if !ok {
		return nil, gas, fmt.Errorf("algorithm %s is not loaded", funcName)
	}
	inputType, outputType := funcInfo.getTypeList()
	log.Info("Loaded upgrade algorithm ABI types", "input", inputType, "output", outputType)

	// Gas sufficient check
	if gas < funcInfo.gas {
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
	remainGas := gas - uint64(funcInfo.gas)
	log.Info("Successfully called upgrade algorithm", "gas", gas, "output", encodedOutput)
	return encodedOutput, remainGas, nil
}

func callPreloadAlgo(algo preload.PreLoadAlgorithm, gas uint64, encodedInput []byte) ([]byte, uint64, error) {
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
		log.Error("Failed to call preload algorithm", "err", err)
		return nil, gas, err
	}

	// Decode output
	encodedOutput, err := PackOutput(output, otype)
	if err != nil {
		log.Error("Failed to pack preload output", "err", err)
		return nil, gas, err
	}

	// Gas deduction
	remainGas := gas - algo.RequiredGas()
	log.Info("Successfully called preload algorithm", "gas", gas, "output", encodedOutput)
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
	defer func() {
		if r := recover(); r != nil {
			ret = nil
			err = fmt.Errorf("cryptoupgrade function panic: %v", r)
		}
	}()

	v := reflect.ValueOf(fn)

	// Chech whether fn is function type
	if v.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided value is not a function")
	}

	// Construct parameters to input
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	// Reflect Call
	result := v.Call(in)
	out := make([]interface{}, len(result))

	for i, r := range result {
		// Convert reflect.Value to interface{}
		out[i] = r.Interface()
	}
	return out, nil
}

// Go plugin compile
func PluginCompile(srcPath string, outputPath string) error {
	directoryInit()
	goBin := pluginGoBinary()
	if abs, err := filepath.Abs(srcPath); err == nil {
		srcPath = abs
	}
	if abs, err := filepath.Abs(outputPath); err == nil {
		outputPath = abs
	}
	cmd := exec.Command(goBin, "build", "-buildmode=plugin", "-tags=urfave_cli_no_docs,ckzg", "-trimpath", "-o", outputPath, srcPath)
	if buildDir := pluginBuildDir(); buildDir != "" {
		cmd.Dir = buildDir
	}
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
