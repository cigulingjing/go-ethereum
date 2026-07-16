// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "./CandidateAlgorithmsBigMod.sol";

contract Dh2048Solidity is CandidateAlgorithmsBigMod {
    function Dh2048Public(bytes calldata privateKey) external view returns (bytes memory) {
        bytes memory p = dh2048Prime();
        bytes memory x = privateKey;
        require(validScalar(x), "invalid private key");
        return modExp(hex"02", x, p);
    }

    function Dh2048Secret(bytes calldata privateKey, bytes calldata peerPublicKey) external view returns (bytes memory) {
        bytes memory p = dh2048Prime();
        bytes memory x = privateKey;
        bytes memory y = peerPublicKey;
        require(validScalar(x), "invalid private key");
        require(groupElement(y), "invalid peer key");
        return modExp(y, x, p);
    }

    function SkippedAlgorithms() external pure returns (string memory) {
        return "Dh2048Private is skipped: Solidity cannot generate secure private randomness.";
    }
}
