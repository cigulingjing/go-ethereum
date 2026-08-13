package cryptoupgrade

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/blake2b"
	evmadapter "github.com/ethereum/go-ethereum/cryptoupgrade/internal/evm"
)

// precompile 计费规则保持独立，避免影响动态升级路径的 gas 元数据。
const (
	precompileMalformedInputGas uint64 = 1000
	precompileLightBaseGas      uint64 = 3000
	precompileMediumBaseGas     uint64 = 25000
	precompileHeavyBaseGas      uint64 = 200000
	precompileLightWordGas      uint64 = 60
	precompileMediumWordGas     uint64 = 600
	precompileHeavyWordGas      uint64 = 2000

	precompilePBKDF2MaxIterations = 1000000
	precompilePBKDF2MaxKeyLength  = 1 << 20
	precompileMaxUint64           = ^uint64(0)
)

type precompileHandler func([]interface{}) ([]interface{}, error)

type precompileSpec struct {
	address     common.Address
	name        string
	inputTypes  []string
	outputTypes []string
	requiredGas func([]byte) uint64
	handler     precompileHandler
}

// Precompile describes one deterministic cryptoupgrade algorithm exposed through the EVM precompile path.
// Precompile 实现 PrecompiledContract 接口，使算法可以通过固定地址被 EVM 直接调用。
type Precompile struct {
	address     common.Address
	name        string
	inputTypes  []string
	outputTypes []string
	requiredGas func([]byte) uint64
	handler     precompileHandler
}

var precompiles = mustPrecompiles([]precompileSpec{
	{
		address:     common.CryptoUpgradeAddAddress,
		name:        "Add",
		inputTypes:  []string{"uint256", "uint256"},
		outputTypes: []string{"uint256"},
		requiredGas: precompileEncodedGas(precompileLightBaseGas, precompileLightWordGas),
		handler:     precompileAdd,
	},
	{
		address:     common.CryptoUpgradeSha256Address,
		name:        "Sha256",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileLightBaseGas, precompileLightWordGas),
		handler:     precompileSha256,
	},
	{
		address:     common.CryptoUpgradeBlake2bSum256Address,
		name:        "Blake2bSum256",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileLightBaseGas, precompileLightWordGas),
		handler:     precompileBlake2bSum256,
	},
	{
		address:     common.CryptoUpgradeBlake2bSum384Address,
		name:        "Blake2bSum384",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileLightBaseGas, precompileLightWordGas),
		handler:     precompileBlake2bSum384,
	},
	{
		address:     common.CryptoUpgradeBlake2bSum512Address,
		name:        "Blake2bSum512",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileLightBaseGas, precompileLightWordGas),
		handler:     precompileBlake2bSum512,
	},
	{
		address:     common.CryptoUpgradeAesCBCEncryptAddress,
		name:        "AesCBCEncrypt",
		inputTypes:  []string{"bytes", "bytes", "bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileMediumBaseGas, precompileMediumWordGas),
		handler:     precompileAesCBCEncrypt,
	},
	{
		address:     common.CryptoUpgradeAesCBCDecryptAddress,
		name:        "AesCBCDecrypt",
		inputTypes:  []string{"bytes", "bytes", "bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileMediumBaseGas, precompileMediumWordGas),
		handler:     precompileAesCBCDecrypt,
	},
	{
		address:     common.CryptoUpgradePbkdf2Sha256Address,
		name:        "Pbkdf2Sha256",
		inputTypes:  []string{"bytes", "bytes", "uint256", "uint256"},
		outputTypes: []string{"bytes"},
		requiredGas: precompilePBKDF2Gas,
		handler:     precompilePBKDF2Sha256,
	},
	{
		address:     common.CryptoUpgradeDh2048PrivateAddress,
		name:        "Dh2048Private",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileHeavyBaseGas, precompileHeavyWordGas),
		handler:     precompileDH2048Private,
	},
	{
		address:     common.CryptoUpgradeDh2048PublicAddress,
		name:        "Dh2048Public",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileHeavyBaseGas, precompileHeavyWordGas),
		handler:     precompileDH2048Public,
	},
	{
		address:     common.CryptoUpgradeDh2048SecretAddress,
		name:        "Dh2048Secret",
		inputTypes:  []string{"bytes", "bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileHeavyBaseGas, precompileHeavyWordGas),
		handler:     precompileDH2048Secret,
	},
	{
		address:     common.CryptoUpgradeEd25519KeygenAddress,
		name:        "Ed25519Keygen",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileMediumBaseGas, precompileMediumWordGas),
		handler:     precompileEd25519Keygen,
	},
	{
		address:     common.CryptoUpgradeEd25519PublicAddress,
		name:        "Ed25519PublicKey",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileMediumBaseGas, precompileMediumWordGas),
		handler:     precompileEd25519PublicKey,
	},
	{
		address:     common.CryptoUpgradeEd25519SignAddress,
		name:        "Ed25519Sign",
		inputTypes:  []string{"bytes", "bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileMediumBaseGas, precompileMediumWordGas),
		handler:     precompileEd25519Sign,
	},
	{
		address:     common.CryptoUpgradeEd25519VerifyAddress,
		name:        "Ed25519Verify",
		inputTypes:  []string{"bytes", "bytes", "bytes"},
		outputTypes: []string{"bool"},
		requiredGas: precompileEncodedGas(precompileMediumBaseGas, precompileMediumWordGas),
		handler:     precompileEd25519Verify,
	},
	{
		address:     common.CryptoUpgradePedersenCommitAddress,
		name:        "PedersenCommit",
		inputTypes:  []string{"bytes", "bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileHeavyBaseGas, precompileHeavyWordGas),
		handler:     precompilePedersenCommit,
	},
	{
		address:     common.CryptoUpgradePedersenVerifyAddress,
		name:        "PedersenVerify",
		inputTypes:  []string{"bytes", "bytes", "bytes"},
		outputTypes: []string{"bool"},
		requiredGas: precompileEncodedGas(precompileHeavyBaseGas, precompileHeavyWordGas),
		handler:     precompilePedersenVerify,
	},
	{
		address:     common.CryptoUpgradeSchnorrPublicAddress,
		name:        "SchnorrPublicKey",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: precompileEncodedGas(precompileHeavyBaseGas, precompileHeavyWordGas),
		handler:     precompileSchnorrPublicKey,
	},
	{
		address:     common.CryptoUpgradeSchnorrVerifyAddress,
		name:        "SchnorrVerify",
		inputTypes:  []string{"bytes", "bytes"},
		outputTypes: []string{"bool"},
		requiredGas: precompileEncodedGas(precompileHeavyBaseGas, precompileHeavyWordGas),
		handler:     precompileSchnorrVerify,
	},
})

var precompileRegistry = evmadapter.MustRegistry(
	precompiles,
	func(precompile Precompile) common.Address { return precompile.address },
	func(precompile Precompile) string { return precompile.name },
)

func mustPrecompiles(specs []precompileSpec) []Precompile {
	out := make([]Precompile, len(specs))
	for i, spec := range specs {
		if spec.address == (common.Address{}) {
			panic("cryptoupgrade precompile has empty address")
		}
		if spec.name == "" {
			panic("cryptoupgrade precompile has empty name")
		}
		if spec.requiredGas == nil {
			panic("cryptoupgrade precompile has nil gas function")
		}
		if spec.handler == nil {
			panic("cryptoupgrade precompile has nil handler")
		}
		mustABIArguments(spec.inputTypes)
		mustABIArguments(spec.outputTypes)
		out[i] = Precompile{
			address:     spec.address,
			name:        spec.name,
			inputTypes:  append([]string(nil), spec.inputTypes...),
			outputTypes: append([]string(nil), spec.outputTypes...),
			requiredGas: spec.requiredGas,
			handler:     spec.handler,
		}
	}
	return out
}

func mustABIArguments(types []string) {
	for _, typ := range types {
		if _, err := abi.NewType(typ, "", nil); err != nil {
			panic(fmt.Sprintf("invalid cryptoupgrade precompile ABI type %q: %v", typ, err))
		}
	}
}

// Core包调用的入口
func Precompiles() []Precompile {
	return precompileRegistry.Entries()
}

func PrecompileAddresses() []common.Address {
	return precompileRegistry.Addresses()
}

func PrecompileByName(name string) (Precompile, bool) {
	return precompileRegistry.ByName(name)
}

func (p Precompile) Address() common.Address {
	return p.address
}

func (p Precompile) Name() string {
	return p.name
}

func (p Precompile) InputTypes() []string {
	return append([]string(nil), p.inputTypes...)
}

func (p Precompile) OutputTypes() []string {
	return append([]string(nil), p.outputTypes...)
}

func (p Precompile) RequiredGas(input []byte) uint64 {
	return p.requiredGas(input)
}

func (p Precompile) Run(input []byte) ([]byte, error) {
	args, err := UnpackInput(input, p.inputTypes)
	if err != nil {
		return nil, err
	}
	output, err := p.handler(args)
	if err != nil {
		return nil, err
	}
	return PackOutput(output, p.outputTypes)
}

func precompileEncodedGas(base, perWord uint64) func([]byte) uint64 {
	return func(input []byte) uint64 {
		return precompileSaturatingAdd(base, precompileSaturatingMul(perWord, precompileWords(len(input))))
	}
}

func precompilePBKDF2Gas(input []byte) uint64 {
	gas := precompileEncodedGas(precompileMediumBaseGas, precompileMediumWordGas)(input)
	args, err := UnpackInput(input, []string{"bytes", "bytes", "uint256", "uint256"})
	if err != nil {
		return precompileSaturatingAdd(precompileMalformedInputGas, precompileSaturatingMul(precompileLightWordGas, precompileWords(len(input))))
	}
	iterations, ok := precompileBoundedInt(args[2], 1, precompilePBKDF2MaxIterations)
	if !ok {
		return gas
	}
	keyLength, ok := precompileBoundedInt(args[3], 0, precompilePBKDF2MaxKeyLength)
	if !ok || keyLength == 0 {
		return gas
	}
	blocks := precompileWords(keyLength)
	rounds := precompileSaturatingMul(uint64(iterations), blocks)
	return precompileSaturatingAdd(gas, precompileSaturatingMul(rounds, 25))
}

func precompileWords(n int) uint64 {
	return uint64(n+31) / 32
}

func precompileSaturatingAdd(a, b uint64) uint64 {
	if precompileMaxUint64-a < b {
		return precompileMaxUint64
	}
	return a + b
}

func precompileSaturatingMul(a, b uint64) uint64 {
	if a != 0 && b > precompileMaxUint64/a {
		return precompileMaxUint64
	}
	return a * b
}

func precompileBytes(args []interface{}, index int) ([]byte, error) {
	v, ok := args[index].([]byte)
	if !ok {
		return nil, fmt.Errorf("argument %d has type %T, want []byte", index, args[index])
	}
	return v, nil
}

func precompileBig(args []interface{}, index int) (*big.Int, error) {
	v, ok := args[index].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("argument %d has type %T, want *big.Int", index, args[index])
	}
	return v, nil
}

func precompileBoundedInt(arg interface{}, min, max int) (int, bool) {
	v, ok := arg.(*big.Int)
	if !ok || v == nil || !v.IsInt64() {
		return 0, false
	}
	n := v.Int64()
	if n < int64(min) || n > int64(max) {
		return 0, false
	}
	return int(n), true
}

func precompileAdd(args []interface{}) ([]interface{}, error) {
	a, err := precompileBig(args, 0)
	if err != nil {
		return nil, err
	}
	b, err := precompileBig(args, 1)
	if err != nil {
		return nil, err
	}
	sum := new(big.Int).Add(a, b)
	if sum.BitLen() > 256 {
		sum.Mod(sum, precompileUint256Mod())
	}
	return []interface{}{sum}, nil
}

func precompileSha256(args []interface{}) ([]interface{}, error) {
	data, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	return []interface{}{append([]byte(nil), sum[:]...)}, nil
}

func precompileBlake2bSum256(args []interface{}) ([]interface{}, error) {
	data, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	sum := blake2b.Sum256(data)
	return []interface{}{append([]byte(nil), sum[:]...)}, nil
}

func precompileBlake2bSum384(args []interface{}) ([]interface{}, error) {
	data, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	sum := blake2b.Sum384(data)
	return []interface{}{append([]byte(nil), sum[:]...)}, nil
}

func precompileBlake2bSum512(args []interface{}) ([]interface{}, error) {
	data, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	sum := blake2b.Sum512(data)
	return []interface{}{append([]byte(nil), sum[:]...)}, nil
}

func precompileAesCBCEncrypt(args []interface{}) ([]interface{}, error) {
	key, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	iv, err := precompileBytes(args, 1)
	if err != nil {
		return nil, err
	}
	plaintext, err := precompileBytes(args, 2)
	if err != nil {
		return nil, err
	}
	out, err := aesCBCEncrypt(key, iv, plaintext)
	if err != nil {
		return nil, err
	}
	return []interface{}{out}, nil
}

func precompileAesCBCDecrypt(args []interface{}) ([]interface{}, error) {
	key, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	iv, err := precompileBytes(args, 1)
	if err != nil {
		return nil, err
	}
	ciphertext, err := precompileBytes(args, 2)
	if err != nil {
		return nil, err
	}
	out, err := aesCBCDecrypt(key, iv, ciphertext)
	if err != nil {
		return nil, err
	}
	return []interface{}{out}, nil
}

func aesCBCEncrypt(key, iv, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(iv) != block.BlockSize() {
		return nil, fmt.Errorf("AES-CBC IV length %d, want %d", len(iv), block.BlockSize())
	}
	padded := aesCBCPKCS7Pad(plaintext, block.BlockSize())
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)
	return ciphertext, nil
}

func aesCBCDecrypt(key, iv, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(iv) != block.BlockSize() {
		return nil, fmt.Errorf("AES-CBC IV length %d, want %d", len(iv), block.BlockSize())
	}
	if len(ciphertext) == 0 || len(ciphertext)%block.BlockSize() != 0 {
		return nil, errors.New("AES-CBC ciphertext length is not a positive block multiple")
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)
	return aesCBCPKCS7Unpad(plaintext, block.BlockSize())
}

func aesCBCPKCS7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+padding)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(padding)
	}
	return out
}

func aesCBCPKCS7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("invalid PKCS#7 data length")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, errors.New("invalid PKCS#7 padding")
	}
	for _, b := range data[len(data)-padding:] {
		if int(b) != padding {
			return nil, errors.New("invalid PKCS#7 padding")
		}
	}
	out := make([]byte, len(data)-padding)
	copy(out, data[:len(data)-padding])
	return out, nil
}

func precompilePBKDF2Sha256(args []interface{}) ([]interface{}, error) {
	password, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	salt, err := precompileBytes(args, 1)
	if err != nil {
		return nil, err
	}
	iterations, ok := precompileBoundedInt(args[2], 1, precompilePBKDF2MaxIterations)
	if !ok {
		return nil, errors.New("PBKDF2 iterations out of range")
	}
	keyLength, ok := precompileBoundedInt(args[3], 0, precompilePBKDF2MaxKeyLength)
	if !ok {
		return nil, errors.New("PBKDF2 key length out of range")
	}
	return []interface{}{pbkdf2Sha256(password, salt, iterations, keyLength)}, nil
}

func pbkdf2Sha256(password, salt []byte, iterations, keyLength int) []byte {
	if keyLength == 0 {
		return []byte{}
	}
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	blocks := (keyLength + hashLen - 1) / hashLen
	var blockIndex [4]byte
	u := make([]byte, 0, hashLen)
	t := make([]byte, hashLen)
	derived := make([]byte, 0, blocks*hashLen)

	for block := 1; block <= blocks; block++ {
		prf.Reset()
		prf.Write(salt)
		blockIndex[0] = byte(block >> 24)
		blockIndex[1] = byte(block >> 16)
		blockIndex[2] = byte(block >> 8)
		blockIndex[3] = byte(block)
		prf.Write(blockIndex[:])
		u = prf.Sum(u[:0])
		copy(t, u)

		for round := 2; round <= iterations; round++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for i := range t {
				t[i] ^= u[i]
			}
		}
		derived = append(derived, t...)
	}
	return derived[:keyLength]
}

func precompileDH2048Private(args []interface{}) ([]interface{}, error) {
	seed, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	if len(seed) == 0 {
		return nil, errors.New("DH2048 private seed is empty")
	}
	p := precompileRFC3526Prime()
	max := new(big.Int).Sub(p, big.NewInt(3))
	digest := sha512.Sum512(seed)
	x := new(big.Int).SetBytes(digest[:])
	x.Mod(x, max)
	x.Add(x, big.NewInt(2))
	return []interface{}{precompileFixedBytes(x, precompileRFC3526FieldBytes())}, nil
}

func precompileDH2048Public(args []interface{}) ([]interface{}, error) {
	privateKey, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	p := precompileRFC3526Prime()
	x := precompileRFC3526Scalar(privateKey, p)
	y := new(big.Int).Exp(big.NewInt(2), x, p)
	return []interface{}{precompileFixedBytes(y, precompileRFC3526FieldBytes())}, nil
}

func precompileDH2048Secret(args []interface{}) ([]interface{}, error) {
	privateKey, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	peerPublicKey, err := precompileBytes(args, 1)
	if err != nil {
		return nil, err
	}
	p := precompileRFC3526Prime()
	y := new(big.Int).SetBytes(peerPublicKey)
	if y.Cmp(big.NewInt(1)) <= 0 || y.Cmp(new(big.Int).Sub(p, big.NewInt(1))) >= 0 {
		return nil, errors.New("invalid DH2048 peer public key")
	}
	x := precompileRFC3526Scalar(privateKey, p)
	secret := new(big.Int).Exp(y, x, p)
	return []interface{}{precompileFixedBytes(secret, precompileRFC3526FieldBytes())}, nil
}

func precompileEd25519Keygen(args []interface{}) ([]interface{}, error) {
	seed, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("Ed25519 seed length %d, want %d", len(seed), ed25519.SeedSize)
	}
	return []interface{}{append([]byte(nil), ed25519.NewKeyFromSeed(seed)...)}, nil
}

func precompileEd25519PublicKey(args []interface{}) ([]interface{}, error) {
	privateKey, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	if len(privateKey) == ed25519.SeedSize {
		privateKey = ed25519.NewKeyFromSeed(privateKey)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("Ed25519 private key length %d, want %d", len(privateKey), ed25519.PrivateKeySize)
	}
	publicKey := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
	return []interface{}{append([]byte(nil), publicKey...)}, nil
}

func precompileEd25519Sign(args []interface{}) ([]interface{}, error) {
	privateKey, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	message, err := precompileBytes(args, 1)
	if err != nil {
		return nil, err
	}
	if len(privateKey) == ed25519.SeedSize {
		privateKey = ed25519.NewKeyFromSeed(privateKey)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("Ed25519 private key length %d, want %d", len(privateKey), ed25519.PrivateKeySize)
	}
	return []interface{}{ed25519.Sign(ed25519.PrivateKey(privateKey), message)}, nil
}

func precompileEd25519Verify(args []interface{}) ([]interface{}, error) {
	publicKey, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	message, err := precompileBytes(args, 1)
	if err != nil {
		return nil, err
	}
	signature, err := precompileBytes(args, 2)
	if err != nil {
		return nil, err
	}
	if len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize {
		return []interface{}{false}, nil
	}
	return []interface{}{ed25519.Verify(ed25519.PublicKey(publicKey), message, signature)}, nil
}

func precompilePedersenCommit(args []interface{}) ([]interface{}, error) {
	message, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	blinding, err := precompileBytes(args, 1)
	if err != nil {
		return nil, err
	}
	return []interface{}{pedersenCommit(message, blinding)}, nil
}

func precompilePedersenVerify(args []interface{}) ([]interface{}, error) {
	message, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	blinding, err := precompileBytes(args, 1)
	if err != nil {
		return nil, err
	}
	commitment, err := precompileBytes(args, 2)
	if err != nil {
		return nil, err
	}
	expected := pedersenCommit(message, blinding)
	return []interface{}{bytes.Equal(expected, commitment)}, nil
}

func precompileSchnorrPublicKey(args []interface{}) ([]interface{}, error) {
	secret, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	p, q := precompileRFC3526Subgroup()
	x := precompileRFC3526SubgroupScalar(secret, q)
	y := new(big.Int).Exp(big.NewInt(4), x, p)
	return []interface{}{precompileFixedBytes(y, precompileRFC3526FieldBytes())}, nil
}

func precompileSchnorrVerify(args []interface{}) ([]interface{}, error) {
	message, err := precompileBytes(args, 0)
	if err != nil {
		return nil, err
	}
	proof, err := precompileBytes(args, 1)
	if err != nil {
		return nil, err
	}
	return []interface{}{schnorrVerify(message, proof)}, nil
}

func precompileUint256Mod() *big.Int {
	return new(big.Int).Lsh(big.NewInt(1), 256)
}

const precompileRFC3526PrimeHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3" +
	"404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BF" +
	"B5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C6" +
	"2F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28" +
	"FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"

func precompileRFC3526Prime() *big.Int {
	p, _ := new(big.Int).SetString(precompileRFC3526PrimeHex, 16)
	return p
}

func precompileRFC3526Subgroup() (*big.Int, *big.Int) {
	p := precompileRFC3526Prime()
	q := new(big.Int).Sub(p, big.NewInt(1))
	q.Rsh(q, 1)
	return p, q
}

func precompileRFC3526FieldBytes() int {
	return (len(precompileRFC3526PrimeHex) + 1) / 2
}

func precompileRFC3526Scalar(data []byte, p *big.Int) *big.Int {
	x := new(big.Int).SetBytes(data)
	min := big.NewInt(2)
	max := new(big.Int).Sub(p, min)
	if x.Cmp(min) >= 0 && x.Cmp(max) <= 0 {
		return x
	}
	span := new(big.Int).Sub(p, big.NewInt(3))
	x.Mod(x, span)
	x.Add(x, min)
	return x
}

func precompileRFC3526SubgroupScalar(data []byte, q *big.Int) *big.Int {
	x := new(big.Int).SetBytes(data)
	x.Mod(x, q)
	if x.Sign() == 0 {
		x.SetInt64(1)
	}
	return x
}

func precompileFixedBytes(x *big.Int, size int) []byte {
	if x == nil {
		return nil
	}
	b := x.Bytes()
	if len(b) > size {
		return b[len(b)-size:]
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

func pedersenCommit(message, blinding []byte) []byte {
	p, q := precompileRFC3526Subgroup()
	g := big.NewInt(4)
	h := pedersenH(p)
	m := precompileRFC3526ModScalar(message, q)
	r := precompileRFC3526ModScalar(blinding, q)

	gm := new(big.Int).Exp(g, m, p)
	hr := new(big.Int).Exp(h, r, p)
	commitment := new(big.Int).Mul(gm, hr)
	commitment.Mod(commitment, p)
	return precompileFixedBytes(commitment, precompileRFC3526FieldBytes())
}

func precompileRFC3526ModScalar(data []byte, q *big.Int) *big.Int {
	x := new(big.Int).SetBytes(data)
	x.Mod(x, q)
	return x
}

func pedersenH(p *big.Int) *big.Int {
	for counter := byte(0); ; counter++ {
		sum := sha256.Sum256([]byte("cryptoupgrade-pedersen-h-" + string([]byte{counter})))
		h := new(big.Int).SetBytes(sum[:])
		h.Mod(h, p)
		h.Exp(h, big.NewInt(2), p)
		if h.Cmp(big.NewInt(1)) > 0 {
			return h
		}
	}
}

func schnorrVerify(message, proof []byte) bool {
	size := precompileRFC3526FieldBytes()
	if len(proof) != 3*size {
		return false
	}
	p, q := precompileRFC3526Subgroup()
	y := new(big.Int).SetBytes(proof[:size])
	t := new(big.Int).SetBytes(proof[size : 2*size])
	s := new(big.Int).SetBytes(proof[2*size:])
	if !precompileRFC3526GroupElement(y, p) || !precompileRFC3526GroupElement(t, p) || s.Cmp(q) >= 0 {
		return false
	}

	g := big.NewInt(4)
	c := schnorrChallenge(q, y, t, message)
	left := new(big.Int).Exp(g, s, p)
	yc := new(big.Int).Exp(y, c, p)
	right := new(big.Int).Mul(t, yc)
	right.Mod(right, p)
	return bytes.Equal(precompileFixedBytes(left, size), precompileFixedBytes(right, size))
}

func schnorrChallenge(q, y, t *big.Int, message []byte) *big.Int {
	h := sha256.New()
	h.Write([]byte("cryptoupgrade-schnorr"))
	h.Write(precompileFixedBytes(y, precompileRFC3526FieldBytes()))
	h.Write(precompileFixedBytes(t, precompileRFC3526FieldBytes()))
	h.Write(message)
	c := new(big.Int).SetBytes(h.Sum(nil))
	c.Mod(c, q)
	return c
}

func precompileRFC3526GroupElement(x, p *big.Int) bool {
	return x.Cmp(big.NewInt(1)) > 0 && x.Cmp(new(big.Int).Sub(p, big.NewInt(1))) < 0
}
