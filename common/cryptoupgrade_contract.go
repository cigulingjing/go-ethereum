package common

var (
	CodeStorageAddress   = BytesToAddress([]byte{67}) // 0000000000000000000000000000000000000043
	MutiVoucherAddress   = BytesToAddress([]byte{68}) // 0000000000000000000000000000000000000044
	CoinBaseAddress      = BytesToAddress([]byte{69}) // 0000000000000000000000000000000000000045
	Blake2bSum256Address = BytesToAddress([]byte{70}) // 0000000000000000000000000000000000000046
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
      {
        "indexed": false,
        "internalType": "string",
        "name": "name",
        "type": "string"
      }
    ],
    "name": "codeUploaded",
    "type": "event"
  },
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "name",
        "type": "string"
      }
    ],
    "name": "getCode",
    "outputs": [
      {
        "internalType": "string",
        "name": "",
        "type": "string"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "name",
        "type": "string"
      }
    ],
    "name": "getGas",
    "outputs": [
      {
        "internalType": "uint64",
        "name": "",
        "type": "uint64"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "name",
        "type": "string"
      },
      {
        "internalType": "bytes",
        "name": "input",
        "type": "bytes"
      }
    ],
    "name": "callFunc",
    "outputs": [
      {
        "internalType": "bytes",
        "name": "",
        "type": "bytes"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "name",
        "type": "string"
      }
    ],
    "name": "getInfo",
    "outputs": [
      {
        "internalType": "string",
        "name": "code",
        "type": "string"
      },
      {
        "internalType": "uint64",
        "name": "gas",
        "type": "uint64"
      },
      {
        "internalType": "string",
        "name": "itype",
        "type": "string"
      },
      {
        "internalType": "string",
        "name": "otype",
        "type": "string"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "name",
        "type": "string"
      },
      {
        "internalType": "uint64",
        "name": "_gas",
        "type": "uint64"
      }
    ],
    "name": "updataGas",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "string",
        "name": "name",
        "type": "string"
      },
      {
        "internalType": "string",
        "name": "code",
        "type": "string"
      },
      {
        "internalType": "uint64",
        "name": "gas",
        "type": "uint64"
      },
      {
        "internalType": "string",
        "name": "itype",
        "type": "string"
      },
      {
        "internalType": "string",
        "name": "otype",
        "type": "string"
      }
    ],
    "name": "uploadCode",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]`
