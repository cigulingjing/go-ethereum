// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "./BigMod.sol";

contract SchnorrProof is BigMod {
    function SchnorrVerify(bytes calldata message, bytes calldata proof) external view returns (bool) {
        if (proof.length != 768) {
            return false;
        }
        bytes memory p = dh2048Prime();
        bytes memory q = dh2048Q();
        bytes memory y = sliceCalldata(proof, 0, 256);
        bytes memory t = sliceCalldata(proof, 256, 256);
        bytes memory s = sliceCalldata(proof, 512, 256);
        if (!groupElement(y) || !groupElement(t) || compareBytes(s, q) >= 0) {
            return false;
        }

        bytes32 c = sha256(abi.encodePacked("cryptoupgrade-schnorr", y, t, message));
        bytes memory left = modExp(hex"04", s, p);
        bytes memory yc = modExp(y, abi.encodePacked(c), p);
        bytes memory right = mulModBytes(t, yc, p);
        return keccak256(left) == keccak256(right);
    }
}
