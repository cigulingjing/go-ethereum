// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract Pbkdf2Sha256Contract {
    uint256 private constant PBKDF2_MAX_ITERATIONS = 10000;
    uint256 private constant PBKDF2_MAX_KEY_LENGTH = 1024;

    function Pbkdf2Sha256(
        bytes calldata password,
        bytes calldata salt,
        uint256 iterations,
        uint256 keyLength
    ) external pure returns (bytes memory) {
        if (iterations == 0 || iterations > PBKDF2_MAX_ITERATIONS || keyLength > PBKDF2_MAX_KEY_LENGTH) {
            return "";
        }
        if (keyLength == 0) {
            return "";
        }

        bytes memory key = password;
        uint256 blocks = (keyLength + 31) / 32;
        bytes memory derived = new bytes(blocks * 32);
        for (uint256 blockIndex = 1; blockIndex <= blocks; blockIndex++) {
            bytes32 u = hmacSha256(key, abi.encodePacked(salt, uint32(blockIndex)));
            bytes32 t = u;
            for (uint256 round = 2; round <= iterations; round++) {
                u = hmacSha256(key, abi.encodePacked(u));
                t ^= u;
            }
            writeBytes32(derived, (blockIndex - 1) * 32, t);
        }
        return truncate(derived, keyLength);
    }

    function hmacSha256(bytes memory key, bytes memory data) private pure returns (bytes32) {
        if (key.length > 64) {
            key = abi.encodePacked(sha256(key));
        }
        bytes memory ipad = new bytes(64);
        bytes memory opad = new bytes(64);
        for (uint256 i = 0; i < 64; i++) {
            uint8 keyByte = i < key.length ? uint8(key[i]) : 0;
            ipad[i] = bytes1(keyByte ^ 0x36);
            opad[i] = bytes1(keyByte ^ 0x5c);
        }
        return sha256(bytes.concat(opad, abi.encodePacked(sha256(bytes.concat(ipad, data)))));
    }

    function writeBytes32(bytes memory out, uint256 offset, bytes32 value) private pure {
        for (uint256 i = 0; i < 32 && offset + i < out.length; i++) {
            out[offset + i] = value[i];
        }
    }

    function truncate(bytes memory data, uint256 length) private pure returns (bytes memory) {
        bytes memory out = new bytes(length);
        for (uint256 i = 0; i < length; i++) {
            out[i] = data[i];
        }
        return out;
    }
}
