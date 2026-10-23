// Package pricing contains deterministic, opt-in algorithm gas calculations.
package pricing

import (
	"errors"
	"fmt"
	"math"
	"math/big"
)

var ErrGasOverflow = errors.New("algorithm gas exceeds uint64")

const maxAlphaDenominator = 10000

type Ratio struct {
	Numerator   uint64 `json:"numerator" yaml:"numerator"`
	Denominator uint64 `json:"denominator" yaml:"denominator"`
}

type InputBand struct {
	MaxMetric uint64 `json:"maxMetric" yaml:"maxMetric"`
	Gas       uint64 `json:"gas" yaml:"gas"`
}

type Algorithm struct {
	Name    string      `json:"name" yaml:"name"`
	Version uint64      `json:"version" yaml:"version"`
	Metric  string      `json:"metric" yaml:"metric"`
	Bands   []InputBand `json:"bands" yaml:"bands"`
}

type LoadBand struct {
	MaxCalls   uint64 `json:"maxCalls" yaml:"maxCalls"`
	Multiplier Ratio  `json:"multiplier" yaml:"multiplier"`
}

// Profile 仅供旧实验计算及识别已弃用的链配置；运行时使用版本独立的 Schedule。
type Profile struct {
	Threshold  uint64      `json:"threshold" yaml:"threshold"`
	Alpha      Ratio       `json:"alpha" yaml:"alpha"`
	Algorithms []Algorithm `json:"algorithms" yaml:"algorithms"`
	Load       []LoadBand  `json:"load" yaml:"load"`
}

func (p Profile) Validate() error {
	if p.Threshold == 0 || p.Alpha.Denominator == 0 || p.Alpha.Denominator > maxAlphaDenominator {
		return errors.New("invalid threshold or alpha denominator")
	}
	if len(p.Algorithms) == 0 || len(p.Load) == 0 {
		return errors.New("algorithm and load bands are required")
	}
	seen := make(map[string]map[uint64]bool)
	for _, algorithm := range p.Algorithms {
		if algorithm.Name == "" || algorithm.Metric == "" || len(algorithm.Bands) == 0 {
			return errors.New("algorithm name, metric and input bands are required")
		}
		if seen[algorithm.Name] == nil {
			seen[algorithm.Name] = make(map[uint64]bool)
		}
		if seen[algorithm.Name][algorithm.Version] {
			return fmt.Errorf("duplicate algorithm profile %s version %d", algorithm.Name, algorithm.Version)
		}
		seen[algorithm.Name][algorithm.Version] = true
		var last uint64
		for index, band := range algorithm.Bands {
			if band.Gas == 0 || (index > 0 && band.MaxMetric <= last) {
				return fmt.Errorf("invalid input band for %s", algorithm.Name)
			}
			last = band.MaxMetric
		}
		if last != math.MaxUint64 {
			return fmt.Errorf("input bands for %s must cover every size", algorithm.Name)
		}
	}
	var last uint64
	for index, band := range p.Load {
		if (index > 0 && band.MaxCalls <= last) || band.Multiplier.Denominator == 0 || band.Multiplier.Numerator < band.Multiplier.Denominator {
			return errors.New("invalid load band")
		}
		if index == 0 && band.Multiplier.Numerator != band.Multiplier.Denominator {
			return errors.New("load mapping must begin with M(0)=1")
		}
		if index > 0 && compare(band.Multiplier, p.Load[index-1].Multiplier) < 0 {
			return errors.New("load multipliers must be nondecreasing")
		}
		last = band.MaxCalls
	}
	if last != math.MaxUint64 {
		return errors.New("load bands must cover every count")
	}
	return nil
}

func compare(left, right Ratio) int {
	first := new(big.Int).Mul(new(big.Int).SetUint64(left.Numerator), new(big.Int).SetUint64(right.Denominator))
	second := new(big.Int).Mul(new(big.Int).SetUint64(right.Numerator), new(big.Int).SetUint64(left.Denominator))
	return first.Cmp(second)
}

func (p Profile) Base(name string, version, metricValue uint64) (uint64, error) {
	for _, algorithm := range p.Algorithms {
		if algorithm.Name == name && algorithm.Version == version {
			for _, band := range algorithm.Bands {
				if metricValue <= band.MaxMetric {
					return band.Gas, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("missing gas profile for %s version %d", name, version)
}

func (p Profile) Multiplier(parentCalls uint64) Ratio {
	for _, band := range p.Load {
		if parentCalls <= band.MaxCalls {
			return band.Multiplier
		}
	}
	return Ratio{}
}

func (p Profile) Quote(name string, version, metricValue, index, parentCalls uint64) (uint64, error) {
	if index == 0 {
		return 0, errors.New("call index starts at one")
	}
	base, err := p.Base(name, version, metricValue)
	if err != nil {
		return 0, err
	}
	multiplier := p.Multiplier(parentCalls)
	if multiplier.Denominator == 0 || p.Alpha.Denominator == 0 || p.Alpha.Denominator > maxAlphaDenominator {
		return 0, errors.New("invalid gas profile")
	}
	return quote(base, index, p.Threshold, p.Alpha, multiplier)
}

func quote(base, index, threshold uint64, alpha, multiplier Ratio) (uint64, error) {
	numerator := new(big.Int).Mul(new(big.Int).SetUint64(base), new(big.Int).SetUint64(multiplier.Numerator))
	denominator := new(big.Int).SetUint64(multiplier.Denominator)
	if index > threshold && alpha.Numerator > 0 {
		exponent := index - threshold
		factorNumerator := new(big.Int).SetUint64(alpha.Denominator)
		factorNumerator.Add(factorNumerator, new(big.Int).SetUint64(alpha.Numerator))
		factorDenominator := new(big.Int).SetUint64(alpha.Denominator)
		period := alpha.Denominator / alpha.Numerator
		if alpha.Denominator%alpha.Numerator != 0 {
			period++
		}
		if exponent/period >= 64 {
			return 0, ErrGasOverflow
		}
		power := new(big.Int).SetUint64(exponent)
		numerator.Mul(numerator, new(big.Int).Exp(factorNumerator, power, nil))
		denominator.Mul(denominator, new(big.Int).Exp(factorDenominator, power, nil))
	}
	quotient, remainder := new(big.Int).QuoRem(numerator, denominator, new(big.Int))
	if remainder.Sign() != 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsUint64() {
		return 0, ErrGasOverflow
	}
	return quotient.Uint64(), nil
}
