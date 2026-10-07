package main

import "math/big"

// UpgradeConsistencyProbe v1 returns a fixed value while retaining the shared ABI.
func UpgradeConsistencyProbe(_, _ *big.Int) *big.Int { return big.NewInt(1) }
