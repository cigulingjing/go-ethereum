package cryptoupgrade

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

var (
	algoInfoMap = make(map[string]algoInfo)
	algoInfoMu  sync.RWMutex
)

type algoInfo struct {
	code  string
	gas   uint64
	itype string
	otype string
}

type diskAlgoInfo struct {
	Code  string `json:"code"`
	Gas   uint64 `json:"gas"`
	IType string `json:"itype"`
	OType string `json:"otype"`
}

func (c algoInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(diskAlgoInfo{
		Code:  c.code,
		Gas:   c.gas,
		IType: c.itype,
		OType: c.otype,
	})
}

func (c *algoInfo) UnmarshalJSON(data []byte) error {
	var disk diskAlgoInfo
	if err := json.Unmarshal(data, &disk); err != nil {
		return err
	}
	c.code = disk.Code
	c.gas = disk.Gas
	c.itype = disk.IType
	c.otype = disk.OType
	return nil
}

func (c *algoInfo) getTypeList() ([]string, []string) {
	return splitTypeList(c.itype), splitTypeList(c.otype)
}

func splitTypeList(types string) []string {
	if strings.TrimSpace(types) == "" {
		return nil
	}
	parts := strings.Split(types, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// When geth exit, need to store @algoInfoMap
func Store() error {
	directoryInit()
	return storeAlgoMap(algoInfoPath)
}

func getAlgorithmInfo(name string) (algoInfo, bool) {
	algoInfoMu.RLock()
	defer algoInfoMu.RUnlock()

	info, ok := algoInfoMap[name]
	return info, ok
}

func setAlgorithmInfo(name string, info algoInfo) {
	algoInfoMu.Lock()
	defer algoInfoMu.Unlock()

	algoInfoMap[name] = info
}

func storeAlgoMap(filename string) error {
	algoInfoMu.RLock()
	defer algoInfoMu.RUnlock()

	data, err := json.MarshalIndent(algoInfoMap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func loadFromFile(filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// File does not exist yet, ignore this error.
			return nil
		}
		return err
	}

	algoInfoMu.Lock()
	defer algoInfoMu.Unlock()

	return json.Unmarshal(file, &algoInfoMap)
}
