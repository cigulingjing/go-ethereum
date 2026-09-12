// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract Sha256Contract {
    function Sha256(bytes calldata data) external pure returns (bytes memory) {
        return abi.encodePacked(sha256(data));
    }
}
