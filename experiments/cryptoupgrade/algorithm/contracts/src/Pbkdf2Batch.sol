// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

interface IPbkdf2Reference {
    function Pbkdf2Sha256(bytes calldata password, bytes calldata salt, uint256 iterations, uint256 keyLength)
        external pure returns (bytes memory);
}

interface ICodeStoragePbkdf2 {
    function callFunc(string calldata name, bytes calldata input) external returns (bytes memory);
}

contract Pbkdf2Batch {
    address internal constant CODE_STORAGE = 0x0000000000000000000000000000000000000043;
    IPbkdf2Reference public immutable pbkdf2Contract;
    bytes32 public lastDigest;

    event BatchComplete(uint256 calls, bytes32 digest);

    constructor(address referenceAddress) {
        pbkdf2Contract = IPbkdf2Reference(referenceAddress);
    }

    function solidityCalls(uint256 calls, uint256 iterations, bytes32 expectedOutputHash) external returns (bytes32 digest) {
        require(calls > 0 && calls <= 16);
        for (uint256 index = 0; index < calls; index++) {
            bytes memory derived = pbkdf2Contract.Pbkdf2Sha256(bytes("password"), bytes("salt"), iterations, 32);
            require(keccak256(derived) == expectedOutputHash);
            digest = keccak256(abi.encodePacked(digest, derived));
        }
        lastDigest = digest;
        emit BatchComplete(calls, digest);
    }

    function evocryptCalls(uint256 calls, uint256 iterations, bytes32 expectedOutputHash) external returns (bytes32 digest) {
        require(calls > 0 && calls <= 16);
        for (uint256 index = 0; index < calls; index++) {
            bytes memory derived = ICodeStoragePbkdf2(CODE_STORAGE).callFunc(
                "Pbkdf2Sha256", abi.encode(bytes("password"), bytes("salt"), iterations, uint256(32))
            );
            require(keccak256(derived) == expectedOutputHash);
            digest = keccak256(abi.encodePacked(digest, derived));
        }
        lastDigest = digest;
        emit BatchComplete(calls, digest);
    }
}
