// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "./CandidateAlgorithmsBigMod.sol";

contract PedersenCommitSolidity is CandidateAlgorithmsBigMod {
    function PedersenCommit(bytes calldata message, bytes calldata blinding) external view returns (bytes memory) {
        return pedersenCommit(message, blinding);
    }

    function PedersenVerify(
        bytes calldata message,
        bytes calldata blinding,
        bytes calldata commitment
    ) external view returns (bool) {
        if (commitment.length != 256) {
            return false;
        }
        bytes memory expected = pedersenCommit(message, blinding);
        return keccak256(expected) == keccak256(commitment);
    }

    function pedersenCommit(bytes calldata message, bytes calldata blinding) private view returns (bytes memory) {
        bytes memory p = dh2048Prime();
        bytes memory h = pedersenH(p);
        bytes memory gm = modExp(hex"04", message, p);
        bytes memory hr = modExp(h, blinding, p);
        return mulModBytes(gm, hr, p);
    }
}
