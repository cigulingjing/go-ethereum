package model

import "strings"

// AlgorithmInfo 描述动态算法在链上提交并由节点本地激活的元数据。
// 该类型只承载跨模块数据，升级状态和持久化流程由上层模块管理。
type AlgorithmInfo struct {
	Code  string `json:"code"`
	Gas   uint64 `json:"gas"`
	IType string `json:"itype"`
	OType string `json:"otype"`
}

// AlgorithmVersionInfo 描述链上提交的一个算法版本计划。
type AlgorithmVersionInfo struct {
	AlgorithmInfo
	Version         uint64 `json:"version"`
	ActivationBlock uint64 `json:"activationBlock"`
}

// Base returns the legacy metadata view used by existing single-version paths.
func (i AlgorithmVersionInfo) Base() AlgorithmInfo {
	return i.AlgorithmInfo
}

// Versioned wraps legacy metadata with explicit version semantics.
func Versioned(info AlgorithmInfo, version, activationBlock uint64) AlgorithmVersionInfo {
	return AlgorithmVersionInfo{
		AlgorithmInfo:   info,
		Version:         version,
		ActivationBlock: activationBlock,
	}
}

// LegacyVersion wraps metadata in the compatibility version used by uploadCode.
func LegacyVersion(info AlgorithmInfo) AlgorithmVersionInfo {
	return Versioned(info, 1, 0)
}

// InputTypes 返回算法输入参数的 Solidity ABI 类型列表。
func (i AlgorithmInfo) InputTypes() []string {
	return splitTypes(i.IType)
}

// OutputTypes 返回算法返回值的 Solidity ABI 类型列表。
func (i AlgorithmInfo) OutputTypes() []string {
	return splitTypes(i.OType)
}

// NormalizeAlgorithmName 保持 CodeStorage 已有的首字母大写语义。
func NormalizeAlgorithmName(name string) string {
	if name == "" {
		return name
	}
	bytes := []byte(name)
	if bytes[0] >= 'a' && bytes[0] <= 'z' {
		bytes[0] -= 'a' - 'A'
	}
	return string(bytes)
}

func splitTypes(types string) []string {
	if strings.TrimSpace(types) == "" {
		return nil
	}
	parts := strings.Split(types, ",")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}
