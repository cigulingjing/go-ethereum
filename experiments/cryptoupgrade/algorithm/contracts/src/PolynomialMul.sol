// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract PolynomialMulContract {
    uint256 internal constant MAX_POLY_LENGTH = 256;

    function PolynomialMul(
        uint256[] calldata left,
        uint256[] calldata right,
        uint256 modulus
    ) external pure returns (uint256[] memory result) {
        require(left.length > 0, "empty left");
        require(right.length > 0, "empty right");
        require(left.length <= MAX_POLY_LENGTH, "left too large");
        require(right.length <= MAX_POLY_LENGTH, "right too large");
        require(modulus > 0, "zero modulus");

        result = new uint256[](left.length + right.length - 1);
        for (uint256 i = 0; i < left.length; i++) {
            uint256 l = left[i] % modulus;
            for (uint256 j = 0; j < right.length; j++) {
                result[i + j] = addmod(result[i + j], mulmod(l, right[j] % modulus, modulus), modulus);
            }
        }
    }
}
