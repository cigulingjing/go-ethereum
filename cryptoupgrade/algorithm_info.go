package cryptoupgrade

import (
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/repository"
)

type algoInfo = model.AlgorithmInfo

var runtimeAlgorithmRepository = repository.New(runtimePluginPaths.workspace())

// When geth exit, need to store @algoInfoMap
func Store() error {
	if runtimePluginPathsErr != nil {
		return runtimePluginPathsErr
	}
	if err := directoryInit(); err != nil {
		return err
	}
	return runtimeAlgorithmRepository.Save()
}

func getAlgorithmInfo(name string) (algoInfo, bool) {
	return runtimeAlgorithmRepository.Active(name)
}

func getUploadedAlgorithmInfo(name string) (algoInfo, bool) {
	return runtimeAlgorithmRepository.Uploaded(name)
}

func setAlgorithmInfo(name string, info algoInfo) {
	runtimeAlgorithmRepository.SetActive(name, info)
}

func setUploadedAlgorithmInfo(name string, info algoInfo) {
	runtimeAlgorithmRepository.SetUploaded(name, info)
}

func updateCodeStorageAlgorithmGas(name string, gas uint64) (bool, bool) {
	return runtimeAlgorithmRepository.UpdateGas(name, gas)
}

func storeAlgoMap(filename string) error {
	return runtimeAlgorithmRepository.SaveTo(filename)
}

func loadFromFile(filename string) error {
	return runtimeAlgorithmRepository.LoadFrom(filename)
}
