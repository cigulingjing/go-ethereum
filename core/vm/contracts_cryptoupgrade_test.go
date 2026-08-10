package vm

import (
	"bytes"
	"crypto/ed25519"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestCryptoUpgradePrecompilesRegistered(t *testing.T) {
	contracts := activePrecompiledContracts(params.Rules{})
	addresses := ActivePrecompiles(params.Rules{})
	addressSet := make(map[common.Address]bool)
	for _, address := range addresses {
		if addressSet[address] {
			t.Fatalf("duplicate active precompile address %s", address)
		}
		addressSet[address] = true
	}

	for _, entry := range cryptoupgrade.Precompiles() {
		if _, ok := contracts[entry.Address()]; !ok {
			t.Fatalf("missing cryptoupgrade precompile contract %s at %s", entry.Name(), entry.Address())
		}
		if !addressSet[entry.Address()] {
			t.Fatalf("missing cryptoupgrade precompile address %s at %s", entry.Name(), entry.Address())
		}
		if _, mutated := PrecompiledContractsHomestead[entry.Address()]; mutated {
			t.Fatalf("cryptoupgrade precompile %s was inserted into the Homestead static map", entry.Name())
		}
	}
	for _, address := range []common.Address{
		common.BytesToAddress([]byte{0x01}),
		common.BytesToAddress([]byte{0x02}),
		common.Blake2bSum256Address,
	} {
		if _, ok := contracts[address]; !ok {
			t.Fatalf("existing precompile %s missing after cryptoupgrade merge", address)
		}
	}
}

func TestCryptoUpgradePrecompileRunAndErrors(t *testing.T) {
	contracts := activePrecompiledContracts(params.Rules{})
	entry := mustCryptoUpgradeEntry(t, "Sha256")
	input := packCryptoUpgradeABI(t, entry.InputTypes(), []byte("abc"))
	expected, err := entry.Run(input)
	if err != nil {
		t.Fatalf("reference run failed: %v", err)
	}
	precompile := contracts[entry.Address()]
	requiredGas := precompile.RequiredGas(input)
	ret, remaining, err := RunPrecompiledContract(nil, precompile, entry.Address(), input, NewGasBudget(requiredGas+123, 0), nil, params.Rules{})
	if err != nil {
		t.Fatalf("RunPrecompiledContract failed: %v", err)
	}
	if !bytes.Equal(ret, expected) {
		t.Fatalf("precompile output mismatch: got %x want %x", ret, expected)
	}
	if remaining.RegularGas != 123 {
		t.Fatalf("remaining gas = %d, want 123", remaining.RegularGas)
	}

	if _, _, err := RunPrecompiledContract(nil, precompile, entry.Address(), input, NewGasBudget(requiredGas-1, 0), nil, params.Rules{}); err != ErrOutOfGas {
		t.Fatalf("OOG error = %v, want %v", err, ErrOutOfGas)
	}
	if _, _, err := RunPrecompiledContract(nil, precompile, entry.Address(), []byte{0x01}, NewGasBudget(precompile.RequiredGas([]byte{0x01}), 0), nil, params.Rules{}); err == nil {
		t.Fatalf("invalid ABI input succeeded")
	}

	verifyEntry := mustCryptoUpgradeEntry(t, "Ed25519Verify")
	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	message := []byte("message")
	signature := ed25519.Sign(privateKey, message)
	verifyInput := packCryptoUpgradeABI(t, verifyEntry.InputTypes(), []byte(publicKey), message, signature)
	verifyExpected, err := verifyEntry.Run(verifyInput)
	if err != nil {
		t.Fatalf("reference verify failed: %v", err)
	}
	verifyPrecompile := contracts[verifyEntry.Address()]
	verifyRet, _, err := RunPrecompiledContract(nil, verifyPrecompile, verifyEntry.Address(), verifyInput, NewGasBudget(verifyPrecompile.RequiredGas(verifyInput), 0), nil, params.Rules{})
	if err != nil {
		t.Fatalf("Ed25519Verify precompile failed: %v", err)
	}
	if !bytes.Equal(verifyRet, verifyExpected) {
		t.Fatalf("boolean ABI output mismatch: got %x want %x", verifyRet, verifyExpected)
	}
}

func TestCryptoUpgradePrecompileEVMCallPaths(t *testing.T) {
	entry := mustCryptoUpgradeEntry(t, "Sha256")
	input := packCryptoUpgradeABI(t, entry.InputTypes(), []byte("abc"))
	expected, err := entry.Run(input)
	if err != nil {
		t.Fatalf("reference run failed: %v", err)
	}
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	evm := NewEVM(BlockContext{
		CanTransfer: func(StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:    func(StateDB, common.Address, common.Address, *uint256.Int, *params.Rules) {},
		BlockNumber: big.NewInt(0),
		Random:      new(common.Hash),
	}, statedb, params.TestChainConfig, Config{})

	caller := common.HexToAddress("0x1234")
	gas := NewGasBudget(entry.RequiredGas(input)+100000, 0)
	ret, _, err := evm.Call(caller, entry.Address(), input, gas, new(uint256.Int))
	if err != nil {
		t.Fatalf("CALL failed: %v", err)
	}
	if !bytes.Equal(ret, expected) {
		t.Fatalf("CALL output mismatch: got %x want %x", ret, expected)
	}

	ret, _, err = evm.StaticCall(caller, entry.Address(), input, gas)
	if err != nil {
		t.Fatalf("STATICCALL failed: %v", err)
	}
	if !bytes.Equal(ret, expected) {
		t.Fatalf("STATICCALL output mismatch: got %x want %x", ret, expected)
	}
}

func TestCryptoUpgradeCodeStorageSpecialCaseRemains(t *testing.T) {
	contracts := activePrecompiledContracts(params.Rules{})
	if _, ok := contracts[common.CodeStorageAddress]; ok {
		t.Fatalf("CodeStorageAddress should remain outside the normal precompile map")
	}
	if !isCryptoUpgradeCall(common.CodeStorageAddress, cryptoupgrade.CodeStorageABI.Methods["getGas"].ID) {
		t.Fatalf("CodeStorage getGas selector is no longer recognized")
	}
	if isCryptoUpgradeCall(common.CryptoUpgradeSha256Address, packCryptoUpgradeABI(t, mustCryptoUpgradeEntry(t, "Sha256").InputTypes(), []byte("abc"))) {
		t.Fatalf("algorithm precompile call should not use CodeStorage special-case dispatch")
	}
}

func mustCryptoUpgradeEntry(t *testing.T, name string) cryptoupgrade.Precompile {
	t.Helper()
	entry, ok := cryptoupgrade.PrecompileByName(name)
	if !ok {
		t.Fatalf("missing cryptoupgrade precompile %s", name)
	}
	return entry
}

func packCryptoUpgradeABI(t *testing.T, types []string, values ...interface{}) []byte {
	t.Helper()
	args := make(abi.Arguments, 0, len(types))
	for _, typ := range types {
		abiType, err := abi.NewType(typ, "", nil)
		if err != nil {
			t.Fatalf("invalid ABI type %s: %v", typ, err)
		}
		args = append(args, abi.Argument{Type: abiType})
	}
	out, err := args.Pack(values...)
	if err != nil {
		t.Fatalf("ABI pack failed: %v", err)
	}
	return out
}
