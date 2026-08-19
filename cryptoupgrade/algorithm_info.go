package cryptoupgrade

import (
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/repository"
)

type algoInfo = model.AlgorithmInfo
type algoVersionInfo = model.AlgorithmVersionInfo

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

func getUploadedAlgorithmVersionInfo(name string, version uint64) (algoVersionInfo, bool) {
	return runtimeAlgorithmRepository.UploadedVersion(name, version)
}

func getActiveAlgorithmVersionInfo(name string) (algoVersionInfo, bool) {
	return runtimeAlgorithmRepository.ActiveVersion(name)
}

func getActiveAlgorithmVersionInfoAt(name string, blockNumber uint64) (algoVersionInfo, bool) {
	return runtimeAlgorithmRepository.ActiveVersionAt(name, blockNumber)
}

func setAlgorithmInfo(name string, info algoInfo) {
	runtimeAlgorithmRepository.SetActive(name, info)
}

func setUploadedAlgorithmInfo(name string, info algoInfo) {
	runtimeAlgorithmRepository.SetUploaded(name, info)
}

func setAlgorithmVersionInfo(name string, info algoVersionInfo) {
	runtimeAlgorithmRepository.SetActiveVersion(name, info)
}

func setUploadedAlgorithmVersionInfo(name string, info algoVersionInfo) {
	runtimeAlgorithmRepository.SetUploadedVersion(name, info)
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
