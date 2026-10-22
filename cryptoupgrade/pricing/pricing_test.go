package pricing

import (
	"errors"
	"math"
	"testing"
)

func testProfile() Profile {
	return Profile{
		Threshold: 2,
		Alpha:     Ratio{Numerator: 1, Denominator: 2},
		Algorithms: []Algorithm{{Name: "Add", Version: 1, Metric: "operandBits", Bands: []InputBand{
			{MaxMetric: 31, Gas: 10}, {MaxMetric: math.MaxUint64, Gas: 21},
		}}},
		Load: []LoadBand{
			{MaxCalls: 0, Multiplier: Ratio{Numerator: 1, Denominator: 1}},
			{MaxCalls: math.MaxUint64, Multiplier: Ratio{Numerator: 5, Denominator: 4}},
		},
	}
}

func TestQuote(t *testing.T) {
	profile := testProfile()
	if err := profile.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		size   uint64
		index  uint64
		parent uint64
		want   uint64
	}{
		{"first", 31, 1, 0, 10},
		{"at threshold", 31, 2, 0, 10},
		{"past threshold", 31, 3, 0, 15},
		{"input and load", 32, 3, 3, 40},
		{"round only once", 32, 4, 3, 60},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := profile.Quote("Add", 1, test.size, test.index, test.parent)
			if err != nil || got != test.want {
				t.Fatalf("Quote = %d, %v; want %d", got, err, test.want)
			}
		})
	}
	if _, err := profile.Quote("Add", 2, 0, 1, 0); err == nil {
		t.Fatal("missing version accepted")
	}
	if _, err := profile.Quote("Add", 1, 0, 0, 0); err == nil {
		t.Fatal("zero index accepted")
	}
}

func TestOverflowAndZeroAlpha(t *testing.T) {
	profile := testProfile()
	profile.Alpha = Ratio{Numerator: 0, Denominator: 1}
	got, err := profile.Quote("Add", 1, 0, math.MaxUint64, 0)
	if err != nil || got != 10 {
		t.Fatalf("zero alpha: %d %v", got, err)
	}
	profile.Alpha = Ratio{Numerator: 1, Denominator: 1}
	if _, err := profile.Quote("Add", 1, 0, math.MaxUint64, 0); !errors.Is(err, ErrGasOverflow) {
		t.Fatalf("large index: %v", err)
	}
	profile.Algorithms[0].Bands[0].Gas = math.MaxUint64
	if _, err := profile.Quote("Add", 1, 0, 3, 1); !errors.Is(err, ErrGasOverflow) {
		t.Fatalf("large base: %v", err)
	}
}

func TestRejectInvalidProfiles(t *testing.T) {
	for _, change := range []func(*Profile){
		func(profile *Profile) { profile.Threshold = 0 },
		func(profile *Profile) { profile.Alpha.Denominator = 0 },
		func(profile *Profile) { profile.Alpha.Denominator = maxAlphaDenominator + 1 },
		func(profile *Profile) { profile.Algorithms[0].Bands[0].Gas = 0 },
		func(profile *Profile) { profile.Algorithms[0].Bands[1].MaxMetric = 30 },
		func(profile *Profile) { profile.Algorithms[0].Metric = "" },
		func(profile *Profile) { profile.Load[1].Multiplier.Numerator = 0 },
		func(profile *Profile) { profile.Load[1].MaxCalls = 0 },
		func(profile *Profile) { profile.Algorithms = append(profile.Algorithms, profile.Algorithms[0]) },
	} {
		profile := testProfile()
		change(&profile)
		if err := profile.Validate(); err == nil {
			t.Fatalf("invalid profile accepted: %+v", profile)
		}
	}
}
