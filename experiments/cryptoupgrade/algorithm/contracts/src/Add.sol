// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract AddContract {
    function Add(int256 a, int256 b) external pure returns (int256) {
        return a + b;
    }
}
