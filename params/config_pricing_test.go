package params

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/cryptoupgrade/pricing"
)

func TestExperimentalCryptoUpgradePricingConfig(t *testing.T) {
	profile := pricing.Profile{Threshold: 2, Alpha: pricing.Ratio{Numerator: 1, Denominator: 2},
		Algorithms: []pricing.Algorithm{{Name: "Pbkdf2Sha256", Version: 1, Metric: "encodedInputBytes",
			Bands: []pricing.InputBand{{MaxMetric: math.MaxUint64, Gas: 3000}}}},
		Load: []pricing.LoadBand{{MaxCalls: math.MaxUint64, Multiplier: pricing.Ratio{Numerator: 1, Denominator: 1}}}}
	config := *AllCliqueProtocolChanges
	config.CryptoUpgradePricing = &CryptoUpgradePricingConfig{ActivationBlock: 10, Profile: profile}
	if err := config.CheckConfigForkOrder(); err != nil {
		t.Fatal(err)
	}
	changed := config
	other := *config.CryptoUpgradePricing
	other.Profile.Alpha = pricing.Ratio{Numerator: 1, Denominator: 3}
	changed.CryptoUpgradePricing = &other
	if err := config.CheckCompatible(&changed, 9, 0); err != nil {
		t.Fatalf("unexpected pre-activation incompatibility: %v", err)
	}
	if err := config.CheckCompatible(&changed, 10, 0); err == nil {
		t.Fatal("post-activation profile mutation accepted")
	}
	changed.CryptoUpgradePricing = &CryptoUpgradePricingConfig{ActivationBlock: 10, Profile: profile}
	changed.CryptoUpgradePricing.Profile.Load = []pricing.LoadBand{
		{MaxCalls: 0, Multiplier: pricing.Ratio{Numerator: 1, Denominator: 1}},
		{MaxCalls: math.MaxUint64, Multiplier: pricing.Ratio{Numerator: 2, Denominator: 1}},
	}
	if err := changed.CheckConfigForkOrder(); err == nil {
		t.Fatal("unsupported non-neutral parent factor accepted")
	}
}
