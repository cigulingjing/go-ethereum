package common

var (
	CodeStorageAddress   = BytesToAddress([]byte{67}) // 0000000000000000000000000000000000000043
	MutiVoucherAddress   = BytesToAddress([]byte{68}) // 0000000000000000000000000000000000000044
	CoinBaseAddress      = BytesToAddress([]byte{69}) // 0000000000000000000000000000000000000045
	Blake2bSum256Address = BytesToAddress([]byte{70}) // 0000000000000000000000000000000000000046

	CryptoUpgradeAddAddress            = BytesToAddress([]byte{0x47}) // 0000000000000000000000000000000000000047
	CryptoUpgradeSha256Address         = BytesToAddress([]byte{0x48}) // 0000000000000000000000000000000000000048
	CryptoUpgradeBlake2bSum256Address  = BytesToAddress([]byte{0x49}) // 0000000000000000000000000000000000000049
	CryptoUpgradeBlake2bSum384Address  = BytesToAddress([]byte{0x4a}) // 000000000000000000000000000000000000004a
	CryptoUpgradeBlake2bSum512Address  = BytesToAddress([]byte{0x4b}) // 000000000000000000000000000000000000004b
	CryptoUpgradeAesCBCEncryptAddress  = BytesToAddress([]byte{0x4c}) // 000000000000000000000000000000000000004c
	CryptoUpgradeAesCBCDecryptAddress  = BytesToAddress([]byte{0x4d}) // 000000000000000000000000000000000000004d
	CryptoUpgradePbkdf2Sha256Address   = BytesToAddress([]byte{0x4e}) // 000000000000000000000000000000000000004e
	CryptoUpgradeDh2048PrivateAddress  = BytesToAddress([]byte{0x4f}) // 000000000000000000000000000000000000004f
	CryptoUpgradeDh2048PublicAddress   = BytesToAddress([]byte{0x50}) // 0000000000000000000000000000000000000050
	CryptoUpgradeDh2048SecretAddress   = BytesToAddress([]byte{0x51}) // 0000000000000000000000000000000000000051
	CryptoUpgradeEd25519KeygenAddress  = BytesToAddress([]byte{0x52}) // 0000000000000000000000000000000000000052
	CryptoUpgradeEd25519PublicAddress  = BytesToAddress([]byte{0x53}) // 0000000000000000000000000000000000000053
	CryptoUpgradeEd25519SignAddress    = BytesToAddress([]byte{0x54}) // 0000000000000000000000000000000000000054
	CryptoUpgradeEd25519VerifyAddress  = BytesToAddress([]byte{0x55}) // 0000000000000000000000000000000000000055
	CryptoUpgradePedersenCommitAddress = BytesToAddress([]byte{0x56}) // 0000000000000000000000000000000000000056
	CryptoUpgradePedersenVerifyAddress = BytesToAddress([]byte{0x57}) // 0000000000000000000000000000000000000057
	CryptoUpgradeSchnorrPublicAddress  = BytesToAddress([]byte{0x58}) // 0000000000000000000000000000000000000058
	CryptoUpgradeSchnorrVerifyAddress  = BytesToAddress([]byte{0x59}) // 0000000000000000000000000000000000000059
	CryptoUpgradePolynomialMulAddress  = BytesToAddress([]byte{0x5a}) // 000000000000000000000000000000000000005a
)

var CoinbaseABI_json = `[
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "string",
        "name": "source",
        "type": "string"
      },
      {
        "indexed": true,
        "internalType": "string",
        "name": "rewardType",
        "type": "string"
      },
      {
        "indexed": true,
        "internalType": "uint256",
        "name": "timestamp",
        "type": "uint256"
      },
      {
        "indexed": false,
        "internalType": "address[]",
        "name": "selectedAddresses",
        "type": "address[]"
      },
      {
        "indexed": false,
        "internalType": "uint256[]",
        "name": "rewards",
        "type": "uint256[]"
      }
    ],
    "name": "CoinbaseAdded",
    "type": "event"
  }
]`

var MutivoucherABI_json = `[]`

var CodeStorageABI_json = `[
  {
    "anonymous": false,
    "inputs": [
      {"indexed": false, "internalType": "string", "name": "name", "type": "string"}
    ],
    "name": "codeUploaded",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": false, "internalType": "string", "name": "name", "type": "string"},
      {"indexed": false, "internalType": "uint64", "name": "version", "type": "uint64"},
      {"indexed": false, "internalType": "uint64", "name": "activationBlock", "type": "uint64"}
    ],
    "name": "codeVersionUploaded",
    "type": "event"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"}
    ],
    "name": "getCode",
    "outputs": [
      {"internalType": "string", "name": "", "type": "string"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"}
    ],
    "name": "getGas",
    "outputs": [
      {"internalType": "uint64", "name": "", "type": "uint64"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"},
      {"internalType": "bytes", "name": "input", "type": "bytes"}
    ],
    "name": "callFunc",
    "outputs": [
      {"internalType": "bytes", "name": "", "type": "bytes"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"}
    ],
    "name": "getInfo",
    "outputs": [
      {"internalType": "string", "name": "code", "type": "string"},
      {"internalType": "uint64", "name": "gas", "type": "uint64"},
      {"internalType": "string", "name": "itype", "type": "string"},
      {"internalType": "string", "name": "otype", "type": "string"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"},
      {"internalType": "uint64", "name": "version", "type": "uint64"}
    ],
    "name": "getVersionInfo",
    "outputs": [
      {"internalType": "string", "name": "code", "type": "string"},
      {"internalType": "uint64", "name": "gas", "type": "uint64"},
      {"internalType": "string", "name": "itype", "type": "string"},
      {"internalType": "string", "name": "otype", "type": "string"},
      {"internalType": "uint64", "name": "storedVersion", "type": "uint64"},
      {"internalType": "uint64", "name": "activationBlock", "type": "uint64"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"}
    ],
    "name": "getActiveVersion",
    "outputs": [
      {"internalType": "uint64", "name": "version", "type": "uint64"},
      {"internalType": "uint64", "name": "activationBlock", "type": "uint64"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"}
    ],
    "name": "getActiveInfo",
    "outputs": [
      {"internalType": "string", "name": "code", "type": "string"},
      {"internalType": "uint64", "name": "gas", "type": "uint64"},
      {"internalType": "string", "name": "itype", "type": "string"},
      {"internalType": "string", "name": "otype", "type": "string"},
      {"internalType": "uint64", "name": "version", "type": "uint64"},
      {"internalType": "uint64", "name": "activationBlock", "type": "uint64"}
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"},
      {"internalType": "uint64", "name": "_gas", "type": "uint64"}
    ],
    "name": "updataGas",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"},
      {"internalType": "string", "name": "code", "type": "string"},
      {"internalType": "uint64", "name": "gas", "type": "uint64"},
      {"internalType": "string", "name": "itype", "type": "string"},
      {"internalType": "string", "name": "otype", "type": "string"}
    ],
    "name": "uploadCode",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"},
      {"internalType": "uint64", "name": "version", "type": "uint64"},
      {"internalType": "string", "name": "code", "type": "string"},
      {"internalType": "uint64", "name": "gas", "type": "uint64"},
      {"internalType": "string", "name": "itype", "type": "string"},
      {"internalType": "string", "name": "otype", "type": "string"},
      {"internalType": "uint64", "name": "activationBlock", "type": "uint64"}
    ],
    "name": "uploadCodeVersion",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType": "string", "name": "name", "type": "string"},
      {"internalType": "uint64", "name": "version", "type": "uint64"},
      {"internalType": "string", "name": "code", "type": "string"},
      {"internalType": "uint64", "name": "gas", "type": "uint64"},
      {"internalType": "string", "name": "itype", "type": "string"},
      {"internalType": "string", "name": "otype", "type": "string"}
    ],
    "name": "uploadCodeImmediate",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]`
