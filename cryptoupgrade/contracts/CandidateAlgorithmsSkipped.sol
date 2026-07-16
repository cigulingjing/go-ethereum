// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract CandidateAlgorithmsSkipped {
    function SkippedAlgorithms() external pure returns (string memory) {
        return "RandomBytes, AES-CBC, Ed25519 keygen/sign/verify, and ShamirSplit/ShamirRecover are skipped for Solidity benchmarking: secure randomness, AES/Ed25519 primitives, or compact 521-bit field arithmetic are not practical without new dependencies.";
    }
}
