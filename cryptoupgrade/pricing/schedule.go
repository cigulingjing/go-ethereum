package pricing

import (
	"bytes"
	"errors"
	"math"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// MaxBands 限制提案解码及每次调用的定价开销。
const MaxBands = 64

// Schedule 随算法版本发布；输入指标固定为 ABI 编码参数的字节数。
type Schedule struct {
	Threshold uint64      `json:"threshold"`
	Alpha     Ratio       `json:"alpha"`
	Bands     []InputBand `json:"bands"`
}

func (s Schedule) Validate() error {
	if s.Threshold == 0 || s.Alpha.Denominator == 0 || s.Alpha.Denominator > maxAlphaDenominator {
		return errors.New("invalid pricing threshold or alpha denominator")
	}
	if len(s.Bands) == 0 || len(s.Bands) > MaxBands {
		return errors.New("pricing requires 1 to 64 input bands")
	}
	var last uint64
	for i, band := range s.Bands {
		if band.Gas == 0 || (i > 0 && (band.MaxMetric <= last || band.Gas < s.Bands[i-1].Gas)) {
			return errors.New("pricing bands must have increasing bounds and positive nondecreasing gas")
		}
		last = band.MaxMetric
	}
	if last != math.MaxUint64 {
		return errors.New("pricing bands must cover every input size")
	}
	return nil
}

func (s Schedule) Quote(inputBytes, index uint64) (uint64, error) {
	if err := s.Validate(); err != nil {
		return 0, err
	}
	if index == 0 {
		return 0, errors.New("call index starts at one")
	}
	for _, band := range s.Bands {
		if inputBytes <= band.MaxMetric {
			return quote(band.Gas, index, s.Threshold, s.Alpha, Ratio{Numerator: 1, Denominator: 1})
		}
	}
	return 0, errors.New("missing input band")
}

var scheduleArguments = func() abi.Arguments {
	var args abi.Arguments
	for _, name := range []string{"uint64", "uint64", "uint64", "uint64[]", "uint64[]"} {
		typ, err := abi.NewType(name, "", nil)
		if err != nil {
			panic(err)
		}
		args = append(args, abi.Argument{Type: typ})
	}
	return args
}()

// Encode 使用 Solidity 可直接 abi.decode 的规范格式，避免 JSON 表达差异。
func (s Schedule) Encode() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	limits, gas := make([]uint64, len(s.Bands)), make([]uint64, len(s.Bands))
	for i, band := range s.Bands {
		limits[i], gas[i] = band.MaxMetric, band.Gas
	}
	return scheduleArguments.Pack(s.Threshold, s.Alpha.Numerator, s.Alpha.Denominator, limits, gas)
}

func Decode(encoded []byte) (Schedule, error) {
	// 五个头字段、两个数组长度以及最多两组 MaxBands 个元素。
	if len(encoded) < 9*32 || len(encoded) > (7+2*MaxBands)*32 {
		return Schedule{}, errors.New("invalid pricing payload size")
	}
	values, err := scheduleArguments.Unpack(encoded)
	if err != nil {
		return Schedule{}, err
	}
	s := Schedule{Threshold: values[0].(uint64), Alpha: Ratio{Numerator: values[1].(uint64), Denominator: values[2].(uint64)}}
	limits, gas := values[3].([]uint64), values[4].([]uint64)
	if len(limits) != len(gas) {
		return Schedule{}, errors.New("pricing band arrays differ in length")
	}
	for i := range limits {
		s.Bands = append(s.Bands, InputBand{MaxMetric: limits[i], Gas: gas[i]})
	}
	canonical, err := s.Encode()
	if err != nil {
		return Schedule{}, err
	}
	if !bytes.Equal(canonical, encoded) {
		return Schedule{}, errors.New("noncanonical pricing encoding")
	}
	return s, nil
}
