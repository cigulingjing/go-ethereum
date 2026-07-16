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
)

// 原生的计费规则
const (
	nativeMalformedInputGas uint64 = 1000
	nativeLightBaseGas      uint64 = 3000
	nativeMediumBaseGas     uint64 = 25000
	nativeHeavyBaseGas      uint64 = 200000
	nativeLightWordGas      uint64 = 60
	nativeMediumWordGas     uint64 = 600
	nativeHeavyWordGas      uint64 = 2000

	nativePBKDF2MaxIterations = 1000000
	nativePBKDF2MaxKeyLength  = 1 << 20
	nativeMaxUint64           = ^uint64(0)
)

type nativePrecompileHandler func([]interface{}) ([]interface{}, error)

type nativePrecompileSpec struct {
	address     common.Address
	name        string
	inputTypes  []string
	outputTypes []string
	requiredGas func([]byte) uint64
	handler     nativePrecompileHandler
}

// NativePrecompile describes one deterministic cryptoupgrade algorithm exposed
// through the EVM precompile path.
type NativePrecompile struct {
	address     common.Address
	name        string
	inputTypes  []string
	outputTypes []string
	requiredGas func([]byte) uint64
	handler     nativePrecompileHandler
}

var nativePrecompiles = mustNativePrecompiles([]nativePrecompileSpec{
	{
		address:     common.CryptoUpgradeAddAddress,
		name:        "Add",
		inputTypes:  []string{"uint256", "uint256"},
		outputTypes: []string{"uint256"},
		requiredGas: nativeEncodedGas(nativeLightBaseGas, nativeLightWordGas),
		handler:     nativeAdd,
	},
	{
		address:     common.CryptoUpgradeSha256Address,
		name:        "Sha256",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeLightBaseGas, nativeLightWordGas),
		handler:     nativeSha256,
	},
	{
		address:     common.CryptoUpgradeBlake2bSum256Address,
		name:        "Blake2bSum256",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeLightBaseGas, nativeLightWordGas),
		handler:     nativeBlake2bSum256,
	},
	{
		address:     common.CryptoUpgradeBlake2bSum384Address,
		name:        "Blake2bSum384",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeLightBaseGas, nativeLightWordGas),
		handler:     nativeBlake2bSum384,
	},
	{
		address:     common.CryptoUpgradeBlake2bSum512Address,
		name:        "Blake2bSum512",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeLightBaseGas, nativeLightWordGas),
		handler:     nativeBlake2bSum512,
	},
	{
		address:     common.CryptoUpgradeAesCBCEncryptAddress,
		name:        "AesCBCEncrypt",
		inputTypes:  []string{"bytes", "bytes", "bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeMediumBaseGas, nativeMediumWordGas),
		handler:     nativeAesCBCEncrypt,
	},
	{
		address:     common.CryptoUpgradeAesCBCDecryptAddress,
		name:        "AesCBCDecrypt",
		inputTypes:  []string{"bytes", "bytes", "bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeMediumBaseGas, nativeMediumWordGas),
		handler:     nativeAesCBCDecrypt,
	},
	{
		address:     common.CryptoUpgradePbkdf2Sha256Address,
		name:        "Pbkdf2Sha256",
		inputTypes:  []string{"bytes", "bytes", "uint256", "uint256"},
		outputTypes: []string{"bytes"},
		requiredGas: nativePBKDF2Gas,
		handler:     nativePBKDF2Sha256,
	},
	{
		address:     common.CryptoUpgradeDh2048PrivateAddress,
		name:        "Dh2048Private",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeHeavyBaseGas, nativeHeavyWordGas),
		handler:     nativeDH2048Private,
	},
	{
		address:     common.CryptoUpgradeDh2048PublicAddress,
		name:        "Dh2048Public",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeHeavyBaseGas, nativeHeavyWordGas),
		handler:     nativeDH2048Public,
	},
	{
		address:     common.CryptoUpgradeDh2048SecretAddress,
		name:        "Dh2048Secret",
		inputTypes:  []string{"bytes", "bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeHeavyBaseGas, nativeHeavyWordGas),
		handler:     nativeDH2048Secret,
	},
	{
		address:     common.CryptoUpgradeEd25519KeygenAddress,
		name:        "Ed25519Keygen",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeMediumBaseGas, nativeMediumWordGas),
		handler:     nativeEd25519Keygen,
	},
	{
		address:     common.CryptoUpgradeEd25519PublicAddress,
		name:        "Ed25519PublicKey",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeMediumBaseGas, nativeMediumWordGas),
		handler:     nativeEd25519PublicKey,
	},
	{
		address:     common.CryptoUpgradeEd25519SignAddress,
		name:        "Ed25519Sign",
		inputTypes:  []string{"bytes", "bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeMediumBaseGas, nativeMediumWordGas),
		handler:     nativeEd25519Sign,
	},
	{
		address:     common.CryptoUpgradeEd25519VerifyAddress,
		name:        "Ed25519Verify",
		inputTypes:  []string{"bytes", "bytes", "bytes"},
		outputTypes: []string{"bool"},
		requiredGas: nativeEncodedGas(nativeMediumBaseGas, nativeMediumWordGas),
		handler:     nativeEd25519Verify,
	},
	{
		address:     common.CryptoUpgradePedersenCommitAddress,
		name:        "PedersenCommit",
		inputTypes:  []string{"bytes", "bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeHeavyBaseGas, nativeHeavyWordGas),
		handler:     nativePedersenCommit,
	},
	{
		address:     common.CryptoUpgradePedersenVerifyAddress,
		name:        "PedersenVerify",
		inputTypes:  []string{"bytes", "bytes", "bytes"},
		outputTypes: []string{"bool"},
		requiredGas: nativeEncodedGas(nativeHeavyBaseGas, nativeHeavyWordGas),
		handler:     nativePedersenVerify,
	},
	{
		address:     common.CryptoUpgradeSchnorrPublicAddress,
		name:        "SchnorrPublicKey",
		inputTypes:  []string{"bytes"},
		outputTypes: []string{"bytes"},
		requiredGas: nativeEncodedGas(nativeHeavyBaseGas, nativeHeavyWordGas),
		handler:     nativeSchnorrPublicKey,
	},
	{
		address:     common.CryptoUpgradeSchnorrVerifyAddress,
		name:        "SchnorrVerify",
		inputTypes:  []string{"bytes", "bytes"},
		outputTypes: []string{"bool"},
		requiredGas: nativeEncodedGas(nativeHeavyBaseGas, nativeHeavyWordGas),
		handler:     nativeSchnorrVerify,
	},
})

func mustNativePrecompiles(specs []nativePrecompileSpec) []NativePrecompile {
	out := make([]NativePrecompile, len(specs))
	for i, spec := range specs {
		if spec.address == (common.Address{}) {
			panic("cryptoupgrade native precompile has empty address")
		}
		if spec.name == "" {
			panic("cryptoupgrade native precompile has empty name")
		}
		if spec.requiredGas == nil {
			panic("cryptoupgrade native precompile has nil gas function")
		}
		if spec.handler == nil {
			panic("cryptoupgrade native precompile has nil handler")
		}
		mustABIArguments(spec.inputTypes)
		mustABIArguments(spec.outputTypes)
		out[i] = NativePrecompile{
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
			panic(fmt.Sprintf("invalid cryptoupgrade native ABI type %q: %v", typ, err))
		}
	}
}

// Core包调用的入口
func NativePrecompiles() []NativePrecompile {
	out := make([]NativePrecompile, len(nativePrecompiles))
	copy(out, nativePrecompiles)
	return out
}

func NativePrecompileAddresses() []common.Address {
	out := make([]common.Address, len(nativePrecompiles))
	for i, p := range nativePrecompiles {
		out[i] = p.address
	}
	return out
}

func NativePrecompileByName(name string) (NativePrecompile, bool) {
	for _, p := range nativePrecompiles {
		if p.name == name {
			return p, true
		}
	}
	return NativePrecompile{}, false
}

func (p NativePrecompile) Address() common.Address {
	return p.address
}

func (p NativePrecompile) Name() string {
	return p.name
}

func (p NativePrecompile) InputTypes() []string {
	return append([]string(nil), p.inputTypes...)
}

func (p NativePrecompile) OutputTypes() []string {
	return append([]string(nil), p.outputTypes...)
}

func (p NativePrecompile) RequiredGas(input []byte) uint64 {
	return p.requiredGas(input)
}

func (p NativePrecompile) Run(input []byte) ([]byte, error) {
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

func nativeEncodedGas(base, perWord uint64) func([]byte) uint64 {
	return func(input []byte) uint64 {
		return nativeSaturatingAdd(base, nativeSaturatingMul(perWord, nativeWords(len(input))))
	}
}

func nativePBKDF2Gas(input []byte) uint64 {
	gas := nativeEncodedGas(nativeMediumBaseGas, nativeMediumWordGas)(input)
	args, err := UnpackInput(input, []string{"bytes", "bytes", "uint256", "uint256"})
	if err != nil {
		return nativeSaturatingAdd(nativeMalformedInputGas, nativeSaturatingMul(nativeLightWordGas, nativeWords(len(input))))
	}
	iterations, ok := nativeBoundedInt(args[2], 1, nativePBKDF2MaxIterations)
	if !ok {
		return gas
	}
	keyLength, ok := nativeBoundedInt(args[3], 0, nativePBKDF2MaxKeyLength)
	if !ok || keyLength == 0 {
		return gas
	}
	blocks := nativeWords(keyLength)
	rounds := nativeSaturatingMul(uint64(iterations), blocks)
	return nativeSaturatingAdd(gas, nativeSaturatingMul(rounds, 25))
}

func nativeWords(n int) uint64 {
	return uint64(n+31) / 32
}

func nativeSaturatingAdd(a, b uint64) uint64 {
	if nativeMaxUint64-a < b {
		return nativeMaxUint64
	}
	return a + b
}

func nativeSaturatingMul(a, b uint64) uint64 {
	if a != 0 && b > nativeMaxUint64/a {
		return nativeMaxUint64
	}
	return a * b
}

func nativeBytes(args []interface{}, index int) ([]byte, error) {
	v, ok := args[index].([]byte)
	if !ok {
		return nil, fmt.Errorf("argument %d has type %T, want []byte", index, args[index])
	}
	return v, nil
}

func nativeBig(args []interface{}, index int) (*big.Int, error) {
	v, ok := args[index].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("argument %d has type %T, want *big.Int", index, args[index])
	}
	return v, nil
}

func nativeBoundedInt(arg interface{}, min, max int) (int, bool) {
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

func nativeAdd(args []interface{}) ([]interface{}, error) {
	a, err := nativeBig(args, 0)
	if err != nil {
		return nil, err
	}
	b, err := nativeBig(args, 1)
	if err != nil {
		return nil, err
	}
	sum := new(big.Int).Add(a, b)
	if sum.BitLen() > 256 {
		sum.Mod(sum, nativeUint256Mod())
	}
	return []interface{}{sum}, nil
}

func nativeSha256(args []interface{}) ([]interface{}, error) {
	data, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	return []interface{}{append([]byte(nil), sum[:]...)}, nil
}

func nativeBlake2bSum256(args []interface{}) ([]interface{}, error) {
	data, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	sum := blake2b.Sum256(data)
	return []interface{}{append([]byte(nil), sum[:]...)}, nil
}

func nativeBlake2bSum384(args []interface{}) ([]interface{}, error) {
	data, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	sum := blake2b.Sum384(data)
	return []interface{}{append([]byte(nil), sum[:]...)}, nil
}

func nativeBlake2bSum512(args []interface{}) ([]interface{}, error) {
	data, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	sum := blake2b.Sum512(data)
	return []interface{}{append([]byte(nil), sum[:]...)}, nil
}

func nativeAesCBCEncrypt(args []interface{}) ([]interface{}, error) {
	key, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	iv, err := nativeBytes(args, 1)
	if err != nil {
		return nil, err
	}
	plaintext, err := nativeBytes(args, 2)
	if err != nil {
		return nil, err
	}
	out, err := aesCBCEncrypt(key, iv, plaintext)
	if err != nil {
		return nil, err
	}
	return []interface{}{out}, nil
}

func nativeAesCBCDecrypt(args []interface{}) ([]interface{}, error) {
	key, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	iv, err := nativeBytes(args, 1)
	if err != nil {
		return nil, err
	}
	ciphertext, err := nativeBytes(args, 2)
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

func nativePBKDF2Sha256(args []interface{}) ([]interface{}, error) {
	password, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	salt, err := nativeBytes(args, 1)
	if err != nil {
		return nil, err
	}
	iterations, ok := nativeBoundedInt(args[2], 1, nativePBKDF2MaxIterations)
	if !ok {
		return nil, errors.New("PBKDF2 iterations out of range")
	}
	keyLength, ok := nativeBoundedInt(args[3], 0, nativePBKDF2MaxKeyLength)
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

func nativeDH2048Private(args []interface{}) ([]interface{}, error) {
	seed, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	if len(seed) == 0 {
		return nil, errors.New("DH2048 private seed is empty")
	}
	p := nativeRFC3526Prime()
	max := new(big.Int).Sub(p, big.NewInt(3))
	digest := sha512.Sum512(seed)
	x := new(big.Int).SetBytes(digest[:])
	x.Mod(x, max)
	x.Add(x, big.NewInt(2))
	return []interface{}{nativeFixedBytes(x, nativeRFC3526FieldBytes())}, nil
}

func nativeDH2048Public(args []interface{}) ([]interface{}, error) {
	privateKey, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	p := nativeRFC3526Prime()
	x := nativeRFC3526Scalar(privateKey, p)
	y := new(big.Int).Exp(big.NewInt(2), x, p)
	return []interface{}{nativeFixedBytes(y, nativeRFC3526FieldBytes())}, nil
}

func nativeDH2048Secret(args []interface{}) ([]interface{}, error) {
	privateKey, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	peerPublicKey, err := nativeBytes(args, 1)
	if err != nil {
		return nil, err
	}
	p := nativeRFC3526Prime()
	y := new(big.Int).SetBytes(peerPublicKey)
	if y.Cmp(big.NewInt(1)) <= 0 || y.Cmp(new(big.Int).Sub(p, big.NewInt(1))) >= 0 {
		return nil, errors.New("invalid DH2048 peer public key")
	}
	x := nativeRFC3526Scalar(privateKey, p)
	secret := new(big.Int).Exp(y, x, p)
	return []interface{}{nativeFixedBytes(secret, nativeRFC3526FieldBytes())}, nil
}

func nativeEd25519Keygen(args []interface{}) ([]interface{}, error) {
	seed, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("Ed25519 seed length %d, want %d", len(seed), ed25519.SeedSize)
	}
	return []interface{}{append([]byte(nil), ed25519.NewKeyFromSeed(seed)...)}, nil
}

func nativeEd25519PublicKey(args []interface{}) ([]interface{}, error) {
	privateKey, err := nativeBytes(args, 0)
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

func nativeEd25519Sign(args []interface{}) ([]interface{}, error) {
	privateKey, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	message, err := nativeBytes(args, 1)
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

func nativeEd25519Verify(args []interface{}) ([]interface{}, error) {
	publicKey, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	message, err := nativeBytes(args, 1)
	if err != nil {
		return nil, err
	}
	signature, err := nativeBytes(args, 2)
	if err != nil {
		return nil, err
	}
	if len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize {
		return []interface{}{false}, nil
	}
	return []interface{}{ed25519.Verify(ed25519.PublicKey(publicKey), message, signature)}, nil
}

func nativePedersenCommit(args []interface{}) ([]interface{}, error) {
	message, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	blinding, err := nativeBytes(args, 1)
	if err != nil {
		return nil, err
	}
	return []interface{}{pedersenCommit(message, blinding)}, nil
}

func nativePedersenVerify(args []interface{}) ([]interface{}, error) {
	message, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	blinding, err := nativeBytes(args, 1)
	if err != nil {
		return nil, err
	}
	commitment, err := nativeBytes(args, 2)
	if err != nil {
		return nil, err
	}
	expected := pedersenCommit(message, blinding)
	return []interface{}{bytes.Equal(expected, commitment)}, nil
}

func nativeSchnorrPublicKey(args []interface{}) ([]interface{}, error) {
	secret, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	p, q := nativeRFC3526Subgroup()
	x := nativeRFC3526SubgroupScalar(secret, q)
	y := new(big.Int).Exp(big.NewInt(4), x, p)
	return []interface{}{nativeFixedBytes(y, nativeRFC3526FieldBytes())}, nil
}

func nativeSchnorrVerify(args []interface{}) ([]interface{}, error) {
	message, err := nativeBytes(args, 0)
	if err != nil {
		return nil, err
	}
	proof, err := nativeBytes(args, 1)
	if err != nil {
		return nil, err
	}
	return []interface{}{schnorrVerify(message, proof)}, nil
}

func nativeUint256Mod() *big.Int {
	return new(big.Int).Lsh(big.NewInt(1), 256)
}

const nativeRFC3526PrimeHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3" +
	"404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BF" +
	"B5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C6" +
	"2F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28" +
	"FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"

func nativeRFC3526Prime() *big.Int {
	p, _ := new(big.Int).SetString(nativeRFC3526PrimeHex, 16)
	return p
}

func nativeRFC3526Subgroup() (*big.Int, *big.Int) {
	p := nativeRFC3526Prime()
	q := new(big.Int).Sub(p, big.NewInt(1))
	q.Rsh(q, 1)
	return p, q
}

func nativeRFC3526FieldBytes() int {
	return (len(nativeRFC3526PrimeHex) + 1) / 2
}

func nativeRFC3526Scalar(data []byte, p *big.Int) *big.Int {
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

func nativeRFC3526SubgroupScalar(data []byte, q *big.Int) *big.Int {
	x := new(big.Int).SetBytes(data)
	x.Mod(x, q)
	if x.Sign() == 0 {
		x.SetInt64(1)
	}
	return x
}

func nativeFixedBytes(x *big.Int, size int) []byte {
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
	p, q := nativeRFC3526Subgroup()
	g := big.NewInt(4)
	h := pedersenH(p)
	m := nativeRFC3526ModScalar(message, q)
	r := nativeRFC3526ModScalar(blinding, q)

	gm := new(big.Int).Exp(g, m, p)
	hr := new(big.Int).Exp(h, r, p)
	commitment := new(big.Int).Mul(gm, hr)
	commitment.Mod(commitment, p)
	return nativeFixedBytes(commitment, nativeRFC3526FieldBytes())
}

func nativeRFC3526ModScalar(data []byte, q *big.Int) *big.Int {
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
	size := nativeRFC3526FieldBytes()
	if len(proof) != 3*size {
		return false
	}
	p, q := nativeRFC3526Subgroup()
	y := new(big.Int).SetBytes(proof[:size])
	t := new(big.Int).SetBytes(proof[size : 2*size])
	s := new(big.Int).SetBytes(proof[2*size:])
	if !nativeRFC3526GroupElement(y, p) || !nativeRFC3526GroupElement(t, p) || s.Cmp(q) >= 0 {
		return false
	}

	g := big.NewInt(4)
	c := schnorrChallenge(q, y, t, message)
	left := new(big.Int).Exp(g, s, p)
	yc := new(big.Int).Exp(y, c, p)
	right := new(big.Int).Mul(t, yc)
	right.Mod(right, p)
	return bytes.Equal(nativeFixedBytes(left, size), nativeFixedBytes(right, size))
}

func schnorrChallenge(q, y, t *big.Int, message []byte) *big.Int {
	h := sha256.New()
	h.Write([]byte("cryptoupgrade-schnorr"))
	h.Write(nativeFixedBytes(y, nativeRFC3526FieldBytes()))
	h.Write(nativeFixedBytes(t, nativeRFC3526FieldBytes()))
	h.Write(message)
	c := new(big.Int).SetBytes(h.Sum(nil))
	c.Mod(c, q)
	return c
}

func nativeRFC3526GroupElement(x, p *big.Int) bool {
	return x.Cmp(big.NewInt(1)) > 0 && x.Cmp(new(big.Int).Sub(p, big.NewInt(1))) < 0
}
