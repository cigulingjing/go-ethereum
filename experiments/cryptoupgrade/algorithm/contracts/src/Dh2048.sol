// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "./BigMod.sol";

contract Dh2048 is BigMod {
    function Dh2048Secret(bytes calldata privateKey, bytes calldata peerPublicKey) external view returns (bytes memory) {
        bytes memory p = dh2048Prime();
        bytes memory x = privateKey;
        bytes memory y = peerPublicKey;
        if (!validScalar(x) || !groupElement(y)) {
            return "";
        }
        return modExp(y, x, p);
    }
}
