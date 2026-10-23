package pricing

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestScheduleEncodingAndQuotes(t *testing.T) {
	s := Schedule{Threshold: 2, Alpha: Ratio{1, 2}, Bands: []InputBand{{31, 10}, {math.MaxUint64, 21}}}
	raw, err := s.Encode()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(raw)
	if err != nil || !reflect.DeepEqual(s, decoded) {
		t.Fatalf("round trip: %+v, %v", decoded, err)
	}
	for _, tc := range []struct{ size, index, want uint64 }{{31, 2, 10}, {31, 3, 15}, {32, 3, 32}, {32, 4, 48}} {
		got, err := decoded.Quote(tc.size, tc.index)
		if err != nil || got != tc.want {
			t.Fatalf("%+v: gas=%d err=%v", tc, got, err)
		}
	}
	if _, err := decoded.Quote(0, math.MaxUint64); !errors.Is(err, ErrGasOverflow) {
		t.Fatalf("overflow: %v", err)
	}
	if _, err := Decode(append(raw, make([]byte, 32)...)); err == nil {
		t.Fatal("trailing ABI data accepted")
	}
	if _, err := Decode(raw[:len(raw)-1]); err == nil {
		t.Fatal("truncated ABI accepted")
	}
	decoded.Alpha.Numerator = 0
	if gas, err := decoded.Quote(0, math.MaxUint64); err != nil || gas != 10 {
		t.Fatalf("zero alpha: %d %v", gas, err)
	}
}

func TestScheduleRejectsInvalidParameters(t *testing.T) {
	for _, mutate := range []func(*Schedule){
		func(s *Schedule) { s.Threshold = 0 },
		func(s *Schedule) { s.Alpha.Denominator = 0 },
		func(s *Schedule) { s.Alpha.Denominator = 10001 },
		func(s *Schedule) { s.Bands = nil },
		func(s *Schedule) { s.Bands[0].Gas = 0 },
		func(s *Schedule) { s.Bands[1].Gas = 1 },
		func(s *Schedule) { s.Bands[1].MaxMetric = 31 },
		func(s *Schedule) { s.Bands = s.Bands[:1] },
		func(s *Schedule) { s.Bands = make([]InputBand, MaxBands+1) },
	} {
		s := Schedule{Threshold: 2, Alpha: Ratio{1, 2}, Bands: []InputBand{{31, 10}, {math.MaxUint64, 21}}}
		mutate(&s)
		if _, err := s.Encode(); err == nil {
			t.Fatalf("invalid schedule accepted: %+v", s)
		}
	}
}
